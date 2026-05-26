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

// TrackingState holds all fields that are only valid in CameraModeTracking.
// Assigned as a whole by StartTracking / StartTrackingEquatorial so every
// field is reset by construction on each target change.
type TrackingState struct {
	TargetIndex  int
	Distance     float64
	Height       float64
	Yaw          float64
	Pitch        float64
	Offset       engine.Vector3
	LookOutward  bool
}

// JumpState holds all fields that are only valid during a CameraModeJumping
// animation. Assigned as a whole by StartJumpTo so every field is reset by
// construction on each new jump.
type JumpState struct {
	TargetIndex    int
	StartPos       engine.Vector3
	TargetPos      engine.Vector3
	Progress       float64
	Duration       float64
	Queue          []JumpTarget
	CurrentDwell   float64
	DwellRemaining float64
	TargetViewDist float64
	StartYaw       float64
	StartPitch     float64
	TargetYaw      float64
	TargetPitch    float64
}

// CameraState holds camera position, orientation, and animation state.
type CameraState struct {
	Position engine.Vector3
	Forward  engine.Vector3
	Up       engine.Vector3
	Yaw      float64
	Pitch    float64
	Roll     float64 // accumulated roll around the forward axis (radians); 0 = world-up orientation
	Mode     CameraMode

	// Jump holds all state for the current jump animation (only valid when
	// Mode == CameraModeJumping). StartJumpTo assigns a fresh JumpState so
	// every field is reset by construction.
	Jump JumpState

	// Velocity is a persistent drift applied to the camera position every
	// frame (AU/s, free-fly mode only). Set to zero to stop.
	Velocity engine.Vector3

	// Tracking holds all state for tracking a specific object (only valid when
	// Mode == CameraModeTracking). StartTracking / StartTrackingEquatorial
	// assign a fresh TrackingState so every field is reset by construction.
	Tracking TrackingState

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
		Position: engine.Vector3{X: 0, Y: 50, Z: -100},
		Forward:  engine.Vector3{X: 0, Y: 0, Z: 1},
		Up:       engine.Vector3{X: 0, Y: 1, Z: 0},
		Yaw:      0,
		Pitch:    0,
		Mode:     CameraModeFree,
		Tracking: TrackingState{
			Distance: 50.0,
			Height:   20.0,
			Yaw:      math.Pi,
			Pitch:    0.3,
		},
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

// FaceTarget rotates the camera to look at targetPos without moving its position.
// Exits tracking mode if active so the orientation change is not overridden.
func (c *CameraState) FaceTarget(targetPos engine.Vector3) {
	if c.Mode == CameraModeTracking {
		c.StopTracking()
	}
	lookDir := targetPos.Sub(c.Position)
	if lookDir.Length() < 0.01 {
		return
	}
	lookDir = lookDir.Normalize()
	c.Yaw = math.Atan2(float64(lookDir.X), float64(lookDir.Z))
	c.Pitch = math.Asin(math.Max(-1.0, math.Min(1.0, float64(lookDir.Y))))
	c.UpdateForwardFromAngles()
}

// StartJumpTo initiates a smooth camera jump to a target object.
func (c *CameraState) StartJumpTo(targetIndex int, targetPos engine.Vector3, viewDistance float64) {
	c.Mode = CameraModeJumping
	c.Jump.StartPos = c.Position
	c.Jump.TargetIndex = targetIndex

	direction := c.Position.Sub(targetPos).Normalize()
	if direction.Length() < 0.1 {
		direction = engine.Vector3{X: 0, Y: 0, Z: -1}
	}
	c.Jump.TargetPos = targetPos.Add(direction.Scale(float32(viewDistance)))
	c.Jump.TargetPos.Y = c.Jump.TargetPos.Y + float32(viewDistance*0.3)
	c.Jump.Progress = 0.0

	// Scale duration by travel distance so long jumps feel traversed.
	// World coordinates: Earth = 100 units from Sol; sqrt(travel)*0.1 gives
	// ~1.5s for nearby hops and ~3s for system-crossing jumps. Clamped [1.5, 3.0].
	travel := float64(c.Position.Sub(c.Jump.TargetPos).Length())
	c.Jump.Duration = math.Max(1.5, math.Min(3.0, math.Sqrt(travel)*0.1))

	c.Jump.TargetViewDist = viewDistance

	// Save start angles and compute the target look direction so UpdateJump can
	// smoothly interpolate the camera's facing over the duration of the jump.
	c.Jump.StartYaw = c.Yaw
	c.Jump.StartPitch = c.Pitch
	lookDir := targetPos.Sub(c.Jump.TargetPos)
	if lookDir.Length() > 0.01 {
		lookDir = lookDir.Normalize()
		c.Jump.TargetYaw = math.Atan2(float64(lookDir.X), float64(lookDir.Z))
		c.Jump.TargetPitch = math.Asin(math.Max(-1.0, math.Min(1.0, float64(lookDir.Y))))
	} else {
		c.Jump.TargetYaw = c.Yaw
		c.Jump.TargetPitch = c.Pitch
	}
}

// UpdateJump advances the jump animation by dt seconds.
func (c *CameraState) UpdateJump(dt float64) {
	if c.Mode != CameraModeJumping {
		return
	}
	c.Jump.Progress += dt / c.Jump.Duration
	if c.Jump.Progress >= 1.0 {
		c.Position = c.Jump.TargetPos
		c.Yaw = c.Jump.TargetYaw
		c.Pitch = c.Jump.TargetPitch
		c.UpdateForwardFromAngles()
		c.Mode = CameraModeFree
		return
	}
	t := c.Jump.Progress
	// Asymmetric easing: remap t through t^(2/3) before applying smoothstep.
	// The remap shifts the velocity peak to t≈0.37 so the camera spends ~37%
	// of the time accelerating and ~63% decelerating — arrival is a smooth
	// coast-in rather than a pop.
	tIn := math.Pow(t, 2.0/3.0) // t^(2/3) — ease-out remap
	smoothT := float32(tIn * tIn * (3.0 - 2.0*tIn))
	c.Position.X = c.Jump.StartPos.X + smoothT*(c.Jump.TargetPos.X-c.Jump.StartPos.X)
	c.Position.Y = c.Jump.StartPos.Y + smoothT*(c.Jump.TargetPos.Y-c.Jump.StartPos.Y)
	c.Position.Z = c.Jump.StartPos.Z + smoothT*(c.Jump.TargetPos.Z-c.Jump.StartPos.Z)
	// Interpolate camera facing toward the destination over the same curve.
	// Wrap yaw delta into (-π, π) so the camera always takes the short arc.
	dyaw := c.Jump.TargetYaw - c.Jump.StartYaw
	for dyaw > math.Pi {
		dyaw -= 2 * math.Pi
	}
	for dyaw < -math.Pi {
		dyaw += 2 * math.Pi
	}
	c.Yaw = c.Jump.StartYaw + float64(smoothT)*dyaw
	c.Pitch = c.Jump.StartPitch + float64(smoothT)*(c.Jump.TargetPitch-c.Jump.StartPitch)
	c.UpdateForwardFromAngles()
}

// StartTracking locks the camera to track a specific object (orbital view).
func (c *CameraState) StartTracking(targetIndex int) {
	c.Mode = CameraModeTracking
	c.Tracking.TargetIndex = targetIndex
	c.Tracking.Yaw = math.Pi
	c.Tracking.Pitch = 0.3
	c.Tracking.LookOutward = false
	c.Tracking.Offset = engine.Vector3{} // reset accumulated WASD offset on every target change
}

// StartTrackingEquatorial locks the camera to track from the equatorial plane.
func (c *CameraState) StartTrackingEquatorial(targetIndex int) {
	c.Mode = CameraModeTracking
	c.Tracking.TargetIndex = targetIndex
	c.Tracking.Yaw = math.Pi
	c.Tracking.Pitch = 0.0
	c.Tracking.LookOutward = true
	c.Tracking.Offset = engine.Vector3{} // reset accumulated WASD offset on every target change
}

// UpdateTracking recomputes the camera position relative to the tracked object.
func (c *CameraState) UpdateTracking(state *engine.SimulationState) {
	if c.Mode != CameraModeTracking {
		return
	}
	if c.Tracking.TargetIndex < 0 || c.Tracking.TargetIndex >= len(state.Objects) {
		c.Mode = CameraModeFree
		return
	}

	target := state.Objects[c.Tracking.TargetIndex]
	// Clamp TrackDistance so the camera stays outside the target body surface.
	if minDist := float64(target.Meta.PhysicalRadius) + 0.5; c.Tracking.Distance < minDist {
		c.Tracking.Distance = minDist
	}
	x := float32(c.Tracking.Distance * math.Cos(c.Tracking.Pitch) * math.Sin(c.Tracking.Yaw))
	y := float32(c.Tracking.Distance * math.Sin(c.Tracking.Pitch))
	z := float32(c.Tracking.Distance * math.Cos(c.Tracking.Pitch) * math.Cos(c.Tracking.Yaw))

	basePosition := target.Anim.Position.Add(engine.Vector3{X: x, Y: y, Z: z})
	c.Position = basePosition.Add(c.Tracking.Offset)

	if c.Tracking.LookOutward {
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
	c.Tracking.Yaw += deltaYaw
	c.Tracking.Pitch += deltaPitch
	if c.Tracking.Pitch > math.Pi/2.0-0.01 {
		c.Tracking.Pitch = math.Pi/2.0 - 0.01
	}
	if c.Tracking.Pitch < -math.Pi/2.0+0.01 {
		c.Tracking.Pitch = -math.Pi/2.0 + 0.01
	}
}

// TickTracking processes one frame of tracking-mode physics.
// wasdDelta is the pre-scaled world-space offset delta from WASD input (computed from
// the previous frame's Forward/Right — 1-frame lag is imperceptible at interactive rates).
// arrowDelta is the pre-scaled world-axis delta from arrow keys.
// mouseYaw/mousePitch are the angular delta in radians (signed, pre-scaled by sensitivity).
// zoomDelta is the signed Tracking.Distance change (positive = zoom out); 0 = no zoom.
// resetOffset clears the WASD-accumulated offset when true.
func (c *CameraState) TickTracking(state *engine.SimulationState, dt float32,
	wasdDelta, arrowDelta engine.Vector3,
	mouseYaw, mousePitch, zoomDelta float64,
	resetOffset bool,
) {
	// Zoom (applied before mode-specific angle updates to match original pre-switch order).
	if zoomDelta != 0 {
		c.Tracking.Distance += zoomDelta
		if c.Tracking.Distance < engine.CameraTrackDistMin {
			c.Tracking.Distance = engine.CameraTrackDistMin
		}
		if c.Tracking.Distance > engine.CameraTrackDistMax {
			c.Tracking.Distance = engine.CameraTrackDistMax
		}
	}
	// Dwell countdown for multi-hop jump queues.
	if c.Jump.DwellRemaining > 0 {
		c.Jump.DwellRemaining -= float64(dt)
		if c.Jump.DwellRemaining <= 0 && len(c.Jump.Queue) > 0 {
			next := c.Jump.Queue[0]
			c.Jump.Queue = c.Jump.Queue[1:]
			c.Jump.CurrentDwell = next.DwellSeconds
			c.StartJumpTo(next.TargetIndex, next.TargetPos, next.ViewDist)
			return
		}
	}
	// Orbit animation tick.
	if c.OrbitSpeed != 0 && c.OrbitRadiansRemaining > 0 {
		delta := c.OrbitSpeed * float64(dt)
		c.Tracking.Yaw += delta
		c.OrbitRadiansRemaining -= math.Abs(delta)
		if c.OrbitRadiansRemaining <= 0 {
			c.OrbitSpeed = 0
		}
	}
	// Mouse-look.
	if mouseYaw != 0 || mousePitch != 0 {
		c.AdjustTrackAngles(mouseYaw, mousePitch)
	}
	// Recompute position/forward from updated tracking angles.
	c.UpdateTracking(state)
	// Offset adjustments (reset wins over WASD/arrow, matching original behavior).
	if resetOffset {
		c.Tracking.Offset = engine.Vector3{}
	} else {
		c.Tracking.Offset = c.Tracking.Offset.Add(wasdDelta).Add(arrowDelta)
	}
}

// TickFreeFly processes one frame of free-fly mode physics (no ShipInstance).
// All input vectors must be pre-zeroed by the caller when input is suspended.
// freeZoom moves the camera along its forward vector (positive = forward).
func (c *CameraState) TickFreeFly(state *engine.SimulationState, dt float32,
	wasdDelta, arrowDelta engine.Vector3,
	mouseYaw, mousePitch, rollDelta float64,
	freeZoom float32,
) {
	c.Yaw += mouseYaw
	c.Pitch += mousePitch
	if c.Pitch > 1.5 {
		c.Pitch = 1.5
	}
	if c.Pitch < -1.5 {
		c.Pitch = -1.5
	}
	c.Roll += rollDelta
	c.UpdateForwardFromAngles()
	if freeZoom != 0 {
		c.Position = c.Position.Add(c.Forward.Scale(freeZoom))
	}
	c.Position = c.Position.Add(wasdDelta).Add(arrowDelta)
	// Persistent velocity drift (set via gRPC NavigationService).
	if c.Velocity.X != 0 || c.Velocity.Y != 0 || c.Velocity.Z != 0 {
		c.Position = c.Position.Add(c.Velocity.Scale(dt))
	}
	// Body-exclusion: push camera out of any solid named body.
	for _, obj := range state.Objects {
		cat := obj.Meta.Category
		if cat == engine.CategoryBelt || cat == engine.CategoryRing || cat == engine.CategoryAsteroid {
			continue
		}
		if obj.Meta.PhysicalRadius <= 0 {
			continue
		}
		diff := c.Position.Sub(obj.Anim.Position)
		d := float64(diff.Length())
		minDist := float64(obj.Meta.PhysicalRadius) + 0.5
		if d < minDist {
			if d > 0.001 {
				c.Position = obj.Anim.Position.Add(diff.Normalize().Scale(float32(minDist)))
			} else {
				c.Position = obj.Anim.Position.Add(engine.Vector3{Y: float32(minDist)})
			}
		}
	}
}

// TickJump processes one frame of jump-mode physics.
// All input vectors must be pre-zeroed by the caller when input is suspended.
// freeZoom moves the camera along its forward vector before the animation step.
func (c *CameraState) TickJump(state *engine.SimulationState, dt float32,
	mouseYaw, mousePitch float64,
	arrowDelta engine.Vector3,
	freeZoom float32,
) {
	if freeZoom != 0 {
		c.Position = c.Position.Add(c.Forward.Scale(freeZoom))
	}
	c.UpdateJump(float64(dt))
	// Arrival: UpdateJump changed Mode to CameraModeFree — start tracking immediately.
	if c.Mode == CameraModeFree {
		c.StartTracking(c.Jump.TargetIndex)
		c.Tracking.Distance = c.Jump.TargetViewDist
		c.Tracking.Offset = engine.Vector3{}
		c.UpdateTracking(state)
		if c.PendingOrbitSpeed != 0 {
			c.OrbitSpeed = c.PendingOrbitSpeed
			c.OrbitRadiansRemaining = c.PendingOrbitRadians
			c.PendingOrbitSpeed = 0
			c.PendingOrbitRadians = 0
		}
		c.Jump.DwellRemaining = c.Jump.CurrentDwell
		if c.Jump.DwellRemaining <= 0 && len(c.Jump.Queue) > 0 {
			next := c.Jump.Queue[0]
			c.Jump.Queue = c.Jump.Queue[1:]
			c.Jump.CurrentDwell = next.DwellSeconds
			c.StartJumpTo(next.TargetIndex, next.TargetPos, next.ViewDist)
		}
		return
	}
	// Still in-flight: apply mouse look and arrow-key repositioning.
	c.Yaw += mouseYaw
	c.Pitch += mousePitch
	if c.Pitch > 1.5 {
		c.Pitch = 1.5
	}
	if c.Pitch < -1.5 {
		c.Pitch = -1.5
	}
	c.UpdateForwardFromAngles()
	c.Position = c.Position.Add(arrowDelta)
}
