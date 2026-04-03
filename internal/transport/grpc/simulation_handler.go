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
// It delegates commands directly to world.World (which embeds engine.Simulation)
// and reads state from atomic snapshots.
//
// SetSpeed and SetAsteroidDataset execute synchronously on the engine's command
// channel (non-blocking send) and return a CommandAck immediately — consistent
// with the CQRS pattern used throughout the server.
type SimulationHandler struct {
	world *world.World
}

// NewSimulationHandler constructs a SimulationHandler backed by w.
func NewSimulationHandler(w *world.World) *SimulationHandler {
	return &SimulationHandler{world: w}
}

func (h *SimulationHandler) SetSpeed(ctx context.Context, req *connect.Request[v1.SetSpeedRequest]) (*connect.Response[v1.SetSpeedResponse], error) {
	if h.world == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	if req.Msg.SecondsPerSecond <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	h.world.SetSpeed(float64(req.Msg.SecondsPerSecond))
	return connect.NewResponse(&v1.SetSpeedResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetSpeed(ctx context.Context, req *connect.Request[v1.GetSpeedRequest]) (*connect.Response[v1.GetSpeedResponse], error) {
	if h.world == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	return connect.NewResponse(&v1.GetSpeedResponse{
		Version:          1,
		SecondsPerSecond: float32(h.world.GetSpeed()),
	}), nil
}

func (h *SimulationHandler) SetDataset(ctx context.Context, req *connect.Request[v1.SetDatasetRequest]) (*connect.Response[v1.SetDatasetResponse], error) {
	if h.world == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	dataset, ok := protoLevelToEngine(req.Msg.Level)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	h.world.SetAsteroidDataset(dataset)
	return connect.NewResponse(&v1.SetDatasetResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetDataset(ctx context.Context, req *connect.Request[v1.GetDatasetRequest]) (*connect.Response[v1.GetDatasetResponse], error) {
	if h.world == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	snap := h.world.Snapshot()
	return connect.NewResponse(&v1.GetDatasetResponse{
		Version: 1,
		Level:   engineDatasetToProto(snap.State.CurrentDataset),
	}), nil
}

func (h *SimulationHandler) GetSimulationTime(ctx context.Context, req *connect.Request[v1.GetSimulationTimeRequest]) (*connect.Response[v1.GetSimulationTimeResponse], error) {
	if h.world == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, nil)
	}
	snap := h.world.Snapshot()
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
