package render

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// DumpLabelDebug writes a full label-pipeline diagnostic to path.
// It captures a snapshot of all inputs to DrawObjectLabels and traces every
// object through selection, projection, and culling — making units explicit
// so any mismatch between AU and pixel space is immediately visible.
func (r *Renderer) DumpLabelDebug(
	state *engine.SimulationState,
	cameraState *ui.CameraState,
	camera rl.Camera3D,
	objectsToRender []*engine.Object,
	mode ui.LabelMode,
	screenW, screenH int32,
	path string,
) {
	dumpLabelDebug(state, cameraState, camera, objectsToRender, mode, screenW, screenH, path)
}

func cameraModeStr(m ui.CameraMode) string {
	switch m {
	case ui.CameraModeFree:
		return "free"
	case ui.CameraModeTracking:
		return "tracking"
	case ui.CameraModeJumping:
		return "jumping"
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

func dumpLabelDebug(
	state *engine.SimulationState,
	cameraState *ui.CameraState,
	camera rl.Camera3D,
	objectsToRender []*engine.Object,
	mode ui.LabelMode,
	screenW, screenH int32,
	path string,
) {
	var buf strings.Builder
	buf.WriteString("=== Space Sim Label Debug Dump ===\n")
	buf.WriteString(fmt.Sprintf("Timestamp:  %s\n\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))

	// --- Label Settings ---
	nearestThreshold := 10.0
	if mode == ui.LabelModeNearest && cameraState.Mode == ui.CameraModeTracking && cameraState.Tracking.Distance > 0 {
		nearestThreshold = math.Max(10.0, cameraState.Tracking.Distance*3.0)
	}
	buf.WriteString("--- Label Settings ---\n")
	buf.WriteString(fmt.Sprintf("  LabelMode:      %s\n", mode.String()))
	buf.WriteString(fmt.Sprintf("  ScreenW:        %d px\n", screenW))
	buf.WriteString(fmt.Sprintf("  ScreenH:        %d px\n", screenH))
	buf.WriteString(fmt.Sprintf("  CameraFOV:      %.1f deg\n", float64(engine.CameraFOV)))
	buf.WriteString(fmt.Sprintf("  NearPlane:      %.4f AU\n", float64(engine.CameraNearPlane)))
	buf.WriteString(fmt.Sprintf("  FarPlane:       %.1f AU\n", float64(engine.CameraFarPlane)))
	if mode == ui.LabelModeNearest {
		buf.WriteString(fmt.Sprintf("  Nearest thresh: %.6f AU\n", nearestThreshold))
		buf.WriteString("  MaxLabels:      5\n")
	} else if mode == ui.LabelModeOn {
		buf.WriteString("  MaxLabels:      20\n")
	}

	// --- Camera State ---
	buf.WriteString("\n--- Camera State ---\n")
	buf.WriteString("  UNITS: position in AU; yaw/pitch in degrees\n")
	buf.WriteString(fmt.Sprintf("  Mode:           %s\n", cameraModeStr(cameraState.Mode)))
	buf.WriteString(fmt.Sprintf("  Position (AU):  x=%.8f  y=%.8f  z=%.8f\n",
		cameraState.Position.X, cameraState.Position.Y, cameraState.Position.Z))
	buf.WriteString(fmt.Sprintf("  Yaw:            %.4f deg\n", cameraState.Yaw*(180.0/math.Pi)))
	buf.WriteString(fmt.Sprintf("  Pitch:          %.4f deg\n", cameraState.Pitch*(180.0/math.Pi)))
	buf.WriteString(fmt.Sprintf("  Raylib cam pos: (%.6f, %.6f, %.6f)  [floating-origin; expect ~0,0,0]\n",
		camera.Position.X, camera.Position.Y, camera.Position.Z))
	buf.WriteString(fmt.Sprintf("  Raylib cam tgt: (%.6f, %.6f, %.6f)\n",
		camera.Target.X, camera.Target.Y, camera.Target.Z))
	if cameraState.Mode == ui.CameraModeTracking {
		buf.WriteString(fmt.Sprintf("  Track idx:      %d\n", cameraState.Tracking.TargetIndex))
		buf.WriteString(fmt.Sprintf("  Track dist:     %.8f AU\n", cameraState.Tracking.Distance))
		if cameraState.Tracking.TargetIndex >= 0 && cameraState.Tracking.TargetIndex < len(state.Objects) {
			buf.WriteString(fmt.Sprintf("  Track name:     %s\n",
				state.Objects[cameraState.Tracking.TargetIndex].Meta.Name))
		}
	}

	// --- Object Counts ---
	eligible := 0
	for _, obj := range objectsToRender {
		if obj.Meta.Category != engine.CategoryAsteroid &&
			obj.Meta.Category != engine.CategoryRing &&
			obj.Meta.Category != engine.CategoryBelt {
			eligible++
		}
	}
	buf.WriteString(fmt.Sprintf("\n--- Objects to render: %d total ---\n", len(objectsToRender)))
	buf.WriteString(fmt.Sprintf("  Eligible for labels (non-asteroid/ring/belt): %d\n", eligible))

	// --- Label Selection ---
	labeled := selectObjectsForLabels(state, cameraState, objectsToRender, mode)
	buf.WriteString(fmt.Sprintf("\n--- Label candidates selected: %d ---\n", len(labeled)))
	labeledSet := make(map[*engine.Object]struct{}, len(labeled))
	for _, obj := range labeled {
		labeledSet[obj] = struct{}{}
	}

	// --- Per-Object Detail ---
	camPos := cameraState.Position
	buf.WriteString("\n--- Per-Object Detail ---\n")
	buf.WriteString("  UNITS: world pos and cam-relative pos in AU; screen pos in px\n")
	buf.WriteString(fmt.Sprintf("  cam.Position = (%.8f, %.8f, %.8f) AU  [subtracted before projection]\n",
		camPos.X, camPos.Y, camPos.Z))

	for _, obj := range objectsToRender {
		isFiltered := obj.Meta.Category == engine.CategoryAsteroid ||
			obj.Meta.Category == engine.CategoryRing ||
			obj.Meta.Category == engine.CategoryBelt

		objPosX := obj.Anim.Position.X - camPos.X
		objPosY := obj.Anim.Position.Y - camPos.Y
		objPosZ := obj.Anim.Position.Z - camPos.Z
		objPos := rl.Vector3{X: objPosX, Y: objPosY, Z: objPosZ}
		dist := math.Sqrt(float64(objPosX*objPosX + objPosY*objPosY + objPosZ*objPosZ))

		// Compute projection upfront so label pos can be shown right after world pos.
		var screenPos rl.Vector2
		var inFront bool
		if !isFiltered {
			screenPos, inFront = projectToScreen(objPos, camera, screenW, screenH)
		}

		_, isLabeled := labeledSet[obj]
		var status string
		switch {
		case isFiltered:
			status = "SKIP(category)"
		case isLabeled:
			status = "LABELED"
		default:
			status = "CANDIDATE(not selected)"
		}

		buf.WriteString(fmt.Sprintf("\n  [%s] %s  cat=%v  importance=%d\n",
			status, obj.Meta.Name, obj.Meta.Category, obj.Meta.Importance))
		buf.WriteString(fmt.Sprintf("    World pos (AU):    x=%.8f  y=%.8f  z=%.8f\n",
			obj.Anim.Position.X, obj.Anim.Position.Y, obj.Anim.Position.Z))
		if isLabeled && inFront {
			labelOffX := float32(scaledInt32(15))
			labelOffY := float32(scaledInt32(10))
			buf.WriteString(fmt.Sprintf("    Label pos (px):    x≈%.2f  y≈%.2f  [approx; pre-clamp/pre-measure]\n",
				screenPos.X+labelOffX, screenPos.Y-labelOffY))
		} else if !isFiltered {
			buf.WriteString("    Label pos (px):    N/A\n")
		}
		buf.WriteString(fmt.Sprintf("    Cam-relative (AU): x=%.8f  y=%.8f  z=%.8f  dist=%.8f AU\n",
			objPosX, objPosY, objPosZ, dist))
		if !isFiltered {
			buf.WriteString(fmt.Sprintf("    inFront:           %v\n", inFront))
			if inFront {
				buf.WriteString(fmt.Sprintf("    Screen pos (px):   x=%.2f  y=%.2f  [object projected pos; label offset above]\n", screenPos.X, screenPos.Y))
				offLeft := screenPos.X < -1000
				offRight := screenPos.X > float32(screenW+1000)
				offTop := screenPos.Y < -1000
				offBottom := screenPos.Y > float32(screenH+1000)
				culled := offLeft || offRight || offTop || offBottom
				if culled {
					buf.WriteString(fmt.Sprintf("    Off-screen culled: true  (left=%v right=%v top=%v bottom=%v)\n",
						offLeft, offRight, offTop, offBottom))
				} else {
					buf.WriteString("    Off-screen culled: false\n")
				}
			}
		}
	}

	// --- Labeled Set Summary ---
	buf.WriteString(fmt.Sprintf("\n--- Labeled Set (%d objects) ---\n", len(labeled)))
	for i, obj := range labeled {
		objPosX := obj.Anim.Position.X - camPos.X
		objPosY := obj.Anim.Position.Y - camPos.Y
		objPosZ := obj.Anim.Position.Z - camPos.Z
		objPos := rl.Vector3{X: objPosX, Y: objPosY, Z: objPosZ}
		screenPos, inFront := projectToScreen(objPos, camera, screenW, screenH)
		buf.WriteString(fmt.Sprintf("  %d. %-20s  inFront=%-5v  screen=(%.1f, %.1f) px\n",
			i+1, obj.Meta.Name, inFront, screenPos.X, screenPos.Y))
	}

	if err := os.WriteFile(path, []byte(buf.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "label debug dump: %v\n", err)
	}
}
