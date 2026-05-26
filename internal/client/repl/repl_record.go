package repl

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execRecordCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.RecordStart:
		resolved, err := resolveRecordingPath(c.Filename)
		if err != nil {
			return false, err
		}
		resp, err := r.recClient.StartRecording(ctx, connect.NewRequest(&v1.StartRecordingRequest{
			Version:    1,
			OutputPath: resolved,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  record start %s  event_id=%s  status=%s\n", resolved, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.RecordPause:
		resp, err := r.recClient.PauseRecording(ctx, connect.NewRequest(&v1.PauseRecordingRequest{Version: 1}))
		if err != nil {
			return false, err
		}
		r.printf("ok  record pause  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.RecordStop:
		resp, err := r.recClient.StopRecording(ctx, connect.NewRequest(&v1.StopRecordingRequest{Version: 1}))
		if err != nil {
			return false, err
		}
		r.printf("ok  record stop  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.RecordDelete:
		resolved, err := resolveRecordingPath(c.Filename)
		if err != nil {
			return false, err
		}
		if err := os.Remove(resolved); err != nil {
			return false, fmt.Errorf("record delete: %w", err)
		}
		r.printf("deleted %s\n", resolved)

	}
	return false, nil
}
