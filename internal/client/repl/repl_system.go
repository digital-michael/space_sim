package repl

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execSystemCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.SystemList:
		resp, err := r.sysClient.ListSystems(ctx, connect.NewRequest(&v1.ListSystemsRequest{}))
		if err != nil {
			return false, err
		}
		for _, s := range resp.Msg.Systems {
			r.printf("  %s\n", s.Label)
		}

	case commands.SystemGet:
		resp, err := r.sysClient.GetActiveSystem(ctx, connect.NewRequest(&v1.GetActiveSystemRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("active system: %s\n", resp.Msg.GetActive().GetLabel())

	case commands.SystemLoad:
		resp, err := r.sysClient.LoadSystem(ctx, connect.NewRequest(&v1.LoadSystemRequest{Path: c.Label}))
		if err != nil {
			return false, err
		}
		r.bodyNames = nil // new system — invalidate body name cache
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		if r.syncMode {
			r.waitForSystem(ctx, c.Label)
		}

	case commands.WindowGet:
		resp, err := r.winClient.GetWindow(ctx, connect.NewRequest(&v1.GetWindowRequest{}))
		if err != nil {
			return false, err
		}
		w := resp.Msg.GetWindow()
		r.printf("size: %dx%d  maximized: %v  fullscreen: %v\n",
			w.GetWidth(), w.GetHeight(), w.GetMaximized(), w.GetFullscreen())

	case commands.WindowSize:
		resp, err := r.winClient.SetWindowSize(ctx, connect.NewRequest(&v1.SetWindowSizeRequest{
			Width: c.Width, Height: c.Height,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.WindowMaximize:
		resp, err := r.winClient.SetWindowMaximize(ctx, connect.NewRequest(&v1.SetWindowMaximizeRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.WindowRestore:
		resp, err := r.winClient.SetWindowRestore(ctx, connect.NewRequest(&v1.SetWindowRestoreRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.WindowFullscreen:
		resp, err := r.winClient.SetWindowFullscreen(ctx, connect.NewRequest(&v1.SetWindowFullscreenRequest{
			Fullscreen: c.On,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	}
	return false, nil
}
