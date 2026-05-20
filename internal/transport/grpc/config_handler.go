package grpcserver

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	rayapp "github.com/digital-michael/space_sim/internal/client/go/raylib/app"
)

// ConfigHandler implements spacesimv1connect.ConfigServiceHandler.
// It exposes keybindings hot-reload and active binding query over gRPC.
type ConfigHandler struct {
	sendCmd func(rayapp.AppCmd) bool
}

// NewConfigHandler constructs a ConfigHandler.
func NewConfigHandler(sendCmd func(rayapp.AppCmd) bool) *ConfigHandler {
	return &ConfigHandler{sendCmd: sendCmd}
}

// ReloadKeybindings re-reads the keybindings config file and hot-swaps the
// active KeyMap without restarting the app.
func (h *ConfigHandler) ReloadKeybindings(_ context.Context, req *connect.Request[v1.ReloadKeybindingsRequest]) (*connect.Response[v1.ReloadKeybindingsResponse], error) {
	// Pass an empty Path so the app reloads from its configured default.
	if !h.sendCmd(rayapp.ReloadKeymapCmd{Path: ""}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	return connect.NewResponse(&v1.ReloadKeybindingsResponse{
		Version: 1,
		Ack:     &v1.CommandAck{EventId: uuid.NewString(), Status: v1.AckStatus_ACK_STATUS_QUEUED},
	}), nil
}

// GetKeybindings returns a snapshot of the active key binding table.
func (h *ConfigHandler) GetKeybindings(ctx context.Context, _ *connect.Request[v1.GetKeybindingsRequest]) (*connect.Response[v1.GetKeybindingsResponse], error) {
	respCh := make(chan []rayapp.KeyBindingEntry, 1)
	if !h.sendCmd(rayapp.GetKeymapCmd{RespCh: respCh}) {
		return nil, connect.NewError(connect.CodeUnavailable, errCmdFull)
	}
	select {
	case entries := <-respCh:
		pb := make([]*v1.KeyBinding, 0, len(entries))
		for _, e := range entries {
			pb = append(pb, &v1.KeyBinding{
				Action: e.Action,
				Key:    e.Key,
				Mods:   e.Mods,
			})
		}
		return connect.NewResponse(&v1.GetKeybindingsResponse{
			Version:  1,
			Bindings: pb,
		}), nil
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, ctx.Err())
	}
}
