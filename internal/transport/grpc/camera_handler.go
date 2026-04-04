package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/google/uuid"
)

// CameraHandler implements spacesimv1connect.CameraServiceHandler.
type CameraHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewCameraHandler constructs a CameraHandler.
func NewCameraHandler(sendCmd func(rayapp.AppCmd) bool) *CameraHandler {
	return &CameraHandler{sendCmd: sendCmd}
}

func (h *CameraHandler) GetCamera(ctx context.Context, _ *connect.Request[v1.GetCameraRequest]) (*connect.Response[v1.GetCameraResponse], error) {
	respCh := make(chan rayapp.CameraSnapshot, 1)
	if !h.sendCmd(rayapp.GetCameraCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case snap := <-respCh:
		mode := "free"
		if snap.Mode == ui.CameraModeTracking {
			mode = "tracking"
		} else if snap.Mode == ui.CameraModeJumping {
			mode = "jumping"
		}
		return connect.NewResponse(&v1.GetCameraResponse{
			Version: 1,
			Camera: &v1.WireCameraState{
				YawDeg:      float32(snap.YawDeg),
				PitchDeg:    float32(snap.PitchDeg),
				PosX:        float64(snap.Position.X),
				PosY:        float64(snap.Position.Y),
				PosZ:        float64(snap.Position.Z),
				Mode:        mode,
				TrackTarget: snap.TrackTarget,
			},
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}

func (h *CameraHandler) SetCameraOrient(_ context.Context, req *connect.Request[v1.SetCameraOrientRequest]) (*connect.Response[v1.SetCameraOrientResponse], error) {
	if !h.sendCmd(rayapp.CameraOrientCmd{
		YawDeg:   float64(req.Msg.YawDeg),
		PitchDeg: float64(req.Msg.PitchDeg),
	}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetCameraOrientResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *CameraHandler) SetCameraPosition(_ context.Context, req *connect.Request[v1.SetCameraPositionRequest]) (*connect.Response[v1.SetCameraPositionResponse], error) {
	if !h.sendCmd(rayapp.CameraPositionCmd{
		Pos: engine.Vector3{X: float32(req.Msg.PosX), Y: float32(req.Msg.PosY), Z: float32(req.Msg.PosZ)},
	}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetCameraPositionResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *CameraHandler) SetCameraTrack(_ context.Context, req *connect.Request[v1.SetCameraTrackRequest]) (*connect.Response[v1.SetCameraTrackResponse], error) {
	if !h.sendCmd(rayapp.CameraTrackCmd{Name: req.Msg.Name}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetCameraTrackResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *CameraHandler) StartOrbit(_ context.Context, req *connect.Request[v1.StartOrbitRequest]) (*connect.Response[v1.StartOrbitResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidOrbit)
	}
	if req.Msg.SpeedDegPerSec == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidOrbit)
	}
	if req.Msg.Orbits <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidOrbit)
	}
	if !h.sendCmd(rayapp.OrbitCmd{
		Name:           req.Msg.Name,
		SpeedDegPerSec: float64(req.Msg.SpeedDegPerSec),
		Orbits:         float64(req.Msg.Orbits),
	}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.StartOrbitResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}
