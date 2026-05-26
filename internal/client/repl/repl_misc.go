package repl

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

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

	case commands.DebugLabels:
		return false, r.runDebugLabels(ctx)

	}
	return false, nil
}

func (r *REPL) runDebugLabels(ctx context.Context) error {
	snap, err := r.oneSnapshot(ctx)
	if err != nil {
		return err
	}
	camResp, err := r.camClient.GetCamera(ctx, connect.NewRequest(&v1.GetCameraRequest{}))
	if err != nil {
		return err
	}
	perfResp, err := r.perfClient.GetPerformance(ctx, connect.NewRequest(&v1.GetPerformanceRequest{}))
	if err != nil {
		return err
	}

	cam := camResp.Msg.GetCamera()
	labelsMode := perfResp.Msg.State.GetLabelsMode()
	camX, camY, camZ := cam.GetPosX(), cam.GetPosY(), cam.GetPosZ()

	var buf strings.Builder
	buf.WriteString("=== Space Sim Label Debug Dump (REPL) ===\n")
	buf.WriteString(fmt.Sprintf("Timestamp:  %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))
	buf.WriteString("NOTE: REPL snapshot \u2014 no screen projection (use in-app CTRL+\\ for full dump)\n")
	buf.WriteString("NOTE: Importance scores not available via gRPC snapshot; priority uses distance only.\n\n")

	// --- Label Settings ---
	const defaultNearestThreshold = 10.0
	buf.WriteString("--- Label Settings ---\n")
	buf.WriteString(fmt.Sprintf("  LabelMode:      %s  [server-reported]\n", labelsMode))
	switch labelsMode {
	case "nearest":
		buf.WriteString(fmt.Sprintf("  Nearest thresh: %.1f AU  [default; tracking dist not in snapshot]\n", defaultNearestThreshold))
		buf.WriteString("  MaxLabels:      5\n")
	case "on":
		buf.WriteString("  MaxLabels:      20\n")
	}

	// --- Camera State ---
	buf.WriteString("\n--- Camera State ---\n")
	buf.WriteString("  UNITS: position in AU; yaw/pitch in degrees\n")
	buf.WriteString(fmt.Sprintf("  Mode:           %s\n", cam.GetMode()))
	buf.WriteString(fmt.Sprintf("  Position (AU):  x=%.8f  y=%.8f  z=%.8f\n", camX, camY, camZ))
	buf.WriteString(fmt.Sprintf("  Yaw:            %.4f deg\n", cam.GetYawDeg()))
	buf.WriteString(fmt.Sprintf("  Pitch:          %.4f deg\n", cam.GetPitchDeg()))
	if t := cam.GetTrackTarget(); t != "" {
		buf.WriteString(fmt.Sprintf("  Track target:   %s\n", t))
	}

	// --- Build candidates ---
	type candidate struct {
		body *v1.BodyState
		dist float64
	}
	eligible := make([]candidate, 0, len(snap.Bodies))
	skipped := 0
	for _, body := range snap.Bodies {
		cat := body.GetCategory()
		if cat == "asteroid" || cat == "ring" || cat == "belt" {
			skipped++
			continue
		}
		dx := body.GetPosX() - camX
		dy := body.GetPosY() - camY
		dz := body.GetPosZ() - camZ
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		eligible = append(eligible, candidate{body: body, dist: dist})
	}
	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].dist < eligible[j].dist
	})

	buf.WriteString(fmt.Sprintf("\n--- Bodies: %d total ---\n", len(snap.Bodies)))
	buf.WriteString(fmt.Sprintf("  Skipped (asteroid/ring/belt): %d\n", skipped))
	buf.WriteString(fmt.Sprintf("  Eligible for labels:          %d\n", len(eligible)))

	// --- Apply mode filter ---
	var selected []candidate
	trackTarget := cam.GetTrackTarget()
	switch labelsMode {
	case "nearest":
		for _, c := range eligible {
			isTarget := trackTarget != "" && c.body.GetName() == trackTarget
			if isTarget || c.dist <= defaultNearestThreshold {
				selected = append(selected, c)
			}
		}
		if len(selected) > 5 {
			selected = selected[:5]
		}
	case "on":
		selected = eligible
		if len(selected) > 20 {
			selected = selected[:20]
		}
	}
	buf.WriteString(fmt.Sprintf("  Selected for labels:          %d\n", len(selected)))

	// Build selected set for status lookup
	selectedSet := make(map[string]struct{}, len(selected))
	for _, c := range selected {
		selectedSet[c.body.GetName()] = struct{}{}
	}

	// --- Per-body detail ---
	buf.WriteString("\n--- Body Detail ---\n")
	buf.WriteString("  UNITS: positions in AU\n")
	buf.WriteString(fmt.Sprintf("  cam.Position = (%.8f, %.8f, %.8f) AU\n", camX, camY, camZ))
	for _, c := range eligible {
		name := c.body.GetName()
		cat := c.body.GetCategory()
		isTrackTarget := trackTarget != "" && name == trackTarget
		_, isSel := selectedSet[name]

		var status string
		if isSel {
			status = "LABELED"
		} else {
			switch labelsMode {
			case "nearest":
				if isTrackTarget {
					status = "CANDIDATE(over max 5)"
				} else {
					status = fmt.Sprintf("FILTERED(nearest: %.4f AU > %.1f AU threshold)", c.dist, defaultNearestThreshold)
				}
			case "on":
				status = "CANDIDATE(over max 20)"
			default:
				status = "FILTERED(labels off)"
			}
		}

		buf.WriteString(fmt.Sprintf("\n  [%s] %s  cat=%s\n", status, name, cat))
		buf.WriteString(fmt.Sprintf("    World pos (AU):    x=%.8f  y=%.8f  z=%.8f\n",
			c.body.GetPosX(), c.body.GetPosY(), c.body.GetPosZ()))
		buf.WriteString(fmt.Sprintf("    Cam-relative (AU): dx=%.8f  dy=%.8f  dz=%.8f  dist=%.8f AU\n",
			c.body.GetPosX()-camX, c.body.GetPosY()-camY, c.body.GetPosZ()-camZ, c.dist))
		if isTrackTarget {
			buf.WriteString("    ** TRACKED TARGET **\n")
		}
	}

	// --- Label set summary ---
	buf.WriteString(fmt.Sprintf("\n--- Label Set (%d objects) ---\n", len(selected)))
	for i, c := range selected {
		buf.WriteString(fmt.Sprintf("  %d. %-20s  dist=%.6f AU  track=%v\n",
			i+1, c.body.GetName(), c.dist, c.body.GetName() == trackTarget))
	}

	if err := os.WriteFile("debug.log", []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("debug labels: %w", err)
	}
	r.printf("debug.log written  (%d bodies, %d eligible, %d labeled)\n",
		len(snap.Bodies), len(eligible), len(selected))
	return nil
}
