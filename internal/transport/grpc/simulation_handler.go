package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/sim/world"
	"github.com/google/uuid"
)

// SimulationHandler implements spacesimv1connect.SimulationServiceHandler.
// It resolves the live world lazily via worldFn on each call so the handler
// can be constructed before the Raylib session (and therefore the *world.World)
// exists. worldFn must return nil until the session is ready; all RPCs return
// CodeUnimplemented in that case.
//
// SetSpeed and SetAsteroidDataset execute synchronously on the engine's command
// channel (non-blocking send) and return a CommandAck immediately — consistent
// with the CQRS pattern used throughout the server.
type SimulationHandler struct {
	worldFn func() *world.World
}

// NewSimulationHandler constructs a SimulationHandler. worldFn is called on
// every RPC to obtain the current *world.World; it must be safe to call from
// any goroutine. Pass a function that returns nil (e.g. func() *world.World
// { return nil }) to create a handler that returns CodeUnimplemented until the
// world is available.
//
// Typical use in space-sim-grpc:
//
//	grpcserver.NewSimulationHandler(application.World)
func NewSimulationHandler(worldFn func() *world.World) *SimulationHandler {
	return &SimulationHandler{worldFn: worldFn}
}

// world returns the current *world.World, or nil if worldFn is not set or
// returns nil. Centralises the nil-check so handler methods stay concise.
func (h *SimulationHandler) world() *world.World {
	if h.worldFn == nil {
		return nil
	}
	return h.worldFn()
}

func (h *SimulationHandler) SetSpeed(ctx context.Context, req *connect.Request[v1.SetSpeedRequest]) (*connect.Response[v1.SetSpeedResponse], error) {
	w := h.world()
	if w == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	if req.Msg.SecondsPerSecond < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	w.SetSpeed(float64(req.Msg.SecondsPerSecond))
	return connect.NewResponse(&v1.SetSpeedResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetSpeed(ctx context.Context, req *connect.Request[v1.GetSpeedRequest]) (*connect.Response[v1.GetSpeedResponse], error) {
	w := h.world()
	if w == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	return connect.NewResponse(&v1.GetSpeedResponse{
		Version:          1,
		SecondsPerSecond: float32(w.GetSpeed()),
	}), nil
}

func (h *SimulationHandler) SetDataset(ctx context.Context, req *connect.Request[v1.SetDatasetRequest]) (*connect.Response[v1.SetDatasetResponse], error) {
	w := h.world()
	if w == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	dataset, ok := protoLevelToEngine(req.Msg.Level)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	w.SetAsteroidDataset(dataset)
	return connect.NewResponse(&v1.SetDatasetResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetDataset(ctx context.Context, req *connect.Request[v1.GetDatasetRequest]) (*connect.Response[v1.GetDatasetResponse], error) {
	w := h.world()
	if w == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	snap := w.Snapshot()
	return connect.NewResponse(&v1.GetDatasetResponse{
		Version: 1,
		Level:   engineDatasetToProto(snap.State.CurrentDataset),
	}), nil
}

func (h *SimulationHandler) GetSimulationTime(ctx context.Context, req *connect.Request[v1.GetSimulationTimeRequest]) (*connect.Response[v1.GetSimulationTimeResponse], error) {
	w := h.world()
	if w == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	snap := w.Snapshot()
	return connect.NewResponse(&v1.GetSimulationTimeResponse{
		Version:           1,
		SecondsSinceJ2000: snap.State.Time,
	}), nil
}

// ─── proto ↔ engine conversions ───────────────────────────────────────────

func protoLevelToEngine(level v1.DatasetLevel) (engine.AsteroidDataset, bool) {
	switch level {
	case v1.DatasetLevel_DATASET_LEVEL_SMALL:
		return engine.AsteroidDatasetSmall, true
	case v1.DatasetLevel_DATASET_LEVEL_MEDIUM:
		return engine.AsteroidDatasetMedium, true
	case v1.DatasetLevel_DATASET_LEVEL_LARGE:
		return engine.AsteroidDatasetLarge, true
	case v1.DatasetLevel_DATASET_LEVEL_HUGE:
		return engine.AsteroidDatasetHuge, true
	default:
		return 0, false
	}
}

func engineDatasetToProto(d engine.AsteroidDataset) v1.DatasetLevel {
	switch d {
	case engine.AsteroidDatasetSmall:
		return v1.DatasetLevel_DATASET_LEVEL_SMALL
	case engine.AsteroidDatasetMedium:
		return v1.DatasetLevel_DATASET_LEVEL_MEDIUM
	case engine.AsteroidDatasetLarge:
		return v1.DatasetLevel_DATASET_LEVEL_LARGE
	case engine.AsteroidDatasetHuge:
		return v1.DatasetLevel_DATASET_LEVEL_HUGE
	default:
		return v1.DatasetLevel_DATASET_LEVEL_UNSPECIFIED
	}
}
