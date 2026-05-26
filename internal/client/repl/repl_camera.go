package repl

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execCameraCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.CameraCenter:
		camResp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
		if err != nil {
			return false, err
		}
		target := camResp.Msg.GetCamera().GetTrackTarget()
		if target != "" {
			// Tracking a body — animate jump back to it.
			resp, err := r.navClient.JumpTo(ctx, connect.NewRequest(&v1.JumpToRequest{Names: []string{target}}))
			if err != nil {
				return false, err
			}
			r.printf("centering on %s  event_id=%s  status=%s\n", target, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		} else {
			// Free-fly — teleport to solar system origin.
			resp, err := r.camClient.SetCameraPosition(ctx, connect.NewRequest(&v1.SetCameraPositionRequest{
				PosX: 0, PosY: 0, PosZ: 0,
			}))
			if err != nil {
				return false, err
			}
			r.printf("centering on system origin  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())
		}

	case commands.CameraGet:
		resp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
		if err != nil {
			return false, err
		}
		cam := resp.Msg.GetCamera()
		r.printf("position: x=%.4f  y=%.4f  z=%.4f AU\n", cam.GetPosX(), cam.GetPosY(), cam.GetPosZ())
		r.printf("orient:   yaw=%.2f°  pitch=%.2f°\n", cam.GetYawDeg(), cam.GetPitchDeg())
		r.printf("mode:     %s  tracking: %q\n", cam.GetMode(), cam.GetTrackTarget())

	case commands.CameraOrient:
		resp, err := r.camClient.SetCameraOrient(ctx, connect.NewRequest(&v1.SetCameraOrientRequest{
			YawDeg: c.YawDeg, PitchDeg: c.PitchDeg,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.CameraPosition:
		resp, err := r.camClient.SetCameraPosition(ctx, connect.NewRequest(&v1.SetCameraPositionRequest{
			PosX: c.X, PosY: c.Y, PosZ: c.Z,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.CameraTrack:
		resp, err := r.camClient.SetCameraTrack(ctx, connect.NewRequest(&v1.SetCameraTrackRequest{
			Name: c.Name,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	}
	return false, nil
}
