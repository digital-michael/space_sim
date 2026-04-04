package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/google/uuid"
)

// PerformanceHandler implements spacesimv1connect.PerformanceServiceHandler.
type PerformanceHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewPerformanceHandler constructs a PerformanceHandler.
func NewPerformanceHandler(sendCmd func(rayapp.AppCmd) bool) *PerformanceHandler {
	return &PerformanceHandler{sendCmd: sendCmd}
}

func (h *PerformanceHandler) GetPerformance(ctx context.Context, _ *connect.Request[v1.GetPerformanceRequest]) (*connect.Response[v1.GetPerformanceResponse], error) {
	respCh := make(chan rayapp.PerfSnapshot, 1)
	if !h.sendCmd(rayapp.GetPerfCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case snap := <-respCh:
		o := snap.Options
		return connect.NewResponse(&v1.GetPerformanceResponse{
			Version: 1,
			State: &v1.PerformanceState{
				FrustumCulling:      o.FrustumCulling,
				LodEnabled:          o.LODEnabled,
				InstancedRendering:  o.InstancedRendering,
				SpatialPartition:    o.SpatialPartition,
				PointRendering:      o.PointRendering,
				ImportanceThreshold: int32(o.ImportanceThreshold),
				UseInPlaceSwap:      o.UseInPlaceSwap,
				CameraSpeed:         snap.CameraSpeed,
				NumWorkers:          int32(snap.NumWorkers),
				HudVisible:          snap.HUDVisible,
			},
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}

func (h *PerformanceHandler) SetPerformance(_ context.Context, req *connect.Request[v1.SetPerformanceRequest]) (*connect.Response[v1.SetPerformanceResponse], error) {
	msg := req.Msg
	s := msg.State
	if s == nil {
		s = &v1.PerformanceState{}
	}

	cmd := rayapp.PerfSetCmd{
		Options: ui.PerformanceOptions{
			FrustumCulling:      s.FrustumCulling,
			LODEnabled:          s.LodEnabled,
			InstancedRendering:  s.InstancedRendering,
			SpatialPartition:    s.SpatialPartition,
			PointRendering:      s.PointRendering,
			ImportanceThreshold: int(s.ImportanceThreshold),
			UseInPlaceSwap:      s.UseInPlaceSwap,
		},
		CameraSpeed: s.CameraSpeed,
		NumWorkers:  int(s.NumWorkers),
		HUDVisible:  s.HudVisible,

		SetFrustumCulling:      msg.SetFrustumCulling,
		SetLODEnabled:          msg.SetLodEnabled,
		SetInstancedRendering:  msg.SetInstancedRendering,
		SetSpatialPartition:    msg.SetSpatialPartition,
		SetPointRendering:      msg.SetPointRendering,
		SetImportanceThreshold: msg.SetImportanceThreshold,
		SetUseInPlaceSwap:      msg.SetUseInPlaceSwap,
		SetCameraSpeed:         msg.SetCameraSpeed,
		SetNumWorkers:          msg.SetNumWorkers,
		SetHUDVisible:          msg.SetHudVisible,
	}

	if !h.sendCmd(cmd) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetPerformanceResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}
