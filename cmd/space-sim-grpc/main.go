// Command space-sim-grpc is the Space Sim binary with an embedded ConnectRPC
// server (Option A). The Raylib client runs in the main process and accesses
// simulation state directly. The gRPC server runs in a background goroutine
// and exposes simulation state to external clients (future JS browser client,
// additional Go clients) on a TCP port.
//
// Option B (future): this binary becomes the client only; the server is a
// separate binary. Player identification and registration will be handled on
// gRPC connection as part of that split.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	grpcserver "github.com/digital-michael/space_sim/internal/transport/grpc"
)

func main() {
	renderScaleUsage := `multiply the display size to set the render resolution (forces fixed mode).
Recommended:
  1.0  — matches your display exactly (same as default native mode)
  2.0  — 2× super-sample; doubles linear resolution, 4× the pixels (ideal for recording)
  4.0  — 4× super-sample; very high quality, significant GPU/readback cost
Reasonable range: 0.25 (lightweight) – 4.0 (high quality)
Mutually exclusive with --render-size.`

	renderSizeUsage := `set an explicit render resolution as WIDTHxHEIGHT (forces fixed mode).
Example: --render-size=3840x2160
Useful when you need a specific output resolution regardless of display size.
Mutually exclusive with --render-scale.`

	renderScale := flag.Float64("render-scale", 0, renderScaleUsage)
	renderSize := flag.String("render-size", "", renderSizeUsage)
	noMSAA := flag.Bool("no-msaa", false, "disable 4× MSAA anti-aliasing (enabled by default)")
	noTextures := flag.Bool("no-textures", false, "disable diffuse texture rendering; use solid colors (enabled by default)")
	reset := flag.Bool("reset", false, "restore app.json to factory defaults and exit")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Load app config ───────────────────────────────────────────────────
	appConfigPath := rayapp.DefaultAppConfigPath
	appConfig, err := rayapp.LoadAppConfig(appConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading app config %s: %v\n", appConfigPath, err)
		os.Exit(1)
	}

	// ── Handle --reset ────────────────────────────────────────────────────
	if *reset {
		if err := rayapp.ResetAppConfig(appConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "error resetting config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Reset %s to factory defaults\n", appConfigPath)
		os.Exit(0)
	}

	cfg := rayapp.Config{
		AppConfigPath: appConfigPath,
		AppConfig:     appConfig,
		RenderScale:   *renderScale,
		RenderSize:    *renderSize,
		NoMSAA:        *noMSAA,
		NoTextures:    *noTextures,
	}

	// ── Build application (creates world internally) ───────────────────────
	application, err := rayapp.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating app: %v\n", err)
		os.Exit(1)
	}

	// ── gRPC server ───────────────────────────────────────────────────────
	// WorldHandler implements protocol.Subscriber — register it directly so
	// the interactive loop delivers WorldSnapshot frames to all streaming clients.
	worldHandler := grpcserver.NewWorldHandler()
	application.RegisterSubscriber(worldHandler)

	// application.World is called on every RPC — it resolves to nil until
	// Run() has loaded a session, at which point all SimulationService RPCs
	// become fully functional.
	simHandler := grpcserver.NewSimulationHandler(application.World)

	// All new service handlers use application.SendCmd as the main-thread gate.
	systemHandler := grpcserver.NewSystemHandler(application.SendCmd)
	windowHandler := grpcserver.NewWindowHandler(application.SendCmd)
	cameraHandler := grpcserver.NewCameraHandler(application.SendCmd)
	navHandler := grpcserver.NewNavigationHandler(application.SendCmd)
	perfHandler := grpcserver.NewPerformanceHandler(application.SendCmd)
	shutdownHandler := grpcserver.NewShutdownHandler(stop)
	recordingHandler := grpcserver.NewRecordingHandler(application.SendCmd)

	srv := grpcserver.New(grpcserver.DefaultServerConfig(), grpcserver.Handlers{
		Simulation:  simHandler,
		World:       worldHandler,
		System:      systemHandler,
		Window:      windowHandler,
		Camera:      cameraHandler,
		Navigation:  navHandler,
		Performance: perfHandler,
		Shutdown:    shutdownHandler,
		Recording:   recordingHandler,
	})

	srvDone := make(chan error, 1)
	go func() {
		log.Printf("gRPC server listening on %s", srv.Addr())
		srvDone <- srv.Start(ctx)
	}()

	// ── Run Raylib app (blocks until window closed or ctx cancelled) ───────
	appErr := application.Run(ctx)

	// Cancel context so the gRPC server shuts down, then wait for it.
	stop()
	if srvErr := <-srvDone; srvErr != nil {
		log.Printf("gRPC server shutdown error: %v", srvErr)
	}

	if appErr != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", appErr)
		os.Exit(1)
	}
}
