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
//	--addr          TCP address to listen on (default ":9090")
//	--system-config path to system JSON (default: data/systems/solar_system.json)
//	--sim-speed     simulated seconds per real second (default 3600)
//	--data-dir      content directory served to remote clients (default "data")
//	--idle          behaviour with no clients attached: run | exit (default "run")
//	--idle-grace    zero-client tolerance before --idle=exit acts (default 5s)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/digital-michael/space_sim/internal/server/assets"
	"github.com/digital-michael/space_sim/internal/server/session"
	"github.com/digital-michael/space_sim/internal/sim/engine"
	world "github.com/digital-michael/space_sim/internal/sim/world"
	grpcserver "github.com/digital-michael/space_sim/internal/transport/grpc"
)

// snapshotHz is the rate at which snapshots are sampled and pushed to clients.
const snapshotHz = 30

func main() {
	addr := flag.String("addr", ":9090", "TCP address to listen on")
	systemConfig := flag.String("system-config", "", "path to system JSON (empty = solar system)")
	simSpeed := flag.Float64("sim-speed", 3600, "simulated seconds per real second")
	scriptPath := flag.String("script", "", "admin script to execute on startup (path to .txt file)")
	dataDir := flag.String("data-dir", "data", "directory holding simulation content to serve to remote clients")
	spawnNearBody := flag.String("spawn-near", "", "cluster all client spawns near this named body instead of scattering randomly (testing aid)")
	idle := flag.String("idle", "run", "behaviour when no clients are attached: run | exit (suspend not yet implemented)")
	idleGrace := flag.Duration("idle-grace", 5*time.Second, "how long to tolerate zero attached clients before --idle=exit acts")
	flag.Parse()

	switch *idle {
	case "run", "exit":
	case "suspend":
		fmt.Fprintln(os.Stderr, "error: --idle=suspend is not implemented yet (F-041); use run or exit")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "error: --idle=%q is not recognised; use run or exit\n", *idle)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Build world ───────────────────────────────────────────────────────
	w, err := world.NewWorld(*simSpeed, *systemConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading world: %v\n", err)
		os.Exit(1)
	}

	// ── Build the content bundle remote clients fetch ─────────────────────
	// A failure here is reported but not fatal: the server can still run the
	// simulation and stream snapshots. Clients would render untextured, so the
	// failure must be visible rather than silent.
	bundle, err := assets.Build(*dataDir)
	if err != nil {
		log.Printf("WARNING: could not build asset bundle from %s: %v", *dataDir, err)
		log.Printf("WARNING: remote clients will be unable to fetch content and will render untextured")
		bundle = nil
	} else {
		log.Printf("asset bundle ready: %d entries, %.2f MB, hash %s",
			bundle.EntryCount, float64(bundle.Size())/(1024*1024), bundle.Hash[:16])
	}

	// runCtx lets the idle monitor and ShutdownService trigger the same graceful
	// shutdown path a signal would, rather than exiting abruptly.
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	// ── Build handlers ────────────────────────────────────────────────────
	// The session registry is what lets clients have identity: it assigns a
	// palette colour and a spawn position, and WorldHandler publishes its
	// contents in every snapshot so peers can render each other.
	sessionCfg := session.DefaultConfig()
	sessionCfg.Spawn = spawnNear(w, *spawnNearBody)
	registry := session.NewRegistry(sessionCfg)

	worldHandler := grpcserver.NewWorldHandler(registry)
	simHandler := grpcserver.NewSimulationHandler(func() *world.World { return w })
	assetHandler := grpcserver.NewAssetHandler(bundle, w.SystemPath())
	sessionHandler := grpcserver.NewSessionHandler(registry)
	shutdownHandler := grpcserver.NewShutdownHandler(runCancel)

	// ── Register the services meaningful for a headless server ────────────
	// Window / Camera / Navigation / etc. are render-client concerns: their
	// handlers route through the Raylib app's command channel, which does not
	// exist here, so they are deliberately omitted.
	//
	// Session and Shutdown are NOT render-client concerns and were previously
	// omitted by oversight, which made client identity impossible and left
	// ShutdownService returning 404 despite being implemented.
	mux := http.NewServeMux()

	simPath, simSvc := spacesimv1connect.NewSimulationServiceHandler(simHandler)
	mux.Handle(simPath, simSvc)

	worldPath, worldSvc := spacesimv1connect.NewWorldServiceHandler(worldHandler)
	mux.Handle(worldPath, worldSvc)

	assetPath, assetSvc := spacesimv1connect.NewAssetServiceHandler(assetHandler)
	mux.Handle(assetPath, assetSvc)

	sessionPath, sessionSvc := spacesimv1connect.NewSessionServiceHandler(sessionHandler)
	mux.Handle(sessionPath, sessionSvc)

	shutdownPath, shutdownSvc := spacesimv1connect.NewShutdownServiceHandler(shutdownHandler)
	mux.Handle(shutdownPath, shutdownSvc)

	// Wrap in h2c so the server speaks cleartext HTTP/2.
	//
	// Required, not optional: gRPC is an HTTP/2 protocol. Unary and
	// server-streaming calls happen to survive over HTTP/1.1 because both are
	// half-duplex, but a BIDIRECTIONAL stream cannot — SessionStream died
	// immediately with "write envelope: EOF" until this was wired, which is why it
	// had been implemented but never usable.
	httpSrv := &http.Server{
		Addr:        *addr,
		Handler:     h2c.NewHandler(mux, &http2.Server{}),
		IdleTimeout: 60 * time.Second,
	}

	// Stop paying for snapshots nobody consumes. This gates the O(objects) deep
	// clone on the sim goroutine, which is the dominant idle cost — not the
	// fan-out, which is trivial. Keyed on SUBSCRIBERS rather than sessions: an
	// admin observing without streaming creates no demand for the simulation to
	// advance (F-041 D7) but must still receive snapshots if it is streaming.
	w.SetSnapshotGate(func() bool { return worldHandler.StreamCount() > 0 })

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

	// ── Admin REPL (stdin + optional startup script) ─────────────────────
	go runAdminREPL(ctx, w, stop, *scriptPath)

	if *idle == "exit" {
		go watchIdle(runCtx, worldHandler, *idleGrace, runCancel)
	}

	// ── Start HTTP/gRPC server ────────────────────────────────────────────
	srvDone := make(chan error, 1)
	go func() {
		log.Printf("space-sim-server listening on %s", *addr)
		srvDone <- httpSrv.ListenAndServe()
	}()

	select {
	case <-runCtx.Done():
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

// watchIdle terminates the server once every client has detached.
//
// It keys off the snapshot-subscriber count because renderers do not register
// sessions yet; when they do, this should move to counting active non-admin
// sessions instead (F-041 D7/D13).
//
// Two behaviours matter here. It ARMS only after the first client attaches, so a
// server that has not been connected to yet is never killed by its own startup
// state. And it requires the grace period to elapse with zero subscribers, so a
// client's reconnect backoff does not look like a departure.
func watchIdle(ctx context.Context, h *grpcserver.WorldHandler, grace time.Duration, stop func()) {
	const poll = 500 * time.Millisecond
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	armed := false
	var emptySince time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		n := h.StreamCount()
		if n > 0 {
			if !armed {
				log.Printf("idle monitor armed: %d client(s) attached, will exit %s after the last one leaves", n, grace)
				armed = true
			}
			emptySince = time.Time{}
			continue
		}
		if !armed {
			continue
		}
		if emptySince.IsZero() {
			emptySince = time.Now()
			log.Printf("all clients detached; exiting in %s unless one returns", grace)
			continue
		}
		if time.Since(emptySince) >= grace {
			log.Printf("--idle=exit: no clients for %s, shutting down", grace)
			stop()
			return
		}
	}
}

// spawnNear returns a spawn provider that places a client just off a randomly
// chosen named body, so newly registered clients are both visible and separated.
//
// Without this every client lands at the origin, which in a heliocentric system is
// inside the star: invisible, and identical for every client.
func spawnNear(w *world.World, preferred string) session.SpawnFunc {
	return func() ([3]float64, string) {
		snap := w.LatestSnapshot()
		if snap.State == nil || len(snap.State.Objects) == 0 {
			// Registration can beat the first snapshot; anywhere off the origin
			// beats the centre of the star.
			return [3]float64{120, 0, 0}, ""
		}

		var candidates []*engine.Object
		for _, obj := range snap.State.Objects {
			// A named preference clusters every client near one body so they are
			// mutually visible. Random scatter is the F-020 spec behaviour (Q3) and
			// is right for gameplay, but it puts clients ~100 degrees apart, which
			// makes co-visibility testing impossible.
			if preferred != "" {
				if obj.Meta.Name == preferred {
					candidates = append(candidates, obj)
				}
				continue
			}
			switch obj.Meta.Category {
			case engine.CategoryPlanet, engine.CategoryDwarfPlanet, engine.CategoryMoon:
				candidates = append(candidates, obj)
			}
		}
		if len(candidates) == 0 && preferred != "" {
			log.Printf("WARNING: --spawn-near %q matched no body; falling back to the origin offset", preferred)
		}
		if len(candidates) == 0 {
			return [3]float64{120, 0, 0}, ""
		}

		body := candidates[rand.IntN(len(candidates))]

		// Stand off far enough to clear the body's rendered radius, which is
		// itself inflated well beyond true scale.
		standoff := float64(body.Meta.PhysicalRadius)*3 + 5
		theta := rand.Float64() * 2 * math.Pi
		phi := (rand.Float64() - 0.5) * math.Pi / 4 // keep near the orbital plane

		return [3]float64{
			float64(body.Anim.Position.X) + standoff*math.Cos(theta)*math.Cos(phi),
			float64(body.Anim.Position.Y) + standoff*math.Sin(phi),
			float64(body.Anim.Position.Z) + standoff*math.Sin(theta)*math.Cos(phi),
		}, body.Meta.Name
	}
}
