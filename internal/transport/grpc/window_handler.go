package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
	"github.com/google/uuid"
)

// WindowHandler implements spacesimv1connect.WindowServiceHandler.
// Mutating RPCs push AppCmds to the main thread via sendCmd.
// GetWindow uses a round-trip AppCmd to read window state race-freely.
type WindowHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewWindowHandler constructs a WindowHandler.
func NewWindowHandler(sendCmd func(rayapp.AppCmd) bool) *WindowHandler {
	return &WindowHandler{sendCmd: sendCmd}
}

func (h *WindowHandler) GetWindow(ctx context.Context, _ *connect.Request[v1.GetWindowRequest]) (*connect.Response[v1.GetWindowResponse], error) {
	respCh := make(chan rayapp.WindowSnapshot, 1)
	if !h.sendCmd(rayapp.GetWindowCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case snap := <-respCh:
		return connect.NewResponse(&v1.GetWindowResponse{
			Version: 1,
			Window: &v1.WindowState{
				Width:      snap.Width,
				Height:     snap.Height,
				Fullscreen: snap.Fullscreen,
				Maximized:  snap.Maximized,
			},
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}

func (h *WindowHandler) SetWindowSize(_ context.Context, req *connect.Request[v1.SetWindowSizeRequest]) (*connect.Response[v1.SetWindowSizeResponse], error) {
	if req.Msg.Width <= 0 || req.Msg.Height <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidDimensions)
	}
	if !h.sendCmd(rayapp.WindowSizeCmd{Width: req.Msg.Width, Height: req.Msg.Height}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetWindowSizeResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *WindowHandler) SetWindowMaximize(_ context.Context, _ *connect.Request[v1.SetWindowMaximizeRequest]) (*connect.Response[v1.SetWindowMaximizeResponse], error) {
	if !h.sendCmd(rayapp.WindowMaximizeCmd{}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetWindowMaximizeResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

func (h *WindowHandler) SetWindowRestore(_ context.Context, _ *connect.Request[v1.SetWindowRestoreRequest]) (*connect.Response[v1.SetWindowRestoreResponse], error) {
	if !h.sendCmd(rayapp.WindowRestoreCmd{}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.SetWindowRestoreResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}
