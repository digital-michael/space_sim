package render

import (
	"hash/fnv"
	"math"

	"github.com/digital-michael/space_sim/internal/protocol"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Marker rendering constants (F-021 Phase 1).
// Configurable via configs/app.json in a future pass; values below match the
// spec defaults (see docs/wip/f021-physical-marker-spec.md §4).
const (
	markerBlinkPeriodS       = 1.5   // seconds per full blink cycle
	markerNearRadius         = 0.001 // world-space radius when camera is close (sim units)
	markerFarThresholdSU     = 1.0   // camera distance (sim units) above which screen-space minimum is enforced
	markerCullDistanceSU     = 100.0 // markers beyond this distance (sim units) are not drawn
	markerScreenMinPx        = 4.0   // minimum screen-space pixel radius at any distance
	markerOwnOpacity         = 0.30  // opacity multiplier applied to the local client's own marker
	markerLabelVisibleDistSU = 0.5   // labels are hidden when the camera is farther than this (sim units)
)

// markerPhaseOffset returns a deterministic per-session blink phase offset
// in [0, 2π) derived from the FNV-1a hash of the session ID.
// Different sessions receive different offsets so their blink cycles are
// visually desynchronised even when they share the same blink period.
func markerPhaseOffset(sessionID string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(sessionID))
	return float64(h.Sum32()%10000) / 10000.0 * 2 * math.Pi
}

// markerBlinkAlpha returns the per-frame blink alpha in [0, 1] for a given
// session and elapsed time t (seconds).
//
//	alpha = 0.5 + 0.5 * sin(t * (2π / period) + phaseOffset)
func markerBlinkAlpha(sessionID string, t float64) float64 {
	return 0.5 + 0.5*math.Sin(t*(2*math.Pi/markerBlinkPeriodS)+markerPhaseOffset(sessionID))
}

// markerSphereRadius computes the world-space sphere radius to use when
// drawing a marker at distFromCam sim units from the camera.
//
// Near-camera (dist ≤ markerFarThresholdSU): markerNearRadius is used.
// Far-camera (dist > markerFarThresholdSU): the radius is expanded so the
// sphere subtends at least markerScreenMinPx pixels on screen — the same
// screen-space minimum strategy used for distant star rendering.
//
// Derivation:
//
//	pixelRadius = (worldRadius / dist) * screenH / (2 * tan(FOV/2))
//	worldRadius = pixelRadius * dist * 2 * tan(FOV/2) / screenH
func markerSphereRadius(distFromCam float64, screenH int) float64 {
	r := markerNearRadius
	if distFromCam > markerFarThresholdSU && screenH > 0 {
		halfFovRad := engine.CameraFOV * math.Pi / 180.0 / 2.0
		minWorld := markerScreenMinPx * distFromCam * 2.0 * math.Tan(halfFovRad) / float64(screenH)
		if minWorld > r {
			r = minWorld
		}
	}
	return r
}

// DrawClientMarkers draws a blinking sphere in world space for each session in
// sessions. Must be called inside BeginMode3D / EndMode3D.
//
// All positions are expressed in the floating-origin frame: each marker world
// position is offset by -camPos before drawing, matching the DEF-001 pattern
// used for planet and star rendering.
//
// ownSessionID is the local client's session ID; that marker is rendered at
// markerOwnOpacity to avoid obscuring the player's own view. Pass "" if the
// session ID is not known (all markers render at full blink alpha).
//
// t is elapsed time in seconds; use float64(rl.GetTime()).
func (r *Renderer) DrawClientMarkers(
	sessions []protocol.ClientSessionSnapshot,
	camPos engine.Vector3,
	ownSessionID string,
	t float64,
) {
	if len(sessions) == 0 {
		return
	}

	screenH := currentScreenHeight()

	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DisableDepthMask()

	for _, s := range sessions {
		relX := s.Position[0] - float64(camPos.X)
		relY := s.Position[1] - float64(camPos.Y)
		relZ := s.Position[2] - float64(camPos.Z)
		dist := math.Sqrt(relX*relX + relY*relY + relZ*relZ)

		if dist > markerCullDistanceSU {
			continue
		}

		alpha := markerBlinkAlpha(s.SessionID, t)
		if ownSessionID != "" && s.SessionID == ownSessionID {
			alpha *= markerOwnOpacity
		}

		color := rl.Color{
			R: s.Color[0],
			G: s.Color[1],
			B: s.Color[2],
			A: uint8(alpha * 255),
		}
		pos := rl.Vector3{X: float32(relX), Y: float32(relY), Z: float32(relZ)}
		radius := float32(markerSphereRadius(dist, screenH))
		rl.DrawSphere(pos, radius, color)
	}

	rl.EnableDepthMask()
	rl.EndBlendMode()
}

// DrawClientMarkerLabels draws a 2D text label above each nearby session marker.
// Must be called OUTSIDE BeginMode3D / EndMode3D (2D screen-space pass).
// camera is the Raylib Camera3D used for 3D→2D projection via GetWorldToScreenEx.
//
// Labels are skipped for sessions farther than markerLabelVisibleDistSU from
// the camera and for sessions with an empty Label field.
func (r *Renderer) DrawClientMarkerLabels(
	sessions []protocol.ClientSessionSnapshot,
	camPos engine.Vector3,
	ownSessionID string,
	camera rl.Camera3D,
) {
	if len(sessions) == 0 {
		return
	}

	screenW := int32(currentScreenWidth())
	screenH := int32(currentScreenHeight())
	fontSize := scaledInt32(14)
	labelRiseY := scaledInt32(24) // pixels above projected center

	for _, s := range sessions {
		if s.Label == "" {
			continue
		}

		relX := s.Position[0] - float64(camPos.X)
		relY := s.Position[1] - float64(camPos.Y)
		relZ := s.Position[2] - float64(camPos.Z)
		dist := math.Sqrt(relX*relX + relY*relY + relZ*relZ)

		if dist > markerLabelVisibleDistSU {
			continue
		}

		pos3D := rl.Vector3{X: float32(relX), Y: float32(relY), Z: float32(relZ)}
		screen := rl.GetWorldToScreenEx(pos3D, camera, screenW, screenH)

		// Skip if the projected point is far off-screen (behind camera or clipped).
		if screen.X < -500 || screen.X > float32(screenW+500) ||
			screen.Y < -500 || screen.Y > float32(screenH+500) {
			continue
		}

		label := s.Label
		if ownSessionID != "" && s.SessionID == ownSessionID {
			label += " (you)"
		}

		color := rl.Color{R: s.Color[0], G: s.Color[1], B: s.Color[2], A: 255}
		textW := rl.MeasureText(label, fontSize)
		drawX := int32(screen.X) - textW/2
		drawY := int32(screen.Y) - labelRiseY
		rl.DrawText(label, drawX, drawY, fontSize, color)
	}
}
