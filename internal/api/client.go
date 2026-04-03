package api

// CameraController is the contract for anything that can move the camera
// in response to player input or automated scripting.
//
// TODO: flesh out when the camera model is stable enough to enumerate all
// necessary operations (zoom, pan, orbit, track, jump-to, FOV, etc.).
type CameraController interface {
	// JumpToTarget moves the camera instantly to focus on the named body.
	// TODO: define return type (error? async handle?)
	JumpToTarget(name string)

	// TrackTarget keeps the camera centred on the named body across frames.
	// Pass an empty string to stop tracking.
	TrackTarget(name string)

	// StopTracking releases any active tracking target.
	StopTracking()

	// TODO: Zoom(factor float64)
	// TODO: Pan(deltaX, deltaY float64)
	// TODO: Orbit(deltaYaw, deltaPitch float64)
	// TODO: SetFieldOfView(fovDegrees float64)
	// TODO: Reset() — return to the default view
}

// PlayerView is the contract for reading the player's current point-of-view
// state. Intended for rendering, serialisation, and replay tooling.
//
// TODO: flesh out when the POV model is stable. Expected additions:
// selected body name, active camera mode, current FOV, look-at vector.
type PlayerView interface {
	// FocusedBody returns the name of the body currently in focus, or an
	// empty string if no body is selected.
	FocusedBody() string

	// IsTracking reports whether the camera is locked onto a body.
	IsTracking() bool

	// TODO: CameraPosition() — return transport-safe position value
	// TODO: CameraMode() — return an enum or string mode label
	// TODO: ActiveCategory() — return the currently browsed object category
}
