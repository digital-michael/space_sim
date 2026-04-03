package grpcserver

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// WorldHandler implements both spacesimv1connect.WorldServiceHandler and
// protocol.Subscriber. Register it with app.RegisterSubscriber so the
// interactive loop delivers WorldSnapshot frames here; each active
// StreamSnapshot RPC receives those frames over its own channel.
type WorldHandler struct {
	mu      sync.RWMutex
	streams []chan protocol.WorldSnapshot
}

// NewWorldHandler constructs a WorldHandler. Register it as a subscriber with
// the Raylib app after construction:
//
//	app.RegisterSubscriber(worldHandler)
func NewWorldHandler() *WorldHandler {
	return &WorldHandler{}
}

// Receive implements protocol.Subscriber. It is called by the Broadcaster on
// every physics frame. Must be fast and non-blocking.
func (h *WorldHandler) Receive(snap protocol.WorldSnapshot) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.streams {
		select {
		case ch <- snap:
		default:
			// Drop: this stream consumer is too slow. Next frame will be delivered.
		}
	}
}

// StreamSnapshot registers a per-connection channel, forwards snapshots received
// via Receive, and deregisters on client disconnect or server shutdown.
func (h *WorldHandler) StreamSnapshot(ctx context.Context, req *connect.Request[v1.StreamSnapshotRequest], stream *connect.ServerStream[v1.StreamSnapshotResponse]) error {
	ch := make(chan protocol.WorldSnapshot, 8)
	h.addStream(ch)
	defer h.removeStream(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case snap := <-ch:
			if err := stream.Send(snapshotToProto(snap)); err != nil {
				return err
			}
		}
	}
}

func (h *WorldHandler) addStream(ch chan protocol.WorldSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams = append(h.streams, ch)
}

func (h *WorldHandler) removeStream(ch chan protocol.WorldSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, s := range h.streams {
		if s == ch {
			h.streams = append(h.streams[:i], h.streams[i+1:]...)
			return
		}
	}
}

// ─── conversion ───────────────────────────────────────────────────────────────

func snapshotToProto(snap protocol.WorldSnapshot) *v1.StreamSnapshotResponse {
	bodies := make([]*v1.BodyState, 0, len(snap.State.Objects))
	for _, obj := range snap.State.Objects {
		if !obj.Visible {
			continue
		}
		bodies = append(bodies, objectToProto(obj))
	}
	return &v1.StreamSnapshotResponse{
		Version:        1,
		SimulationTime: snap.State.Time,
		Speed:          float32(snap.Speed),
		Bodies:         bodies,
	}
}

func objectToProto(obj *engine.Object) *v1.BodyState {
	return &v1.BodyState{
		Version:        1,
		Name:           obj.Meta.Name,
		Category:       categoryLabel(obj.Meta.Category),
		ParentName:     obj.Meta.ParentName,
		PosX:           float64(obj.Anim.Position.X),
		PosY:           float64(obj.Anim.Position.Y),
		PosZ:           float64(obj.Anim.Position.Z),
		PhysicalRadius: obj.Meta.PhysicalRadius,
		Visible:        obj.Visible,
	}
}

func categoryLabel(c engine.ObjectCategory) string {
	switch c {
	case engine.CategoryPlanet:
		return "planet"
	case engine.CategoryDwarfPlanet:
		return "dwarf_planet"
	case engine.CategoryMoon:
		return "moon"
	case engine.CategoryAsteroid:
		return "asteroid"
	case engine.CategoryRing:
		return "ring"
	case engine.CategoryStar:
		return "star"
	case engine.CategoryBelt:
		return "belt"
	default:
		return "unknown"
	}
}
