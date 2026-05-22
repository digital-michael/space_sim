package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/server/session"
)

// SessionHandler implements spacesimv1connect.SessionServiceHandler.
// It manages client session registration, unregistration, and listing.
type SessionHandler struct {
	registry session.Registry
}

// NewSessionHandler constructs a SessionHandler backed by the given registry.
func NewSessionHandler(reg session.Registry) *SessionHandler {
	return &SessionHandler{registry: reg}
}

// RegisterClient creates a new session in the registry.
func (h *SessionHandler) RegisterClient(
	ctx context.Context,
	req *connect.Request[v1.RegisterClientRequest],
) (*connect.Response[v1.RegisterClientResponse], error) {
	sess, err := h.registry.Register(session.RegisterRequest{
		ClientUUID:  req.Msg.ClientUuid,
		Label:       req.Msg.Label,
		Role:        session.ClientRole(req.Msg.Role),
		AdminSecret: req.Msg.AdminSecret,
	})
	if err != nil {
		if err == session.ErrCapacityExceeded {
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RegisterClientResponse{
		Version:   1,
		SessionId: sess.SessionID,
		Role:      v1.ClientRole(sess.Role),
		ColorRgb:  []byte{sess.Color[0], sess.Color[1], sess.Color[2]},
	}), nil
}

// UnregisterClient removes a session from the registry.
func (h *SessionHandler) UnregisterClient(
	ctx context.Context,
	req *connect.Request[v1.UnregisterClientRequest],
) (*connect.Response[v1.UnregisterClientResponse], error) {
	h.registry.Unregister(req.Msg.SessionId)
	return connect.NewResponse(&v1.UnregisterClientResponse{Version: 1}), nil
}

// ListSessions returns all currently registered sessions.
func (h *SessionHandler) ListSessions(
	ctx context.Context,
	req *connect.Request[v1.ListSessionsRequest],
) (*connect.Response[v1.ListSessionsResponse], error) {
	all := h.registry.All()
	infos := make([]*v1.ClientSessionInfo, 0, len(all))
	for _, s := range all {
		infos = append(infos, &v1.ClientSessionInfo{
			Version:   1,
			SessionId: s.SessionID,
			Label:     s.Label,
			Role:      v1.ClientRole(s.Role),
			ColorRgb:  []byte{s.Color[0], s.Color[1], s.Color[2]},
		})
	}
	return connect.NewResponse(&v1.ListSessionsResponse{
		Version:  1,
		Sessions: infos,
	}), nil
}
