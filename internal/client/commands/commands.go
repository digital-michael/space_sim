// Package commands defines the typed command set for the Space Sim REPL.
// Each command maps to one ConnectRPC call (or a built-in action).
//
// Parse a raw input line with Parse; the returned Cmd value identifies
// which operation to perform and carries validated arguments.
package commands

import (
	"fmt"
	"strconv"
	"strings"
)

// Cmd is the discriminated union of all REPL commands.
type Cmd interface {
	isCmd()
}

// ─── Command types ────────────────────────────────────────────────────────

// SetSpeed sets the simulation speed multiplier.
//
//	setspeed <seconds_per_second>   e.g.  setspeed 10
type SetSpeed struct {
	SecondsPerSecond float32
}

// GetSpeed queries the current simulation speed.
//
//	getspeed
type GetSpeed struct{}

// SetDataset changes the active asteroid dataset.
//
//	setdataset <small|medium|large|huge>
type SetDataset struct {
	Level string // normalised to lower-case
}

// GetDataset queries the active asteroid dataset.
//
//	getdataset
type GetDataset struct{}

// GetTime queries the current simulation time (seconds since J2000).
//
//	gettime
type GetTime struct{}

// Pause sets the simulation speed to zero.
//
//	pause
type Pause struct{}

// Resume restores the simulation to the last non-zero speed used in this
// REPL session (default 1.0 if speed was never set).
//
//	resume
type Resume struct{}

// Bodies lists all visible bodies in the current live snapshot.
// Category is an optional case-insensitive filter (e.g. "planet", "moon").
//
//	bodies
//	bodies <category>   e.g. bodies planet
type Bodies struct {
	Category string
}

// Inspect fetches position and metadata for a named body.
//
//	inspect <name>   e.g. inspect Earth
type Inspect struct {
	Name string // original case preserved
}

// Status prints a concise summary: speed, dataset, and simulation date.
//
//	status
type Status struct{}

// Stream opens a server-pushed snapshot stream and prints bodies until
// the user presses Ctrl-C.
//
//	stream
type Stream struct{}

// Help prints the command reference.
//
//	help
type Help struct{}

// Quit exits the REPL.
//
//	quit  |  exit
type Quit struct{}

func (SetSpeed) isCmd()   {}
func (GetSpeed) isCmd()   {}
func (SetDataset) isCmd() {}
func (GetDataset) isCmd() {}
func (GetTime) isCmd()    {}
func (Pause) isCmd()      {}
func (Resume) isCmd()     {}
func (Bodies) isCmd()     {}
func (Inspect) isCmd()    {}
func (Status) isCmd()     {}
func (Stream) isCmd()     {}
func (Help) isCmd()       {}
func (Quit) isCmd()       {}

// ─── System commands ──────────────────────────────────────────────────────────

// SystemList lists all discoverable solar-system files.
//
//	system list
type SystemList struct{}

// SystemGet prints the currently loaded system.
//
//	system get
type SystemGet struct{}

// SystemLoad triggers an in-place session reload.
//
//	system load <label>   e.g.  system load solar_system.json
type SystemLoad struct {
	Label string
}

// ─── Window commands ──────────────────────────────────────────────────────────

// WindowGet prints current window dimensions and state.
//
//	window get
type WindowGet struct{}

// WindowSize resizes the window.
//
//	window size <W>x<H>   e.g.  window size 1920x1080
type WindowSize struct {
	Width, Height int32
}

// WindowMaximize maximises the window.
//
//	window maximize
type WindowMaximize struct{}

// WindowRestore restores the window from maximised state.
//
//	window restore
type WindowRestore struct{}

// ─── Camera commands ──────────────────────────────────────────────────────────

// CameraGet prints current camera state.
//
//	camera get
type CameraGet struct{}

// CameraOrient sets the camera yaw and pitch.
//
//	camera orient <yaw_deg> <pitch_deg>   e.g.  camera orient 90 -15
type CameraOrient struct {
	YawDeg, PitchDeg float32
}

// CameraPosition teleports the camera to an AU coordinate.
//
//	camera position <x> <y> <z>
type CameraPosition struct {
	X, Y, Z float64
}

// CameraCenter snaps the camera to the current tracking target, or to the
// solar system origin (0, 0, 0) when in free-fly mode.
//
//	camera center
type CameraCenter struct{}

// CameraTrack locks the camera onto a named body; empty Name returns to free-fly.
//
//	camera track <name>   e.g.  camera track Earth
//	camera track          (no name — free-fly)
type CameraTrack struct {
	Name string
}

// ─── Navigation commands ──────────────────────────────────────────────────────

// NavStop zeroes all persistent camera velocity.
//
//	nav stop
type NavStop struct{}

// NavVelocity prints the current persistent velocity vector.
//
//	nav velocity
type NavVelocity struct{}

// NavMove sets a named-axis velocity component.
// Dir is one of: forward, back, left, right, up, down.
//
//	nav forward <v>
//	nav back    <v>
//	nav left    <v>
//	nav right   <v>
//	nav up      <v>
//	nav down    <v>
type NavMove struct {
	Dir      string  // "forward" | "back" | "left" | "right" | "up" | "down"
	Velocity float32 // AU/s
}

// NavJump queues an animated multi-hop jump.
//
//	nav jump <name> [<name> ...]   e.g.  nav jump Earth Saturn
type NavJump struct {
	Names []string
}

// ─── Performance commands ─────────────────────────────────────────────────────

// PerfGet prints all nine performance knobs.
//
//	perf get
type PerfGet struct{}

// PerfSet updates one named performance knob.
//
//	perf set <field> <value>
//
// Fields: frustum_culling lod instanced_rendering spatial_partition
//
//	point_rendering importance_threshold use_in_place_swap
//	camera_speed workers
type PerfSet struct {
	Field string
	Value string
}

// ─── Shutdown ────────────────────────────────────────────────────────────────

// Clear empties the terminal display.
//
//	clear
type Clear struct{}

// Shutdown asks the server to shut down gracefully.
//
//	shutdown
type Shutdown struct{}

// Orbit starts an animated orbit around a named body.
//
//	orbit <target> <speed_deg_per_sec> <n_orbits>   e.g.  orbit Earth 10 2
type Orbit struct {
	Name           string
	SpeedDegPerSec float64
	Orbits         float64
}

// Sleep pauses REPL script execution for the given number of seconds.
//
//	sleep <seconds>   e.g.  sleep 2.5
type Sleep struct {
	Seconds float64
}

// HUD enables or disables the heads-up display overlay.
//
//	hud on | hud off
type HUD struct {
	Visible bool
}

func (SystemList) isCmd()     {}
func (SystemGet) isCmd()      {}
func (SystemLoad) isCmd()     {}
func (WindowGet) isCmd()      {}
func (WindowSize) isCmd()     {}
func (WindowMaximize) isCmd() {}
func (WindowRestore) isCmd()  {}
func (CameraGet) isCmd()      {}
func (CameraCenter) isCmd()   {}
func (CameraOrient) isCmd()   {}
func (CameraPosition) isCmd() {}
func (CameraTrack) isCmd()    {}
func (NavStop) isCmd()        {}
func (NavVelocity) isCmd()    {}
func (NavMove) isCmd()        {}
func (NavJump) isCmd()        {}
func (PerfGet) isCmd()        {}
func (PerfSet) isCmd()        {}
func (Clear) isCmd()          {}
func (Shutdown) isCmd()       {}
func (Orbit) isCmd()          {}
func (Sleep) isCmd()          {}
func (HUD) isCmd()            {}

// ValidDatasetLevels is the set of accepted level names for SetDataset.
var ValidDatasetLevels = map[string]struct{}{
	"small":  {},
	"medium": {},
	"large":  {},
	"huge":   {},
}

// ErrUnknownCommand is returned when the first token does not match any
// known command name.
type ErrUnknownCommand struct{ Input string }

func (e ErrUnknownCommand) Error() string {
	return fmt.Sprintf("unknown command %q — type 'help' for usage", e.Input)
}

// ErrUsage is returned when a command is recognised but the argument list is
// malformed.
type ErrUsage struct {
	Cmd     string
	Detail  string
	Example string
}

func (e ErrUsage) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Cmd, e.Detail)
	if e.Example != "" {
		msg += fmt.Sprintf("  (usage: %s)", e.Example)
	}
	return msg
}

// Parse parses one trimmed input line into a Cmd. It returns
// ErrUnknownCommand or ErrUsage on parse failure. Empty/comment lines return
// nil, nil — the caller should skip them.
func Parse(line string) (Cmd, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	verb := strings.ToLower(fields[0])
	args := fields[1:]

	switch verb {
	case "setspeed":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "setspeed", Detail: "expected one argument", Example: "setspeed 10"}
		}
		v, err := strconv.ParseFloat(args[0], 32)
		if err != nil || v < 0 {
			return nil, ErrUsage{Cmd: "setspeed", Detail: "argument must be a non-negative number (0 = pause)", Example: "setspeed 10"}
		}
		return SetSpeed{SecondsPerSecond: float32(v)}, nil

	case "getspeed":
		return GetSpeed{}, nil

	case "setdataset":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "setdataset", Detail: "expected one argument", Example: "setdataset small"}
		}
		level := strings.ToLower(args[0])
		if _, ok := ValidDatasetLevels[level]; !ok {
			return nil, ErrUsage{Cmd: "setdataset", Detail: fmt.Sprintf("unknown level %q", args[0]), Example: "setdataset <small|medium|large|huge>"}
		}
		return SetDataset{Level: level}, nil

	case "getdataset":
		return GetDataset{}, nil

	case "gettime":
		return GetTime{}, nil

	case "pause":
		return Pause{}, nil

	case "resume":
		return Resume{}, nil

	case "bodies":
		if len(args) > 1 {
			return nil, ErrUsage{Cmd: "bodies", Detail: "at most one category filter", Example: "bodies planet"}
		}
		filter := ""
		if len(args) == 1 {
			filter = strings.ToLower(args[0])
		}
		return Bodies{Category: filter}, nil

	case "inspect":
		if len(args) == 0 {
			return nil, ErrUsage{Cmd: "inspect", Detail: "expected a body name", Example: "inspect Earth"}
		}
		return Inspect{Name: strings.Join(args, " ")}, nil

	case "status":
		return Status{}, nil

	case "stream":
		return Stream{}, nil

	case "clear":
		return Clear{}, nil

	case "help":
		return Help{}, nil

	case "quit", "exit":
		return Quit{}, nil

	// ── System ────────────────────────────────────────────────────────────────
	case "system":
		if len(args) == 0 {
			return nil, fmt.Errorf("system: subcommand required (list|get|load)")
		}
		switch args[0] {
		case "list":
			return SystemList{}, nil
		case "get":
			return SystemGet{}, nil
		case "load":
			if len(args) < 2 || args[1] == "" {
				return nil, fmt.Errorf("system load: label required")
			}
			return SystemLoad{Label: args[1]}, nil
		default:
			return nil, fmt.Errorf("system: unknown subcommand %q", args[0])
		}

	// ── Window ────────────────────────────────────────────────────────────────
	case "window":
		if len(args) == 0 {
			return nil, fmt.Errorf("window: subcommand required (get|size|maximize|restore)")
		}
		switch args[0] {
		case "get":
			return WindowGet{}, nil
		case "size":
			if len(args) < 2 {
				return nil, fmt.Errorf("window size: WxH required, e.g. 1920x1080")
			}
			parts := strings.SplitN(args[1], "x", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("window size: format must be WxH, got %q", args[1])
			}
			w, err := strconv.Atoi(parts[0])
			if err != nil || w <= 0 {
				return nil, fmt.Errorf("window size: invalid width %q", parts[0])
			}
			h, err := strconv.Atoi(parts[1])
			if err != nil || h <= 0 {
				return nil, fmt.Errorf("window size: invalid height %q", parts[1])
			}
			return WindowSize{Width: int32(w), Height: int32(h)}, nil
		case "maximize":
			return WindowMaximize{}, nil
		case "restore":
			return WindowRestore{}, nil
		default:
			return nil, fmt.Errorf("window: unknown subcommand %q", args[0])
		}

	// ── Camera ────────────────────────────────────────────────────────────────
	case "camera":
		if len(args) == 0 {
			return nil, fmt.Errorf("camera: subcommand required (get|orient|position|track)")
		}
		switch args[0] {
		case "get":
			return CameraGet{}, nil
		case "center":
			return CameraCenter{}, nil
		case "orient":
			if len(args) < 3 {
				return nil, fmt.Errorf("camera orient: yaw pitch required")
			}
			yaw, err := strconv.ParseFloat(args[1], 32)
			if err != nil {
				return nil, fmt.Errorf("camera orient: invalid yaw %q", args[1])
			}
			pitch, err := strconv.ParseFloat(args[2], 32)
			if err != nil {
				return nil, fmt.Errorf("camera orient: invalid pitch %q", args[2])
			}
			return CameraOrient{YawDeg: float32(yaw), PitchDeg: float32(pitch)}, nil
		case "position":
			if len(args) < 4 {
				return nil, fmt.Errorf("camera position: x y z required")
			}
			x, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return nil, fmt.Errorf("camera position: invalid x %q", args[1])
			}
			y, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return nil, fmt.Errorf("camera position: invalid y %q", args[2])
			}
			z, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				return nil, fmt.Errorf("camera position: invalid z %q", args[3])
			}
			return CameraPosition{X: x, Y: y, Z: z}, nil
		case "track":
			name := ""
			if len(args) >= 2 {
				name = args[1]
			}
			return CameraTrack{Name: name}, nil
		default:
			return nil, fmt.Errorf("camera: unknown subcommand %q", args[0])
		}

	// ── Navigation ────────────────────────────────────────────────────────────
	case "nav":
		if len(args) == 0 {
			return nil, fmt.Errorf("nav: subcommand required (stop|velocity|forward|back|left|right|up|down|jump)")
		}
		switch args[0] {
		case "stop":
			return NavStop{}, nil
		case "velocity":
			return NavVelocity{}, nil
		case "forward", "back", "left", "right", "up", "down":
			if len(args) < 2 {
				return nil, fmt.Errorf("nav %s: velocity value required", args[0])
			}
			v, err := strconv.ParseFloat(args[1], 32)
			if err != nil {
				return nil, fmt.Errorf("nav %s: invalid value %q", args[0], args[1])
			}
			return NavMove{Dir: args[0], Velocity: float32(v)}, nil
		case "jump":
			if len(args) < 2 {
				return nil, fmt.Errorf("nav jump: at least one body name required")
			}
			// "nav jump clear" stops tracking without starting a new jump.
			if len(args) == 2 && strings.EqualFold(args[1], "clear") {
				return CameraTrack{Name: ""}, nil
			}
			return NavJump{Names: args[1:]}, nil
		default:
			return nil, fmt.Errorf("nav: unknown subcommand %q", args[0])
		}

	// ── Performance ───────────────────────────────────────────────────────────
	case "perf":
		if len(args) == 0 {
			return nil, fmt.Errorf("perf: subcommand required (get|set)")
		}
		switch args[0] {
		case "get":
			return PerfGet{}, nil
		case "set":
			if len(args) < 3 {
				return nil, fmt.Errorf("perf set: field and value required")
			}
			return PerfSet{Field: args[1], Value: args[2]}, nil
		default:
			return nil, fmt.Errorf("perf: unknown subcommand %q", args[0])
		}

	// ── Shutdown ──────────────────────────────────────────────────────────────
	case "shutdown":
		return Shutdown{}, nil

	// ── Orbit ─────────────────────────────────────────────────────────────────
	case "orbit":
		if len(args) != 3 {
			return nil, ErrUsage{Cmd: "orbit", Detail: "expected target speed orbits", Example: "orbit Earth 10 2"}
		}
		speed, err := strconv.ParseFloat(args[1], 64)
		if err != nil || speed == 0 {
			return nil, ErrUsage{Cmd: "orbit", Detail: "speed must be a non-zero number (deg/sec)", Example: "orbit Earth 10 2"}
		}
		orbits, err := strconv.ParseFloat(args[2], 64)
		if err != nil || orbits <= 0 {
			return nil, ErrUsage{Cmd: "orbit", Detail: "orbits must be a positive number", Example: "orbit Earth 10 2"}
		}
		return Orbit{Name: args[0], SpeedDegPerSec: speed, Orbits: orbits}, nil

	// ── Sleep ─────────────────────────────────────────────────────────────────
	case "sleep":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "sleep", Detail: "expected one argument", Example: "sleep 2.5"}
		}
		secs, err := strconv.ParseFloat(args[0], 64)
		if err != nil || secs < 0 {
			return nil, ErrUsage{Cmd: "sleep", Detail: "argument must be a non-negative number", Example: "sleep 2.5"}
		}
		return Sleep{Seconds: secs}, nil

	// ── HUD ───────────────────────────────────────────────────────────────────
	case "hud":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "hud", Detail: "expected on or off", Example: "hud on"}
		}
		switch strings.ToLower(args[0]) {
		case "on":
			return HUD{Visible: true}, nil
		case "off":
			return HUD{Visible: false}, nil
		default:
			return nil, ErrUsage{Cmd: "hud", Detail: fmt.Sprintf("unknown argument %q", args[0]), Example: "hud on"}
		}

	default:
		return nil, ErrUnknownCommand{Input: fields[0]}
	}
}
