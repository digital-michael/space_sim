package repl

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execSessionCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.SessionRegister:
		resp, err := r.sesClient.RegisterClient(ctx, connect.NewRequest(&v1.RegisterClientRequest{
			Version: 1,
			Label:   c.Label,
			Role:    v1.ClientRole_CLIENT_ROLE_PLAYER,
		}))
		if err != nil {
			return false, err
		}
		r.sessionID = resp.Msg.SessionId
		rgb := resp.Msg.ColorRgb
		r.printf("registered  session_id=%s  role=%v  color=rgb(%d,%d,%d)\n",
			resp.Msg.SessionId, resp.Msg.Role, rgb[0], rgb[1], rgb[2])

	case commands.SessionUnregister:
		if r.sessionID == "" {
			r.printf("not registered\n")
			return false, nil
		}
		_, err := r.sesClient.UnregisterClient(ctx, connect.NewRequest(&v1.UnregisterClientRequest{
			Version:   1,
			SessionId: r.sessionID,
		}))
		if err != nil {
			return false, err
		}
		r.printf("unregistered session %s\n", r.sessionID)
		r.sessionID = ""

	case commands.SessionList:
		resp, err := r.sesClient.ListSessions(ctx, connect.NewRequest(&v1.ListSessionsRequest{Version: 1}))
		if err != nil {
			return false, err
		}
		if len(resp.Msg.Sessions) == 0 {
			r.printf("no sessions\n")
			return false, nil
		}
		r.printf("%-36s  %-32s  %-12s  %-28s  %s\n", "SESSION_ID", "LABEL", "ROLE", "POSITION (AU)", "COLOR")
		r.printf("%s\n", strings.Repeat("-", 120))
		for _, s := range resp.Msg.Sessions {
			rgb := s.ColorRgb
			pos := fmt.Sprintf("%.4f, %.4f, %.4f", s.PosX, s.PosY, s.PosZ)
			r.printf("%-36s  %-32s  %-12v  %-28s  rgb(%d,%d,%d)\n",
				s.SessionId, s.Label, s.Role, pos, rgb[0], rgb[1], rgb[2])
		}

	case commands.SessionKick:
		if r.sessionID == "" {
			r.printf("not registered\n")
			return false, nil
		}
		_, err := r.sesClient.KickClient(ctx, connect.NewRequest(&v1.KickClientRequest{
			Version:         1,
			AdminSessionId:  r.sessionID,
			TargetSessionId: c.TargetSessionID,
		}))
		if err != nil {
			return false, err
		}
		r.printf("kicked session %s\n", c.TargetSessionID)

	case commands.SessionTeleport:
		if r.sessionID == "" {
			r.printf("not registered\n")
			return false, nil
		}
		snap, err := r.oneSnapshot(ctx)
		if err != nil {
			return false, fmt.Errorf("teleport: snapshot: %w", err)
		}
		var posX, posY, posZ float64
		found := false
		for _, b := range snap.Bodies {
			if strings.EqualFold(b.Name, c.Body) {
				posX, posY, posZ = b.PosX, b.PosY, b.PosZ
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Errorf("teleport: body %q not found", c.Body)
		}
		_, err = r.sesClient.TeleportClient(ctx, connect.NewRequest(&v1.TeleportClientRequest{
			Version:         1,
			AdminSessionId:  r.sessionID,
			TargetSessionId: c.TargetSessionID,
			PosX:            posX,
			PosY:            posY,
			PosZ:            posZ,
		}))
		if err != nil {
			return false, err
		}
		r.printf("teleported session %s to %s (%.4f, %.4f, %.4f)\n",
			c.TargetSessionID, c.Body, posX, posY, posZ)

	}
	return false, nil
}
