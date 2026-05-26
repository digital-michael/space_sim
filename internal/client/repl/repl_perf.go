package repl

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/internal/client/commands"
)

func (r *REPL) execPerfCmds(ctx context.Context, cmd commands.Cmd) (bool, error) {
	switch c := cmd.(type) {

	case commands.PerfGet:
		resp, err := r.perfClient.GetPerformance(ctx, connect.NewRequest(&v1.GetPerformanceRequest{}))
		if err != nil {
			return false, err
		}
		p := resp.Msg.GetState()
		r.printf("frustum_culling:      %v\n", p.GetFrustumCulling())
		r.printf("lod_enabled:          %v\n", p.GetLodEnabled())
		r.printf("instanced_rendering:  %v\n", p.GetInstancedRendering())
		r.printf("spatial_partition:    %v\n", p.GetSpatialPartition())
		r.printf("point_rendering:      %v\n", p.GetPointRendering())
		r.printf("importance_threshold: %d\n", p.GetImportanceThreshold())
		r.printf("use_in_place_swap:    %v\n", p.GetUseInPlaceSwap())
		r.printf("camera_speed:         %.4g\n", p.GetCameraSpeed())
		r.printf("workers:              %d\n", p.GetNumWorkers())

	case commands.PerfSet:
		req := perfSetField(c.Field, c.Value)
		if req == nil {
			return false, fmt.Errorf("perf set: unknown field %q", c.Field)
		}
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(req))
		if err != nil {
			return false, err
		}
		r.printf("ok  event_id=%s  status=%s\n", resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.HUD:
		hudReq := &v1.SetPerformanceRequest{
			State:         &v1.PerformanceState{HudVisible: c.Visible},
			SetHudVisible: true,
		}
		if !c.Visible {
			// hud off clears all category flags so that turning individual
			// categories back on via "hud info on" starts from a clean state.
			hudReq.SetHudDebug = true
			hudReq.SetHudInfo = true
			hudReq.SetHudHelp = true
			hudReq.SetHudPlayer = true
		}
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(hudReq))
		if err != nil {
			return false, err
		}
		onOff := "off"
		if c.Visible {
			onOff = "on"
		}
		r.printf("ok  hud %s  event_id=%s  status=%s\n", onOff, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.HUDList:
		perf, err := r.perfClient.GetPerformance(ctx, connect.NewRequest(&v1.GetPerformanceRequest{}))
		if err != nil {
			return false, err
		}
		s := perf.Msg.State
		onOff := func(b bool) string {
			if b {
				return "on"
			}
			return "off"
		}
		r.printf("HUD master: %s\n", onOff(s.HudVisible))
		r.printf("  debug:    %s\n", onOff(s.HudDebug))
		r.printf("  info:     %s\n", onOff(s.HudInfo))
		r.printf("  help:     %s\n", onOff(s.HudHelp))
		r.printf("  player:   %s (reserved)\n", onOff(s.HudPlayer))

	case commands.HUDCategory:
		var req v1.SetPerformanceRequest
		switch c.Category {
		case "debug":
			req = v1.SetPerformanceRequest{State: &v1.PerformanceState{HudDebug: c.Visible}, SetHudDebug: true}
		case "info":
			req = v1.SetPerformanceRequest{State: &v1.PerformanceState{HudInfo: c.Visible}, SetHudInfo: true}
		case "help":
			req = v1.SetPerformanceRequest{State: &v1.PerformanceState{HudHelp: c.Visible}, SetHudHelp: true}
		case "player":
			req = v1.SetPerformanceRequest{State: &v1.PerformanceState{HudPlayer: c.Visible}, SetHudPlayer: true}
		}
		if c.Visible {
			// Turning a category on implicitly re-enables the master so the
			// panel actually appears (e.g. after "hud off" + "hud info on").
			req.State.HudVisible = true
			req.SetHudVisible = true
		}
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(&req))
		if err != nil {
			return false, err
		}
		onOff := "off"
		if c.Visible {
			onOff = "on"
		}
		r.printf("ok  hud %s %s  event_id=%s  status=%s\n", c.Category, onOff, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.Labels:
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(&v1.SetPerformanceRequest{
			State:         &v1.PerformanceState{LabelsMode: c.Mode},
			SetLabelsMode: true,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  labels %s  event_id=%s  status=%s\n", c.Mode, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	case commands.Infra:
		resp, err := r.perfClient.SetPerformance(ctx, connect.NewRequest(&v1.SetPerformanceRequest{
			State:        &v1.PerformanceState{InfraMode: int32(c.Mode)},
			SetInfraMode: true,
		}))
		if err != nil {
			return false, err
		}
		r.printf("ok  infra %d  event_id=%s  status=%s\n", c.Mode, resp.Msg.Ack.GetEventId(), resp.Msg.Ack.GetStatus())

	}
	return false, nil
}
