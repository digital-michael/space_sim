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
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	grpcserver "github.com/digital-michael/space_sim/internal/transport/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── gRPC server ───────────────────────────────────────────────────────
	// Handlers are stubs until Phase 6c wires them to the event queue and
	// runtime environment.
	simHandler := grpcserver.NewSimulationHandler()
	worldHandler := grpcserver.NewWorldHandler()
	srv := grpcserver.New(grpcserver.DefaultServerConfig(), simHandler, worldHandler)

	srvDone := make(chan error, 1)
	go func() {
		log.Printf("gRPC server listening on %s", srv.Addr())
		srvDone <- srv.Start(ctx)
	}()

	// ── Raylib application ────────────────────────────────────────────────
	// The Raylib client accesses simulation state directly (in-process).
	// TODO (Phase 6c): register a broadcaster subscriber that forwards
	// WorldSnapshot frames to the WorldHandler for streaming clients.
	appConfigPath := rayapp.DefaultAppConfigPath
	appConfig, err := rayapp.LoadAppConfig(appConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading app config %s: %v\n", appConfigPath, err)
		os.Exit(1)
	}

	cfg := rayapp.Config{
		AppConfigPath: appConfigPath,
		AppConfig:     appConfig,
	}

	application, err := rayapp.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating app: %v\n", err)
		os.Exit(1)
	}

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

