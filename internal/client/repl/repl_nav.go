package repl

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execNavCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.NavStop:
		resp, err := r.navClient.SetVelocity(ctx, connect.NewRequest(&v1.SetVelocityRequest{
			Velocity: &v1.Velocity3{},
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		r.updateSessionPos(ctx)

	case commands.NavVelocity:
		resp, err := r.navClient.GetVelocity(ctx, connect.NewRequest(&v1.GetVelocityRequest{}))
		if err != nil {
			return false, err
		}
		vel := resp.Msg.GetVelocity()
		r.printf("velocity: vx=%.4f  vy=%.4f  vz=%.4f AU/s\n", vel.GetX(), vel.GetY(), vel.GetZ())

	case commands.NavMove:
		var vx, vy, vz float32
		switch c.Dir {
		case "forward":
			vz = -c.Velocity
		case "back":
			vz = c.Velocity
		case "left":
			vx = -c.Velocity
		case "right":
			vx = c.Velocity
		case "up":
			vy = c.Velocity
		case "down":
			vy = -c.Velocity
		}
		resp, err := r.navClient.SetVelocity(ctx, connect.NewRequest(&v1.SetVelocityRequest{
			Velocity: &v1.Velocity3{X: vx, Y: vy, Z: vz},
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.NavJump:
		resp, err := r.navClient.JumpTo(ctx, connect.NewRequest(&v1.JumpToRequest{Names: c.Names}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		if r.syncMode {
			r.waitForCamera(ctx)
		}
		r.updateSessionPos(ctx)

	case commands.Orbit:
		resp, err := r.camClient.StartOrbit(ctx, connect.NewRequest(&v1.StartOrbitRequest{
			Name:           c.Name,
			SpeedDegPerSec: float32(c.SpeedDegPerSec),
			Orbits:         float32(c.Orbits),
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  orbiting %s at %.2f°/s × %.4g orbits  event_id=%s  status=%s\n",
			c.Name, c.SpeedDegPerSec, c.Orbits, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		if r.syncMode {
			speed := c.SpeedDegPerSec
			if speed < 0 {
				speed = -speed
			}
			if speed > 0 {
				orbitSecs := (c.Orbits * 360.0) / speed
				select {
				case <-ctx.Done():
				case <-time.After(time.Duration(orbitSecs * float64(time.Second))):
				}
			}
		}

	case commands.Sleep:
		time.Sleep(time.Duration(c.Seconds * float64(time.Second)))

	}
	return false, nil
}
