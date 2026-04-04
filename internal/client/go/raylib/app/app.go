package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	render "github.com/digital-michael/space_sim/internal/client/go/raylib/ui/render"
	"github.com/digital-michael/space_sim/internal/protocol"
	worldpkg "github.com/digital-michael/space_sim/internal/sim/world"
)

// appCmdBufSize is the number of AppCmds that can be queued without blocking.
const appCmdBufSize = 32

// App owns the Space Sim application's runtime orchestration.
type App struct {
	cfg         Config
	runtime     *RuntimeContext
	renderer    *render.Renderer
	broadcaster *protocol.Broadcaster
	worldPtr    atomic.Pointer[worldpkg.World]

	// cmdCh is the main-thread command gate. gRPC handler goroutines send
	// AppCmds here; the interactive loop drains it each frame (non-blocking).
	cmdCh chan AppCmd
}

// New constructs the application from validated configuration.
func New(cfg Config) (*App, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &App{
		cfg:         cfg,
		runtime:     NewRuntimeContext(cfg.AppConfig),
		renderer:    render.New(),
		broadcaster: &protocol.Broadcaster{},
		cmdCh:       make(chan AppCmd, appCmdBufSize),
	}, nil
}

// SendCmd enqueues an AppCmd for execution on the OS main thread.
// Returns true if the command was accepted, false if the channel was full.
// The caller must not block on full; it should treat a false return as a
// transient back-pressure signal and retry or surface an error.
func (a *App) SendCmd(cmd AppCmd) bool {
	select {
	case a.cmdCh <- cmd:
		return true
	default:
		log.Printf("AppCmd channel full — dropping %T", cmd)
		return false
	}
}

// RegisterSubscriber adds s to the broadcast list. Every WorldSnapshot
// produced by the interactive loop will be delivered to s.Receive.
func (a *App) RegisterSubscriber(s protocol.Subscriber) {
	a.broadcaster.Register(s)
}

// World returns the active simulation world, or nil if the session has not
// yet been initialised (i.e. Run has not yet loaded a system). Safe to call
// from any goroutine; backed by an atomic pointer so no lock is needed.
func (a *App) World() *worldpkg.World {
	return a.worldPtr.Load()
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
	a.worldPtr.Store(session.sim)

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
