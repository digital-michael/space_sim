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
		infos = append(infos, sessionToProto(s))
	}
	return connect.NewResponse(&v1.ListSessionsResponse{
		Version:  1,
		Sessions: infos,
	}), nil
}

// UpdatePosition stores the client's last-known world position.
func (h *SessionHandler) UpdatePosition(
	ctx context.Context,
	req *connect.Request[v1.UpdatePositionRequest],
) (*connect.Response[v1.UpdatePositionResponse], error) {
	pos := [3]float64{req.Msg.PosX, req.Msg.PosY, req.Msg.PosZ}
	if err := h.registry.UpdatePosition(req.Msg.SessionId, pos); err != nil {
		if err == session.ErrNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdatePositionResponse{Version: 1}), nil
}

// UpdatePOV stores the client's last-known point-of-view direction.
func (h *SessionHandler) UpdatePOV(
	ctx context.Context,
	req *connect.Request[v1.UpdatePOVRequest],
) (*connect.Response[v1.UpdatePOVResponse], error) {
	pov := [3]float32{req.Msg.PovX, req.Msg.PovY, req.Msg.PovZ}
	if err := h.registry.UpdatePOV(req.Msg.SessionId, pov); err != nil {
		if err == session.ErrNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdatePOVResponse{Version: 1}), nil
}

// SessionStream handles the bidirectional streaming RPC. The client sends
// ClientUpdate messages; the server pushes SessionDelta events describing
// session changes (add / update / remove) across all connected clients.
func (h *SessionHandler) SessionStream(
	ctx context.Context,
	stream *connect.BidiStream[v1.ClientUpdate, v1.SessionDelta],
) error {
	// Subscribe to registry events so we can push deltas to this client.
	eventCh, cancel := h.registry.Subscribe()
	defer cancel()

	// Push the current full session list as ADD events so the client bootstraps.
	for _, s := range h.registry.All() {
		delta := &v1.SessionDelta{
			Version:    1,
			ChangeType: v1.ChangeType_CHANGE_TYPE_ADD,
			Session:    sessionToProto(s),
		}
		if err := stream.Send(delta); err != nil {
			return err
		}
	}

	// readErr receives the error (or nil) from the receive goroutine.
	readErr := make(chan error, 1)
	go func() {
		for {
			msg, err := stream.Receive()
			if err != nil {
				readErr <- err
				return
			}
			pos := [3]float64{msg.PosX, msg.PosY, msg.PosZ}
			pov := [3]float32{msg.PovX, msg.PovY, msg.PovZ}
			// Best-effort: ignore not-found (client may have stale ID).
			_ = h.registry.UpdatePosition(msg.SessionId, pos)
			_ = h.registry.UpdatePOV(msg.SessionId, pov)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-readErr:
			return err
		case e, ok := <-eventCh:
			if !ok {
				return nil
			}
			delta := sessionEventToDelta(e)
			if err := stream.Send(delta); err != nil {
				return err
			}
		}
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func sessionToProto(s *session.ClientSession) *v1.ClientSessionInfo {
	return &v1.ClientSessionInfo{
		Version:   1,
		SessionId: s.SessionID,
		Label:     s.Label,
		Role:      v1.ClientRole(s.Role),
		ColorRgb:  []byte{s.Color[0], s.Color[1], s.Color[2]},
		PosX:      s.Position[0],
		PosY:      s.Position[1],
		PosZ:      s.Position[2],
		PovX:      s.POV[0],
		PovY:      s.POV[1],
		PovZ:      s.POV[2],
	}
}

func sessionEventToDelta(e session.SessionEvent) *v1.SessionDelta {
	delta := &v1.SessionDelta{Version: 1}
	switch e.Type {
	case session.SessionEventAdd:
		delta.ChangeType = v1.ChangeType_CHANGE_TYPE_ADD
		delta.Session = sessionToProto(e.Session)
	case session.SessionEventUpdate:
		delta.ChangeType = v1.ChangeType_CHANGE_TYPE_UPDATE
		delta.Session = sessionToProto(e.Session)
	case session.SessionEventRemove:
		delta.ChangeType = v1.ChangeType_CHANGE_TYPE_REMOVE
		delta.Session = &v1.ClientSessionInfo{SessionId: e.ID}
	}
	return delta
}
