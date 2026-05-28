// Command space-sim-client is the remote Space Sim renderer.
// It connects to a running space-sim-server, streams WorldSnapshots over
// ConnectRPC, and renders them with the standard Raylib renderer.
// No local physics simulation is started.
//
// Usage:
//
//	space-sim-client [flags]
//
// Flags:
//
//	--server  host:port of the space-sim-server (default "localhost:8080")
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

func main() {
	serverAddr := "localhost:8080"
	for i, arg := range os.Args[1:] {
		if arg == "--server" && i+2 < len(os.Args) {
			serverAddr = os.Args[i+2]
		}
		if len(arg) > 9 && arg[:9] == "--server=" {
			serverAddr = arg[9:]
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	snapSrc := &grpcSnapshotSource{}

	// Background goroutine: dial server, stream snapshots, store latest.
	go streamSnapshots(ctx, serverAddr, snapSrc)

	// Load config (uses defaults when no file exists at path).
	const appName = "space-sim-client"
	appConfigPath := rayapp.DefaultAppConfigPathFor(appName)
	appConfig, err := rayapp.LoadAppConfig(appConfigPath)
	if err != nil {
		log.Printf("app config: %v (using defaults)", err)
	}

	cfg := rayapp.Config{
		AppConfigPath: appConfigPath,
		AppConfig:     appConfig,
	}

	application, err := rayapp.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "app init error: %v\n", err)
		os.Exit(1)
	}

	if err := application.RunWithSnapshot(ctx, snapSrc); err != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", err)
		os.Exit(1)
	}
}

// grpcSnapshotSource satisfies protocol.SnapshotSource by storing the most
// recent snapshot received from the server in an atomic pointer.
// The render loop calls LatestSnapshot() on every frame; the streaming
// goroutine stores a new value on every received proto frame.
type grpcSnapshotSource struct {
	store atomic.Pointer[protocol.WorldSnapshot]
}

// LatestSnapshot returns the most recently received snapshot, or an empty
// WorldSnapshot if no snapshot has been received yet.
func (g *grpcSnapshotSource) LatestSnapshot() protocol.WorldSnapshot {
	if v := g.store.Load(); v != nil {
		return *v
	}
	return protocol.WorldSnapshot{}
}

// streamSnapshots connects to the server, calls WorldService.StreamSnapshot,
// converts each proto frame to a protocol.WorldSnapshot, and stores it in g.
// Reconnects with exponential back-off on connection failure.
func streamSnapshots(ctx context.Context, serverAddr string, g *grpcSnapshotSource) {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := doStream(ctx, serverAddr, g)
		if err == nil || ctx.Err() != nil {
			return
		}

		log.Printf("stream error: %v — reconnecting in %s", err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// doStream opens one streaming RPC and copies frames into g until the stream
// ends or ctx is cancelled. Returns nil if the stream ended cleanly.
func doStream(ctx context.Context, serverAddr string, g *grpcSnapshotSource) error {
	baseURL := "http://" + serverAddr
	client := spacesimv1connect.NewWorldServiceClient(
		&http.Client{Timeout: 0}, // no per-request timeout; rely on ctx
		baseURL,
		connect.WithGRPC(), // prefer gRPC framing (HTTP/2 upgrade if available)
	)

	stream, err := client.StreamSnapshot(ctx, connect.NewRequest(&v1.StreamSnapshotRequest{Version: 1}))
	if err != nil {
		return fmt.Errorf("StreamSnapshot: %w", err)
	}
	defer stream.Close()

	log.Printf("connected to %s", serverAddr)

	for stream.Receive() {
		msg := stream.Msg()
		snap := protoToSnapshot(msg)
		g.store.Store(&snap)
	}
	return stream.Err()
}

// protoToSnapshot converts a StreamSnapshotResponse to a protocol.WorldSnapshot.
// Fields not present in the wire format (Color, TexturePath, etc.) are left
// at zero/default values; the renderer falls back to solid-colour rendering.
func protoToSnapshot(resp *v1.StreamSnapshotResponse) protocol.WorldSnapshot {
	state := engine.NewSimulationState()
	state.Time = resp.SimulationTime
	state.SecondsPerSecond = resp.Speed

	for _, body := range resp.Bodies {
		obj := &engine.Object{
			Meta: engine.ObjectMetadata{
				Name:           body.Name,
				Category:       parseCategory(body.Category),
				ParentName:     body.ParentName,
				PhysicalRadius: body.PhysicalRadius,
			},
			Anim: engine.AnimationState{
				Position: engine.Vector3{
					X: float32(body.PosX),
					Y: float32(body.PosY),
					Z: float32(body.PosZ),
				},
			},
			Visible: body.Visible,
		}
		state.Objects = append(state.Objects, obj)
		state.ObjectMap[obj.Meta.Name] = obj
	}

	// Derive NavigationOrder from categories present.
	state.NavigationOrder = deriveOrder(state.Objects)

	return protocol.WorldSnapshot{
		State: state,
		Speed: float64(resp.Speed),
	}
}

// parseCategory maps a proto category label to an engine.ObjectCategory.
func parseCategory(s string) engine.ObjectCategory {
	switch s {
	case "star", "star_main_sequence":
		return engine.CategoryStarMainSequence
	case "star_pre_main":
		return engine.CategoryStarPreMain
	case "star_evolved":
		return engine.CategoryStarEvolved
	case "substellar":
		return engine.CategorySubstellar
	case "stellar_remnant", "blackhole":
		return engine.CategoryStellarRemnant
	case "planet":
		return engine.CategoryPlanet
	case "dwarf_planet":
		return engine.CategoryDwarfPlanet
	case "moon":
		return engine.CategoryMoon
	case "asteroid":
		return engine.CategoryAsteroid
	case "ring":
		return engine.CategoryRing
	case "belt":
		return engine.CategoryBelt
	default:
		return engine.CategoryPlanet // safest fallback
	}
}

// deriveOrder returns a stable NavigationOrder from the object set.
func deriveOrder(objects []*engine.Object) []engine.ObjectCategory {
	canonical := []engine.ObjectCategory{
		engine.CategoryStarPreMain,
		engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved,
		engine.CategorySubstellar,
		engine.CategoryStellarRemnant,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryBelt,
		engine.CategoryRogue,
		engine.CategoryArtifact,
	}
	present := make(map[engine.ObjectCategory]bool, len(objects))
	for _, obj := range objects {
		present[obj.Meta.Category] = true
	}
	order := make([]engine.ObjectCategory, 0, len(canonical))
	for _, cat := range canonical {
		if present[cat] {
			order = append(order, cat)
		}
	}
	return order
}
