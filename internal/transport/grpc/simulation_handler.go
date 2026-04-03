package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/google/uuid"
)

// SimulationHandler implements spacesimv1connect.SimulationServiceHandler.
// Phase 6c wires it to internal/server/eventqueue and internal/server/runtime.
type SimulationHandler struct {
	// TODO (Phase 6c): eventQueue *eventqueue.QueueManager
	// TODO (Phase 6c): runtime    *runtime.RuntimeEnvironment
}

// NewSimulationHandler constructs a SimulationHandler.
func NewSimulationHandler() *SimulationHandler {
	return &SimulationHandler{}
}

func (h *SimulationHandler) SetSpeed(ctx context.Context, req *connect.Request[v1.SetSpeedRequest]) (*connect.Response[v1.SetSpeedResponse], error) {
	// TODO (Phase 6c): enqueue a speed-change event and return its ID.
	if req.Msg.SecondsPerSecond <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&v1.SetSpeedResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetSpeed(ctx context.Context, req *connect.Request[v1.GetSpeedRequest]) (*connect.Response[v1.GetSpeedResponse], error) {
	// TODO (Phase 6c): read speed from runtime.RuntimeEnvironment.
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (h *SimulationHandler) SetDataset(ctx context.Context, req *connect.Request[v1.SetDatasetRequest]) (*connect.Response[v1.SetDatasetResponse], error) {
	// TODO (Phase 6c): enqueue a dataset-change event and return its ID.
	if req.Msg.Level == v1.DatasetLevel_DATASET_LEVEL_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&v1.SetDatasetResponse{
		Version: 1,
		Ack: &v1.CommandAck{
			EventId: uuid.NewString(),
			Status:  v1.AckStatus_ACK_STATUS_QUEUED,
		},
	}), nil
}

func (h *SimulationHandler) GetDataset(ctx context.Context, req *connect.Request[v1.GetDatasetRequest]) (*connect.Response[v1.GetDatasetResponse], error) {
	// TODO (Phase 6c): read active dataset from runtime.RuntimeEnvironment.
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (h *SimulationHandler) GetSimulationTime(ctx context.Context, req *connect.Request[v1.GetSimulationTimeRequest]) (*connect.Response[v1.GetSimulationTimeResponse], error) {
	// TODO (Phase 6c): read simulation time from runtime.RuntimeEnvironment.
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
