package repl

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execSimCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.SetSpeed:
		resp, err := r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{
			SecondsPerSecond: c.SecondsPerSecond,
		}))
		if err != nil {
			return false, err
		}
		if c.SecondsPerSecond > 0 {
			r.lastSpeed = c.SecondsPerSecond
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetSpeed:
		resp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("speed = %.4g s/s\n", resp.Msg.SecondsPerSecond)

	case commands.SetDataset:
		level := levelToProto(c.Level)
		resp, err := r.simClient.SetDataset(ctx, connect.NewRequest(&v1.SetDatasetRequest{Level: level}))
		if err != nil {
			return false, err
		}
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s\n", ack.GetEventId(), ack.GetStatus())

	case commands.GetDataset:
		resp, err := r.simClient.GetDataset(ctx, connect.NewRequest(&v1.GetDatasetRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("dataset = %s\n", strings.ToLower(strings.TrimPrefix(resp.Msg.Level.String(), "DATASET_LEVEL_")))

	case commands.GetTime:
		resp, err := r.simClient.GetSimulationTime(ctx, connect.NewRequest(&v1.GetSimulationTimeRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("simulation_time = %.2f s (J2000)\n", resp.Msg.SecondsSinceJ2000)

	case commands.Stream:
		return false, r.runStream(ctx)

	case commands.Pause:
		speedResp, err := r.simClient.GetSpeed(ctx, connect.NewRequest(&v1.GetSpeedRequest{}))
		if err != nil {
			return false, fmt.Errorf("pause: %w", err)
		}
		if cur := speedResp.Msg.SecondsPerSecond; cur > 0 {
			r.lastSpeed = cur
		}
		_, err = r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{SecondsPerSecond: 0}))
		if err != nil {
			return false, err
		}
		r.printf("paused  (resume will restore %.4g sec/sec)\n", r.lastSpeed)

	case commands.Resume:
		speed := r.lastSpeed
		if speed <= 0 {
			speed = 1.0
		}
		resp, err := r.simClient.SetSpeed(ctx, connect.NewRequest(&v1.SetSpeedRequest{SecondsPerSecond: speed}))
		if err != nil {
			return false, err
		}
		r.lastSpeed = speed
		ack := resp.Msg.Ack
		r.printf("ok  event_id=%s  status=%s  (%.4g sec/sec)\n", ack.GetEventId(), ack.GetStatus(), speed)

	case commands.Shutdown:
		_, err := r.sdClient.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
		if err != nil {
			return false, err
		}
		r.printf("server shutting down\n")
		return true, nil

	}
	return false, nil
}
