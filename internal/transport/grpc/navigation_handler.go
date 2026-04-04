package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/google/uuid"
)

// NavigationHandler implements spacesimv1connect.NavigationServiceHandler.
type NavigationHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewNavigationHandler constructs a NavigationHandler.
func NewNavigationHandler(sendCmd func(rayapp.AppCmd) bool) *NavigationHandler {
	return &NavigationHandler{sendCmd: sendCmd}
}

func (h *NavigationHandler) GetVelocity(ctx context.Context, _ *connect.Request[v1.GetVelocityRequest]) (*connect.Response[v1.GetVelocityResponse], error) {
	respCh := make(chan engine.Vector3, 1)
	if !h.sendCmd(rayapp.GetVelocityCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case vel := <-respCh:
		return connect.NewResponse(&v1.GetVelocityResponse{
			Version:  1,
			Velocity: &v1.Velocity3{X: vel.X, Y: vel.Y, Z: vel.Z},
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}

func (h *NavigationHandler) SetVelocity(_ context.Context, req *connect.Request[v1.SetVelocityRequest]) (*connect.Response[v1.SetVelocityResponse], error) {
	vel := req.Msg.Velocity
	var v engine.Vector3
	if vel != nil {
		v = engine.Vector3{X: vel.X, Y: vel.Y, Z: vel.Z}
	}
	if !h.sendCmd(rayapp.SetVelocityCmd{Velocity: v}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetVelocityResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *NavigationHandler) JumpTo(_ context.Context, req *connect.Request[v1.JumpToRequest]) (*connect.Response[v1.JumpToResponse], error) {
	if len(req.Msg.Names) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoJumpTargets)
	}
	if !h.sendCmd(rayapp.JumpToCmd{Names: req.Msg.Names}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.JumpToResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}
