package app

import (
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// AppCmd is a command dispatched from a gRPC handler goroutine to the OS
// main thread. The interactive loop drains the channel each frame (non-blocking)
// and executes each command. This ensures all Raylib calls happen on the
// thread that owns the GL context.
//
// Fire-and-forget commands carry no response. Query commands embed a
// buffered (cap 1) response channel that the main thread writes to exactly once.
type AppCmd interface {
	isAppCmd()
}

// ── Window commands ───────────────────────────────────────────────────────────

// WindowSizeCmd resizes the window. Both dimensions must be > 0.
type WindowSizeCmd struct {
	Width, Height int32
}

// WindowMaximizeCmd maximises the window (requires FLAG_WINDOW_RESIZABLE).
type WindowMaximizeCmd struct{}

// WindowRestoreCmd restores the window to its last non-maximised size.
// If no prior windowed size is recorded, defaults are used.
type WindowRestoreCmd struct{}

// WindowFullscreenCmd toggles true fullscreen on (On=true) or reverts to windowed (On=false).
type WindowFullscreenCmd struct {
	On bool
}

// GetWindowCmd requests a snapshot of the current window state.
// The caller must pre-create the buffered channel and wait on it.
type GetWindowCmd struct {
	RespCh chan<- WindowSnapshot
}

// WindowSnapshot is a read-only point-in-time view of window geometry.
type WindowSnapshot struct {
	Width, Height int32
	Fullscreen    bool
	Maximized     bool
}

// ── Camera commands ───────────────────────────────────────────────────────────

// CameraOrientCmd sets camera yaw and pitch in degrees.
// Yaw: 0–360 (0 = +Z). Pitch: −85 to +85.
type CameraOrientCmd struct {
	YawDeg, PitchDeg float64
}

// CameraPositionCmd teleports the camera to an absolute position in AU.
type CameraPositionCmd struct {
	Pos engine.Vector3
}

// CameraTrackCmd locks the camera onto a named body (continuous follow).
// Name="" returns to free-fly mode.
type CameraTrackCmd struct {
	Name string
}

// GetCameraCmd requests a point-in-time snapshot of camera state.
type GetCameraCmd struct {
	RespCh chan<- CameraSnapshot
}

// CameraSnapshot is a read-only view of camera state.
type CameraSnapshot struct {
	YawDeg      float64
	PitchDeg    float64
	Position    engine.Vector3
	Mode        ui.CameraMode
	TrackTarget string // body name; "" when in free-fly mode
}

// ── Navigation commands ───────────────────────────────────────────────────────

// SetVelocityCmd stores a persistent per-frame velocity drift (AU/s).
// Zeroing all three axes equals nav stop.
type SetVelocityCmd struct {
	Velocity engine.Vector3
}

// JumpToCmd queues an animated multi-hop camera jump.
// Names are resolved against the current simulation snapshot at dispatch time.
// Unresolvable names are silently skipped.
type JumpToCmd struct {
	Names []string
}

// GetVelocityCmd requests the current persistent-velocity vector.
type GetVelocityCmd struct {
	RespCh chan<- engine.Vector3
}

// ── Performance commands ──────────────────────────────────────────────────────

// PerfSetCmd applies a partial update to the nine performance knobs.
// Only fields whose corresponding Set* bool is true are changed.
type PerfSetCmd struct {
	Options     ui.PerformanceOptions
	CameraSpeed float32
	NumWorkers  int
	HUDVisible  bool

	SetFrustumCulling      bool
	SetLODEnabled          bool
	SetInstancedRendering  bool
	SetSpatialPartition    bool
	SetPointRendering      bool
	SetImportanceThreshold bool
	SetUseInPlaceSwap      bool
	SetCameraSpeed         bool
	SetNumWorkers          bool
	SetHUDVisible          bool
}

// GetPerfCmd requests a snapshot of all performance knobs.
type GetPerfCmd struct {
	RespCh chan<- PerfSnapshot
}

// PerfSnapshot is a read-only view of all nine performance knobs plus extras.
type PerfSnapshot struct {
	Options     ui.PerformanceOptions
	CameraSpeed float32
	NumWorkers  int // from engine.SimulationState.NumWorkers (back buffer)
	HUDVisible  bool
}

// ── System commands ───────────────────────────────────────────────────────────

// LoadSystemCmd triggers an in-place session reload using the given path.
// Path validation (must be under data/systems/) is the caller's responsibility.
type LoadSystemCmd struct {
	Path string
}

// GetActiveSystemCmd requests the currently loaded system path.
type GetActiveSystemCmd struct {
	RespCh chan<- string
}

// ── HUD commands ──────────────────────────────────────────────────────────────

// SetHUDCmd shows or hides the heads-up display.
type SetHUDCmd struct {
	Visible bool
}

// ── Orbit commands ────────────────────────────────────────────────────────────

// OrbitCmd starts an animated orbit around a named body.
// SpeedDegPerSec is the angular speed in degrees per second (positive = CCW from above).
// Orbits is the number of full 360° circuits to complete.
type OrbitCmd struct {
	Name           string
	SpeedDegPerSec float64
	Orbits         float64
}

// ── marker interface implementations ─────────────────────────────────────────

func (WindowSizeCmd) isAppCmd()        {}
func (WindowMaximizeCmd) isAppCmd()    {}
func (WindowRestoreCmd) isAppCmd()     {}
func (WindowFullscreenCmd) isAppCmd()  {}
func (GetWindowCmd) isAppCmd()         {}
func (CameraOrientCmd) isAppCmd()    {}
func (CameraPositionCmd) isAppCmd()  {}
func (CameraTrackCmd) isAppCmd()     {}
func (GetCameraCmd) isAppCmd()       {}
func (SetVelocityCmd) isAppCmd()     {}
func (JumpToCmd) isAppCmd()          {}
func (GetVelocityCmd) isAppCmd()     {}
func (PerfSetCmd) isAppCmd()         {}
func (GetPerfCmd) isAppCmd()         {}
func (LoadSystemCmd) isAppCmd()      {}
func (GetActiveSystemCmd) isAppCmd() {}
func (SetHUDCmd) isAppCmd()          {}
func (OrbitCmd) isAppCmd()           {}
