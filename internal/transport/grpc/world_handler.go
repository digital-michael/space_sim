package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
)

// WorldHandler implements spacesimv1connect.WorldServiceHandler.
// Phase 6c wires it to internal/protocol.Broadcaster so every WorldSnapshot
// published by the simulation loop is forwarded to connected streaming clients.
type WorldHandler struct {
	// TODO (Phase 6c): broadcaster *protocol.Broadcaster
}

// NewWorldHandler constructs a WorldHandler.
func NewWorldHandler() *WorldHandler {
	return &WorldHandler{}
}

// StreamSnapshot is a server-streaming RPC. For each connected client it
// subscribes to the Broadcaster and forwards snapshots until the client
// disconnects or the server context is cancelled.
//
// TODO (Phase 6c): subscribe to broadcaster, convert WorldSnapshot -> StreamSnapshotResponse,
// send via stream.Send, and deregister on context cancellation.
func (h *WorldHandler) StreamSnapshot(ctx context.Context, req *connect.Request[v1.StreamSnapshotRequest], stream *connect.ServerStream[v1.StreamSnapshotResponse]) error {
	// Placeholder: block until context cancelled so the stream stays open.
	<-ctx.Done()
	return nil
}
