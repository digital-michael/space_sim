// Package ui provides generic rendering-support and input-state types for
// Space Sim. It depends only on space/engine and the standard library —
// no Raylib types, no dataset-specific knowledge.
package ui

import (
	"math"
	"strings"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// CameraMode represents the active camera control mode.
type CameraMode int

const (
	CameraModeFree CameraMode = iota
	CameraModeJumping
	CameraModeTracking
)

// JumpTarget is a pre-resolved jump destination used in multi-hop sequences.
type JumpTarget struct {
	TargetIndex  int
	TargetPos    engine.Vector3
	ViewDist     float64
	DwellSeconds float64 // seconds to pause at this stop before jumping onward (0 = no pause)
}

// CameraState holds camera position, orientation, and animation state.
type CameraState struct {
	Position engine.Vector3
	Forward  engine.Vector3
	Up       engine.Vector3
	Yaw   float64
	Pitch float64
	Roll  float64 // accumulated roll around the forward axis (radians); 0 = world-up orientation
	Mode  CameraMode

	// Jump animation
	JumpStartPos    engine.Vector3
	JumpTargetPos   engine.Vector3
	JumpProgress    float64
	JumpDuration    float64
	JumpTargetIndex int

	// JumpQueue holds pending jump targets for multi-hop animated sequences.
	// After each jump completes the next entry is started automatically.
	JumpQueue []JumpTarget

	// JumpCurrentDwell is the dwell time (seconds) for the currently-executing hop.
	// Stored here so input.go can read it after UpdateJump marks the hop complete.
	JumpCurrentDwell float64

	// JumpDwellRemaining counts down after arrival before the next hop starts.
	// The camera stays in tracking mode during the dwell.
	JumpDwellRemaining float64

	// JumpTargetViewDist is the view-distance used for the active jump, carried
	// forward so the post-arrival tracking distance is set correctly.
	JumpTargetViewDist float64

	// JumpStartYaw/Pitch and JumpTargetYaw/Pitch are the camera angles at the
	// beginning and end of a jump, used to smoothly interpolate the look direction.
	JumpStartYaw    float64
	JumpStartPitch  float64
	JumpTargetYaw   float64
	JumpTargetPitch float64

	// Velocity is a persistent drift applied to the camera position every
	// frame (AU/s, free-fly mode only). Set to zero to stop.
	Velocity engine.Vector3

	// Tracking
	TrackTargetIndex int
	TrackDistance    float64
	TrackHeight      float64
	TrackOffset      engine.Vector3
	TrackYaw         float64
	TrackPitch       float64
	TrackLookOutward bool

	// Orbit animation
	OrbitSpeed            float64 // rad/sec; positive = counter-clockwise; 0 = not orbiting
	OrbitRadiansRemaining float64 // decrements each frame; orbit ends when ≤ 0

	// PendingOrbit holds orbit parameters to apply when an in-flight jump lands.
	// Non-zero PendingOrbitSpeed signals a pending orbit.
	PendingOrbitSpeed   float64
	PendingOrbitRadians float64
}

// NewCameraState creates a camera with sensible defaults.
func NewCameraState() *CameraState {
	return &CameraState{
		Position:      engine.Vector3{X: 0, Y: 50, Z: -100},
		Forward:       engine.Vector3{X: 0, Y: 0, Z: 1},
		Up:            engine.Vector3{X: 0, Y: 1, Z: 0},
		Yaw:           0,
		Pitch:         0,
		Mode:          CameraModeFree,
		TrackDistance: 50.0,
		TrackHeight:   20.0,
		TrackYaw:      math.Pi,
		TrackPitch:    0.3,
	}
}

// GetRight returns the right vector (Forward × Up, normalised).
func (c *CameraState) GetRight() engine.Vector3 {
	return c.Forward.Cross(c.Up).Normalize()
}

// CalculateAutoZoomDistance returns the camera distance that makes an object
// occupy screenPercent of screen height.
func CalculateAutoZoomDistance(objectRadius float32, screenPercent float32) float64 {
	fovRadians := engine.CameraFOV * (math.Pi / 180.0)
	tanHalfFOV := math.Tan(fovRadians / 2.0)
	distance := float64(objectRadius) / (float64(screenPercent) * tanHalfFOV)
	if distance < engine.CameraTrackDistMin {
		distance = engine.CameraTrackDistMin
	}
	if distance > engine.CameraTrackDistMax {
		distance = engine.CameraTrackDistMax
	}
	return distance
}

// UpdateForwardFromAngles recomputes the forward vector from yaw and pitch.
func (c *CameraState) UpdateForwardFromAngles() {
	c.Forward = engine.Vector3{
		X: float32(math.Cos(c.Pitch) * math.Sin(c.Yaw)),
		Y: float32(math.Sin(c.Pitch)),
		Z: float32(math.Cos(c.Pitch) * math.Cos(c.Yaw)),
	}
	c.Forward = c.Forward.Normalize()
	c.UpdateUpFromRoll()
}

// UpdateUpFromRoll recomputes the Up vector by rotating the natural-up direction
// (world +Y projected onto the plane perpendicular to Forward) by the current Roll
// angle around the Forward axis. Roll = 0 keeps the horizon aligned with world-up.
func (c *CameraState) UpdateUpFromRoll() {
	worldUp := engine.Vector3{Y: 1}
	dot := worldUp.Dot(c.Forward)
	baseUp := worldUp.Sub(c.Forward.Scale(dot))
	if baseUp.Length() < 0.001 {
		// Forward is nearly parallel to world-up (looking straight up or down).
		// Use +Z as a fallback to avoid degenerate up vectors.
		baseUp = engine.Vector3{Z: 1}
	} else {
		baseUp = baseUp.Normalize()
	}
	if c.Roll == 0 {
		c.Up = baseUp
		return
	}
	cosR := float32(math.Cos(c.Roll))
	sinR := float32(math.Sin(c.Roll))
	// Rodrigues rotation of baseUp around Forward by Roll radians.
	c.Up = baseUp.Scale(cosR).Add(c.Forward.Cross(baseUp).Scale(sinR)).Normalize()
}

// StartJumpTo initiates a smooth camera jump to a target object.
func (c *CameraState) StartJumpTo(targetIndex int, targetPos engine.Vector3, viewDistance float64) {
	c.Mode = CameraModeJumping
	c.JumpStartPos = c.Position
	c.JumpTargetIndex = targetIndex

	direction := c.Position.Sub(targetPos).Normalize()
	if direction.Length() < 0.1 {
		direction = engine.Vector3{X: 0, Y: 0, Z: -1}
	}
	c.JumpTargetPos = targetPos.Add(direction.Scale(float32(viewDistance)))
	c.JumpTargetPos.Y = c.JumpTargetPos.Y + float32(viewDistance*0.3)
	c.JumpProgress = 0.0

	// Scale duration by travel distance so long jumps feel traversed.
	// World coordinates: Earth = 100 units from Sol; sqrt(travel)*0.1 gives
	// ~1.5s for nearby hops and ~3s for system-crossing jumps. Clamped [1.5, 3.0].
	travel := float64(c.Position.Sub(c.JumpTargetPos).Length())
	c.JumpDuration = math.Max(1.5, math.Min(3.0, math.Sqrt(travel)*0.1))

	c.JumpTargetViewDist = viewDistance

	// Save start angles and compute the target look direction so UpdateJump can
	// smoothly interpolate the camera's facing over the duration of the jump.
	c.JumpStartYaw = c.Yaw
	c.JumpStartPitch = c.Pitch
	lookDir := targetPos.Sub(c.JumpTargetPos)
	if lookDir.Length() > 0.01 {
		lookDir = lookDir.Normalize()
		c.JumpTargetYaw = math.Atan2(float64(lookDir.X), float64(lookDir.Z))
		c.JumpTargetPitch = math.Asin(math.Max(-1.0, math.Min(1.0, float64(lookDir.Y))))
	} else {
		c.JumpTargetYaw = c.Yaw
		c.JumpTargetPitch = c.Pitch
	}
}

// UpdateJump advances the jump animation by dt seconds.
func (c *CameraState) UpdateJump(dt float64) {
	if c.Mode != CameraModeJumping {
		return
	}
	c.JumpProgress += dt / c.JumpDuration
	if c.JumpProgress >= 1.0 {
		c.Position = c.JumpTargetPos
		c.Yaw = c.JumpTargetYaw
		c.Pitch = c.JumpTargetPitch
		c.UpdateForwardFromAngles()
		c.Mode = CameraModeFree
		return
	}
	t := c.JumpProgress
	// Asymmetric easing: remap t through t^(2/3) before applying smoothstep.
	// The remap shifts the velocity peak to t≈0.37 so the camera spends ~37%
	// of the time accelerating and ~63% decelerating — arrival is a smooth
	// coast-in rather than a pop.
	tIn := math.Pow(t, 2.0/3.0) // t^(2/3) — ease-out remap
	smoothT := float32(tIn * tIn * (3.0 - 2.0*tIn))
	c.Position.X = c.JumpStartPos.X + smoothT*(c.JumpTargetPos.X-c.JumpStartPos.X)
	c.Position.Y = c.JumpStartPos.Y + smoothT*(c.JumpTargetPos.Y-c.JumpStartPos.Y)
	c.Position.Z = c.JumpStartPos.Z + smoothT*(c.JumpTargetPos.Z-c.JumpStartPos.Z)
	// Interpolate camera facing toward the destination over the same curve.
	// Wrap yaw delta into (-π, π) so the camera always takes the short arc.
	dyaw := c.JumpTargetYaw - c.JumpStartYaw
	for dyaw > math.Pi {
		dyaw -= 2 * math.Pi
	}
	for dyaw < -math.Pi {
		dyaw += 2 * math.Pi
	}
	c.Yaw = c.JumpStartYaw + float64(smoothT)*dyaw
	c.Pitch = c.JumpStartPitch + float64(smoothT)*(c.JumpTargetPitch-c.JumpStartPitch)
	c.UpdateForwardFromAngles()
}

// StartTracking locks the camera to track a specific object (orbital view).
func (c *CameraState) StartTracking(targetIndex int) {
	c.Mode = CameraModeTracking
	c.TrackTargetIndex = targetIndex
	c.TrackYaw = math.Pi
	c.TrackPitch = 0.3
	c.TrackLookOutward = false
}

// StartTrackingEquatorial locks the camera to track from the equatorial plane.
func (c *CameraState) StartTrackingEquatorial(targetIndex int) {
	c.Mode = CameraModeTracking
	c.TrackTargetIndex = targetIndex
	c.TrackYaw = math.Pi
	c.TrackPitch = 0.0
	c.TrackLookOutward = true
}

// UpdateTracking recomputes the camera position relative to the tracked object.
func (c *CameraState) UpdateTracking(state *engine.SimulationState) {
	if c.Mode != CameraModeTracking {
		return
	}
	if c.TrackTargetIndex < 0 || c.TrackTargetIndex >= len(state.Objects) {
		c.Mode = CameraModeFree
		return
	}

	target := state.Objects[c.TrackTargetIndex]
	// Clamp TrackDistance so the camera stays outside the target body surface.
	if minDist := float64(target.Meta.PhysicalRadius) + 0.5; c.TrackDistance < minDist {
		c.TrackDistance = minDist
	}
	x := float32(c.TrackDistance * math.Cos(c.TrackPitch) * math.Sin(c.TrackYaw))
	y := float32(c.TrackDistance * math.Sin(c.TrackPitch))
	z := float32(c.TrackDistance * math.Cos(c.TrackPitch) * math.Cos(c.TrackYaw))

	basePosition := target.Anim.Position.Add(engine.Vector3{X: x, Y: y, Z: z})
	c.Position = basePosition.Add(c.TrackOffset)

	if c.TrackLookOutward {
		var lookAtPos engine.Vector3
		if target.Meta.ParentName != "" {
			if parent := state.GetObject(target.Meta.ParentName); parent != nil {
				lookAtPos = parent.Anim.Position
			}
		}
		toLookAt := lookAtPos.Sub(c.Position)
		if toLookAt.Length() > 0.1 {
			c.Forward = toLookAt.Normalize()
		} else {
			c.Forward = c.Position.Sub(target.Anim.Position).Normalize()
		}
	} else {
		c.Forward = target.Anim.Position.Sub(c.Position).Normalize()
	}

	c.Yaw = math.Atan2(float64(c.Forward.X), float64(c.Forward.Z))
	c.Pitch = math.Asin(float64(c.Forward.Y))
}

// StopTracking returns to free-fly mode.
func (c *CameraState) StopTracking() {
	c.Mode = CameraModeFree
}

// HUDState holds per-category visibility for the heads-up display.
// All categories default to true; setting a category false hides only that section.
type HUDState struct {
	Debug  bool // upper-left stats block + lower-left screen/render lines
	Info   bool // lower-right tracking info + selection UI
	Help   bool // bottom-left hint bar ("Ctrl+/ for help …")
	Player bool // reserved — not yet rendered
}

// AllOnHUD returns a HUDState with all implemented categories enabled.
func AllOnHUD() HUDState {
	return HUDState{Debug: true, Info: true, Help: true, Player: false}
}

// AllOffHUD returns a HUDState with all categories disabled.
func AllOffHUD() HUDState {
	return HUDState{Debug: false, Info: false, Help: false, Player: false}
}

// LabelMode controls how object labels are displayed.
type LabelMode int

const (
	LabelModeOff     LabelMode = iota // no labels
	LabelModeOn                       // all eligible objects (up to 20 highest-priority)
	LabelModeNearest                  // only nearby objects + tracked/jump target
)

// LabelModeFromString parses a label mode string ("on", "off", "nearest").
// Unknown strings return LabelModeOff.
func LabelModeFromString(s string) LabelMode {
	switch strings.ToLower(s) {
	case "on":
		return LabelModeOn
	case "nearest":
		return LabelModeNearest
	default:
		return LabelModeOff
	}
}

// String returns the canonical string representation of the mode.
func (m LabelMode) String() string {
	switch m {
	case LabelModeOn:
		return "on"
	case LabelModeNearest:
		return "nearest"
	default:
		return "off"
	}
}

// AdjustTrackAngles adjusts the camera orbit angles (mouse/scroll input).
func (c *CameraState) AdjustTrackAngles(deltaYaw, deltaPitch float64) {
	c.TrackYaw += deltaYaw
	c.TrackPitch += deltaPitch
	if c.TrackPitch > math.Pi/2.0-0.01 {
		c.TrackPitch = math.Pi/2.0 - 0.01
	}
	if c.TrackPitch < -math.Pi/2.0+0.01 {
		c.TrackPitch = -math.Pi/2.0 + 0.01
	}
}
