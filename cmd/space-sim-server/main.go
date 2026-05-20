// Command space-sim-server is the headless Space Sim server.
// It runs the physics simulation and streams WorldSnapshots to remote clients
// via ConnectRPC (WorldService.StreamSnapshot). No Raylib or GUI is linked.
//
// Usage:
//
//	space-sim-server [flags]
//
// Flags:
//
//	--addr          TCP address to listen on (default ":8080")
//	--system-config path to system JSON (default: data/systems/solar_system.json)
//	--sim-speed     simulated seconds per real second (default 3600)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	grpcserver "github.com/digital-michael/space_sim/internal/transport/grpc"
	world "github.com/digital-michael/space_sim/internal/sim/world"
)

// snapshotHz is the rate at which snapshots are sampled and pushed to clients.
const snapshotHz = 30

func main() {
	addr := flag.String("addr", ":8080", "TCP address to listen on")
	systemConfig := flag.String("system-config", "", "path to system JSON (empty = solar system)")
	simSpeed := flag.Float64("sim-speed", 3600, "simulated seconds per real second")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Build world ───────────────────────────────────────────────────────
	w, err := world.NewWorld(*simSpeed, *systemConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading world: %v\n", err)
		os.Exit(1)
	}

	// ── Build handlers ────────────────────────────────────────────────────
	worldHandler := grpcserver.NewWorldHandler()
	simHandler := grpcserver.NewSimulationHandler(func() *world.World { return w })

	// ── Register only the services meaningful for a headless server ───────
	// Window / Camera / Navigation / etc. are render-client concerns and are
	// intentionally omitted here to prevent nil-interface panics.
	mux := http.NewServeMux()

	simPath, simSvc := spacesimv1connect.NewSimulationServiceHandler(simHandler)
	mux.Handle(simPath, simSvc)

	worldPath, worldSvc := spacesimv1connect.NewWorldServiceHandler(worldHandler)
	mux.Handle(worldPath, worldSvc)

	httpSrv := &http.Server{
		Addr:        *addr,
		Handler:     mux,
		IdleTimeout: 60 * time.Second,
	}

	// ── Start simulation ──────────────────────────────────────────────────
	simCtx, simCancel := context.WithCancel(ctx)
	defer simCancel()
	go w.Start(simCtx)

	// ── Snapshot-push goroutine ───────────────────────────────────────────
	// Samples LatestSnapshot at snapshotHz and delivers it to all connected
	// streaming clients via WorldHandler.Receive.
	go func() {
		ticker := time.NewTicker(time.Second / snapshotHz)
		defer ticker.Stop()
		for {
			select {
			case <-simCtx.Done():
				return
			case <-ticker.C:
				snap := w.LatestSnapshot()
				worldHandler.Receive(snap)
			}
		}
	}()

	// ── Start HTTP/gRPC server ────────────────────────────────────────────
	srvDone := make(chan error, 1)
	go func() {
		log.Printf("space-sim-server listening on %s", *addr)
		srvDone <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
		simCancel()
		<-srvDone
	case err := <-srvDone:
		if err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}
}
