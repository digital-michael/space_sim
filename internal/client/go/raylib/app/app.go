package app

import (
	"context"
	"fmt"
	"log"
	"os"

	render "github.com/digital-michael/space_sim/internal/client/go/raylib/ui/render"
)

// App owns the Space Sim application's runtime orchestration.
type App struct {
	cfg      Config
	runtime  *RuntimeContext
	renderer *render.Renderer
}

// New constructs the application from validated configuration.
func New(cfg Config) (*App, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &App{
		cfg:      cfg,
		runtime:  NewRuntimeContext(cfg.AppConfig),
		renderer: render.New(),
	}, nil
}

// Run executes the application on the current thread.
func (a *App) Run(ctx context.Context) error {
	if a.cfg.Debug {
		logFile, err := os.OpenFile("performance_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Warning: Could not open log file: %v\n", err)
		} else {
			defer logFile.Close()
			log.SetOutput(logFile)
		}
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("=== Application Starting ===")
	log.Printf("Performance mode: %v\n", a.cfg.PerformanceMode)

	a.initWindow()
	defer a.closeWindow()
	a.syncRenderState()

	session, err := a.newRuntimeSession(a.cfg.SystemConfig)
	if err != nil {
		return err
	}

	if a.cfg.PerformanceMode {
		simCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		defer session.sim.Stop()
		go session.sim.Start(simCtx)

		log.Printf("Entering performance test mode (profile=%s, threads=%d, locking=%v)", a.cfg.Profile, a.cfg.Threads, !a.cfg.NoLocking)
		a.runPerformanceTest(session.sim, session.cameraState, session.inputState, a.cfg.Profile, a.cfg.Threads, a.cfg.NoLocking)
		log.Println("Performance test returned normally")
		return nil
	}

	return a.runInteractive(ctx, session)
}
