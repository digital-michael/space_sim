package repl

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execMiscCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.Bodies:
		return false, r.runBodies(ctx, c.Category)

	case commands.Inspect:
		return false, r.runInspect(ctx, c.Name)

	case commands.Status:
		return false, r.runStatus(ctx)

	case commands.Clear:
		r.printf("\x1b[2J\x1b[H")

	case commands.Help:
		r.printHelp()

	case commands.Quit:
		r.printf("bye\n")
		return true, nil

	case commands.Sync:
		r.syncMode = c.On
		onOff := "off"
		if c.On {
			onOff = "on"
		}
		r.printf("sync %s\n", onOff)

	case commands.ConfigReloadKeybindings:
		resp, err := r.cfgClient.ReloadKeybindings(ctx, connect.NewRequest(&v1.ReloadKeybindingsRequest{Version: 1}))
		if err != nil {
			return false, err
		}
		r.printf("ok  config reload keybindings  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.HelpKeys:
		resp, err := r.cfgClient.GetKeybindings(ctx, connect.NewRequest(&v1.GetKeybindingsRequest{Version: 1}))
		if err != nil {
			return false, err
		}
		r.printf("%-40s  %-20s  %s\n", "ACTION", "KEY", "MODS")
		r.printf("%s\n", strings.Repeat("-", 70))
		for _, b := range resp.Msg.Bindings {
			mods := strings.Join(b.Mods, "+")
			if mods == "" {
				mods = "-"
			}
			r.printf("%-40s  %-20s  %s\n", b.Action, b.Key, mods)
		}

	}
	return false, nil
}
