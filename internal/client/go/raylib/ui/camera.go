// Package ui provides generic rendering-support and input-state types for
// Space Sim. It depends only on space/engine and the standard library —
// no Raylib types, no dataset-specific knowledge.
package ui

import (
	"math"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// CameraMode represents the active camera control mode.
type CameraMode int

const (
	CameraModeFree CameraMode = iota
	CameraModeJumping
	CameraModeTracking
)

// CameraState holds camera position, orientation, and animation state.
type CameraState struct {
	Position engine.Vector3
	Forward  engine.Vector3
	Up       engine.Vector3
	Yaw      float64
	Pitch    float64
	Mode     CameraMode

	// Jump animation
	JumpStartPos    engine.Vector3
	JumpTargetPos   engine.Vector3
	JumpProgress    float64
	JumpDuration    float64
	JumpTargetIndex int

	// Tracking
	TrackTargetIndex int
	TrackDistance    float64
	TrackHeight      float64
	TrackOffset      engine.Vector3
	TrackYaw         float64
	TrackPitch       float64
	TrackLookOutward bool
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
	c.JumpDuration = 1.5
}

// UpdateJump advances the jump animation by dt seconds.
func (c *CameraState) UpdateJump(dt float64) {
	if c.Mode != CameraModeJumping {
		return
	}
	c.JumpProgress += dt / c.JumpDuration
	if c.JumpProgress >= 1.0 {
		c.Position = c.JumpTargetPos
		c.Mode = CameraModeFree
		return
	}
	t := c.JumpProgress
	smoothT := float32(t * t * (3.0 - 2.0*t))
	c.Position.X = c.JumpStartPos.X + smoothT*(c.JumpTargetPos.X-c.JumpStartPos.X)
	c.Position.Y = c.JumpStartPos.Y + smoothT*(c.JumpTargetPos.Y-c.JumpStartPos.Y)
	c.Position.Z = c.JumpStartPos.Z + smoothT*(c.JumpTargetPos.Z-c.JumpStartPos.Z)
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
