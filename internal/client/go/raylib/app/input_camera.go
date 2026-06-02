package app

import (
	"fmt"
	"math"

	"github.com/digital-michael/space_sim/internal/client/go/raylib/input"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleInputCamera processes camera-control keybindings from handleInput:
// camera.center (reset zoom or face origin) and camera.toggle_free_fly.
func (a *App) handleInputCamera(session *runtimeSession, state *engine.SimulationState, km *input.KeyMap, suspended bool) {
	if suspended {
		return
	}
	// camera.center: Free-fly: face origin. Tracking: reset zoom to 40% auto-distance.
	if km.IsPressed(input.ActionCameraCenter) {
		if session.cameraState.Mode == ui.CameraModeFree {
			toOrigin := engine.Vector3{X: 0, Y: 0, Z: 0}.Sub(session.cameraState.Position)
			if toOrigin.Length() > 0.1 {
				session.cameraState.Forward = toOrigin.Normalize()
				session.cameraState.Yaw = math.Atan2(float64(session.cameraState.Forward.X), float64(session.cameraState.Forward.Z))
				session.cameraState.Pitch = math.Asin(float64(session.cameraState.Forward.Y))
			}
		} else if session.cameraState.Mode == ui.CameraModeTracking {
			idx := session.cameraState.Tracking.TargetIndex
			if idx >= 0 && idx < len(state.Objects) {
				session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(state.Objects[idx].Meta.ViewRadius(), 0.40)
			}
		}
	}
	// camera.toggle_free_fly: Re-enter tracking mode from free-fly (uses last tracked index).
	if km.IsPressed(input.ActionCameraToggleFreeFly) && session.cameraState.Mode == ui.CameraModeFree {
		idx := session.cameraState.Tracking.TargetIndex
		if idx >= 0 && idx < len(state.Objects) {
			session.cameraState.StartTracking(idx)
			session.cameraState.Tracking.Distance = ui.CalculateAutoZoomDistance(state.Objects[idx].Meta.ViewRadius(), 0.40)
		}
	}
}

// updateCameraState applies all per-frame camera physics and returns wheelMove
// for the zoom-indicator overlay.
func (a *App) updateCameraState(session *runtimeSession, state *engine.SimulationState, dt float32) float32 {
	km, cs := a.keyMap.Load(), session.cameraState
	suspended := session.inputState.MainWindowInputSuspended() || a.runtime.SettingsVisible

	var mouseDelta rl.Vector2
	if a.runtime.MouseModeEnabled && !suspended {
		mouseDelta = rl.GetMouseDelta()
	}
	wheelMove := float32(0)
	if !suspended {
		wheelMove = rl.GetMouseWheelMove()
	}

	moveSpeed := a.runtime.CameraSpeed * dt
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		moveSpeed *= 2
	}
	sens := a.runtime.MouseSensitivity
	mouseYaw, mousePitch := -float64(mouseDelta.X*sens), -float64(mouseDelta.Y*sens)
	arrowDelta := cameraArrowDelta(km, moveSpeed, suspended)

	switch cs.Mode {
	case ui.CameraModeJumping:
		fz := cameraFreeZoom(km, wheelMove, a.runtime.CameraSpeed, dt, suspended)
		cs.TickJump(state, dt, mouseYaw, mousePitch, arrowDelta, fz)

	case ui.CameraModeTracking:
		zd := cameraTrackZoom(km, wheelMove, a.runtime.CameraSpeed, dt, suspended)
		wd := cameraTrackWASD(cs, km, moveSpeed, suspended)
		cs.TickTracking(state, dt, wd, arrowDelta, mouseYaw*0.5, mousePitch*0.5, zd,
			!suspended && km.IsPressed(input.ActionCameraReset))

	case ui.CameraModeFree:
		rd := cameraRollDelta(km, dt, suspended)
		if session.ship != nil {
			a.tickFreeFlyShip(session, km, dt, arrowDelta, mouseYaw, mousePitch, rd, suspended)
		} else {
			fz := cameraFreeZoom(km, wheelMove, a.runtime.CameraSpeed, dt, suspended)
			wd := cameraFreeWASD(cs, km, moveSpeed, suspended)
			cs.TickFreeFly(state, dt, wd, arrowDelta, mouseYaw, mousePitch, rd, fz)
		}
		a.mirrorShipFacing(session)
	}
	return wheelMove
}

// cameraArrowDelta returns the pre-scaled world-axis delta from arrow keys.
func cameraArrowDelta(km *input.KeyMap, speed float32, suspended bool) engine.Vector3 {
	if suspended {
		return engine.Vector3{}
	}
	var v engine.Vector3
	if km.IsDown(input.ActionCameraPitchUp) {
		v.Y += speed
	}
	if km.IsDown(input.ActionCameraPitchDown) {
		v.Y -= speed
	}
	if km.IsDown(input.ActionCameraYawLeft) {
		v.X -= speed
	}
	if km.IsDown(input.ActionCameraYawRight) {
		v.X += speed
	}
	return v
}

// cameraTrackZoom returns the signed Tracking.Distance delta for this frame.
// Positive = zoom out (increase distance), negative = zoom in.
func cameraTrackZoom(km *input.KeyMap, wheel, speed, dt float32, suspended bool) float64 {
	if suspended {
		return 0
	}
	delta := -float64(wheel) * float64(speed) * 5.0
	step := float64(speed) * float64(dt) * 50.0
	if km.IsDown(input.ActionCameraZoomIn) {
		delta -= step
	}
	if km.IsDown(input.ActionCameraZoomOut) {
		delta += step
	}
	return delta
}

// cameraFreeZoom returns the forward-axis position delta for free/jump modes.
func cameraFreeZoom(km *input.KeyMap, wheel, speed, dt float32, suspended bool) float32 {
	if suspended {
		return 0
	}
	fz := wheel * speed * 0.5
	keyStep := speed * dt * 5.0
	if km.IsDown(input.ActionCameraZoomIn) {
		fz += keyStep
	}
	if km.IsDown(input.ActionCameraZoomOut) {
		fz -= keyStep
	}
	return fz * 10.0
}

// cameraTrackWASD returns the world-space offset delta for tracking-mode WASD.
func cameraTrackWASD(cs *ui.CameraState, km *input.KeyMap, speed float32, suspended bool) engine.Vector3 {
	if suspended {
		return engine.Vector3{}
	}
	right := cs.GetRight()
	var v engine.Vector3
	if km.IsDown(input.ActionThrustForward) {
		v = v.Add(cs.Forward.Scale(speed))
	}
	if km.IsDown(input.ActionThrustBackward) {
		v = v.Sub(cs.Forward.Scale(speed))
	}
	if km.IsDown(input.ActionThrustLeft) {
		v = v.Sub(right.Scale(speed))
	}
	if km.IsDown(input.ActionThrustRight) {
		v = v.Add(right.Scale(speed))
	}
	return v
}

// cameraFreeWASD returns the world-space position delta for free-fly WASD (no ship).
func cameraFreeWASD(cs *ui.CameraState, km *input.KeyMap, speed float32, suspended bool) engine.Vector3 {
	if suspended {
		return engine.Vector3{}
	}
	right := cs.GetRight()
	var v engine.Vector3
	if km.IsDown(input.ActionThrustForward) {
		v = v.Add(cs.Forward.Scale(speed))
	}
	if km.IsDown(input.ActionThrustBackward) {
		v = v.Sub(cs.Forward.Scale(speed))
	}
	if km.IsDown(input.ActionThrustLeft) {
		v = v.Sub(right.Scale(speed))
	}
	if km.IsDown(input.ActionThrustRight) {
		v = v.Add(right.Scale(speed))
	}
	if km.IsDown(input.ActionThrustUp) {
		v = v.Add(cs.Up.Scale(speed))
	}
	if km.IsDown(input.ActionThrustDown) {
		v = v.Sub(cs.Up.Scale(speed))
	}
	return v
}

// cameraRollDelta returns the roll angular delta for this frame.
func cameraRollDelta(km *input.KeyMap, dt float32, suspended bool) float64 {
	if suspended {
		return 0
	}
	const rollSpeed = 1.5
	var delta float64
	if km.IsDown(input.ActionCameraRollLeft) {
		delta -= rollSpeed * float64(dt)
	}
	if km.IsDown(input.ActionCameraRollRight) {
		delta += rollSpeed * float64(dt)
	}
	return delta
}

// mirrorShipFacing copies the camera forward vector into session.ship.FacingVector.
// No-ops when session.ship is nil.
func (a *App) mirrorShipFacing(session *runtimeSession) {
	if session.ship == nil {
		return
	}
	fwd := session.cameraState.Forward
	session.ship.FacingVector = [3]float32{fwd.X, fwd.Y, fwd.Z}
}

// tickFreeFlyShip handles per-frame kinematics for the ship-piloting path in free-fly mode.
// Mouse/roll are applied to the camera; thrust is applied to ship velocity; position is mirrored back.
func (a *App) tickFreeFlyShip(session *runtimeSession, km *input.KeyMap, dt float32,
	arrowDelta engine.Vector3, mouseYaw, mousePitch, rollDelta float64, suspended bool,
) {
	cs := session.cameraState
	ship := session.ship

	cs.Yaw += mouseYaw
	cs.Pitch += mousePitch
	if cs.Pitch > 1.5 {
		cs.Pitch = 1.5
	}
	if cs.Pitch < -1.5 {
		cs.Pitch = -1.5
	}
	cs.Roll += rollDelta
	cs.UpdateForwardFromAngles()

	if suspended {
		return
	}

	dtF := float64(dt)
	accelSimUnits := ship.EffectiveAccelMaxMS2() / engine.MetersPerSimUnit

	if km.IsPressed(input.ActionBrake) {
		ship.Velocity = [3]float64{}
	}
	if km.IsPressed(input.ActionDriftToggle) {
		session.driftMode = !session.driftMode
		if session.driftMode {
			fmt.Println("Drift mode ON")
		} else {
			fmt.Println("Drift mode OFF")
		}
	}

	if !session.driftMode {
		right := cs.GetRight()
		if km.IsDown(input.ActionThrustForward) {
			ship.Velocity[0] += float64(cs.Forward.X) * accelSimUnits * dtF
			ship.Velocity[1] += float64(cs.Forward.Y) * accelSimUnits * dtF
			ship.Velocity[2] += float64(cs.Forward.Z) * accelSimUnits * dtF
		}
		if km.IsDown(input.ActionThrustBackward) {
			ship.Velocity[0] -= float64(cs.Forward.X) * accelSimUnits * dtF
			ship.Velocity[1] -= float64(cs.Forward.Y) * accelSimUnits * dtF
			ship.Velocity[2] -= float64(cs.Forward.Z) * accelSimUnits * dtF
		}
		if km.IsDown(input.ActionThrustLeft) {
			ship.Velocity[0] -= float64(right.X) * accelSimUnits * dtF
			ship.Velocity[1] -= float64(right.Y) * accelSimUnits * dtF
			ship.Velocity[2] -= float64(right.Z) * accelSimUnits * dtF
		}
		if km.IsDown(input.ActionThrustRight) {
			ship.Velocity[0] += float64(right.X) * accelSimUnits * dtF
			ship.Velocity[1] += float64(right.Y) * accelSimUnits * dtF
			ship.Velocity[2] += float64(right.Z) * accelSimUnits * dtF
		}
		if km.IsDown(input.ActionThrustUp) {
			ship.Velocity[0] += float64(cs.Up.X) * accelSimUnits * dtF
			ship.Velocity[1] += float64(cs.Up.Y) * accelSimUnits * dtF
			ship.Velocity[2] += float64(cs.Up.Z) * accelSimUnits * dtF
		}
		if km.IsDown(input.ActionThrustDown) {
			ship.Velocity[0] -= float64(cs.Up.X) * accelSimUnits * dtF
			ship.Velocity[1] -= float64(cs.Up.Y) * accelSimUnits * dtF
			ship.Velocity[2] -= float64(cs.Up.Z) * accelSimUnits * dtF
		}
	}

	// Clamp to max speed.
	if maxSpeed := ship.Definition.MaxSpeedSimUnitsPerS; maxSpeed > 0 {
		speed := math.Sqrt(
			ship.Velocity[0]*ship.Velocity[0] +
				ship.Velocity[1]*ship.Velocity[1] +
				ship.Velocity[2]*ship.Velocity[2],
		)
		if speed > maxSpeed {
			scale := maxSpeed / speed
			ship.Velocity[0] *= scale
			ship.Velocity[1] *= scale
			ship.Velocity[2] *= scale
		}
	}

	// Position integration.
	ship.Position[0] += ship.Velocity[0] * dtF
	ship.Position[1] += ship.Velocity[1] * dtF
	ship.Position[2] += ship.Velocity[2] * dtF

	// Arrow keys: direct position (no inertia; useful for spectator repositioning).
	ship.Position[0] += float64(arrowDelta.X)
	ship.Position[1] += float64(arrowDelta.Y)

	// Mirror ship world position to camera (camera IS the ship cockpit).
	cs.Position = engine.Vector3{
		X: float32(ship.Position[0]),
		Y: float32(ship.Position[1]),
		Z: float32(ship.Position[2]),
	}
}
