package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
)

// ShutdownHandler implements spacesimv1connect.ShutdownServiceHandler.
// Calling Shutdown cancels the injected cancel function, which causes
// the server's context to terminate and cascades to app.Run returning.
type ShutdownHandler struct {
	cancel context.CancelFunc
}

// NewShutdownHandler constructs a ShutdownHandler.
// cancel is typically the stop function returned by signal.NotifyContext.
func NewShutdownHandler(cancel context.CancelFunc) *ShutdownHandler {
	return &ShutdownHandler{cancel: cancel}
}

func (h *ShutdownHandler) Shutdown(_ context.Context, _ *connect.Request[v1.ShutdownRequest]) (*connect.Response[v1.ShutdownResponse], error) {
	h.cancel()
	return connect.NewResponse(&v1.ShutdownResponse{Version: 1}), nil
}
