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

// WindowFullscreen enters (On=true) or exits (On=false) true fullscreen mode.
//
//	window full on
//	window full off
type WindowFullscreen struct{ On bool }

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

// HUD enables or disables the heads-up display overlay (master switch).
//
//	hud on | hud off
type HUD struct {
	Visible bool
}

// HUDList prints the current visibility state for each HUD category.
//
//	hud list
type HUDList struct{}

// HUDCategory enables or disables a single HUD category.
// Category is one of: debug, info, help, player.
//
//	hud debug on | hud info off | hud help on
type HUDCategory struct {
	Category string // "debug" | "info" | "help" | "player"
	Visible  bool
}

// Labels sets the object label display mode.
//
//	labels on | labels off | labels nearest
type Labels struct {
	Mode string // "on" | "off" | "nearest"
}

// Infra sets the infrastructure ambient-light mode.
// Mode 0 = off, 1 = spotlight (boost ambient for objects in camera FOV centre),
// 2 = reserved / deferred.
//
//	infra 0 | infra 1 | infra 2
type Infra struct {
	Mode int
}

// Sync enables or disables synchronous mode.
// When on, animated commands (nav jump, orbit) block until the animation
// finishes before the REPL reads the next line.
//
//	sync on | sync off
type Sync struct {
	On bool
}

// RecordStart begins a new video recording session.
//
//	record start <filename>   e.g.  record start inner-tour.mp4
//
// If filename is a bare name (no path separators) it is saved to ~/Desktop/.
// An .mp4 extension is appended automatically when missing.
type RecordStart struct {
	Filename string
}

// RecordPause toggles the freeze-frame pause state of the active recording.
//
//	record pause
type RecordPause struct{}

// RecordStop finalises and closes the active recording.
//
//	record stop
type RecordStop struct{}

// RecordDelete deletes a previously saved recording file.
// Path resolution follows the same rules as RecordStart.
//
//	record delete <filename>
type RecordDelete struct {
	Filename string
}

// ConfigReloadKeybindings reloads keybindings from the configured path and
// hot-swaps the active KeyMap.
//
//	config reload keybindings
type ConfigReloadKeybindings struct{}

// HelpKeys prints the active key binding table.
//
//	help keys
type HelpKeys struct{}

// ─── Session commands ────────────────────────────────────────────────────────

// SessionRegister registers this REPL client as a new session.
//
//	session register [label]
type SessionRegister struct{ Label string }

// SessionUnregister removes the current client session.
//
//	session unregister
type SessionUnregister struct{}

// SessionList lists all currently registered sessions.
//
//	session list
type SessionList struct{}

// SessionKick removes a connected client session (admin only).
//
//	session kick <session_id>
type SessionKick struct{ TargetSessionID string }

// SessionTeleport moves a session to a named body's position (admin only).
//
//	session teleport <session_id> <body_name>
type SessionTeleport struct {
	TargetSessionID string
	Body            string
}

func (SystemList) isCmd()              {}
func (SystemGet) isCmd()               {}
func (SystemLoad) isCmd()              {}
func (WindowGet) isCmd()               {}
func (WindowSize) isCmd()              {}
func (WindowMaximize) isCmd()          {}
func (WindowRestore) isCmd()           {}
func (WindowFullscreen) isCmd()        {}
func (CameraGet) isCmd()               {}
func (CameraCenter) isCmd()            {}
func (CameraOrient) isCmd()            {}
func (CameraPosition) isCmd()          {}
func (CameraTrack) isCmd()             {}
func (NavStop) isCmd()                 {}
func (NavVelocity) isCmd()             {}
func (NavMove) isCmd()                 {}
func (NavJump) isCmd()                 {}
func (PerfGet) isCmd()                 {}
func (PerfSet) isCmd()                 {}
func (Clear) isCmd()                   {}
func (Shutdown) isCmd()                {}
func (Orbit) isCmd()                   {}
func (Sleep) isCmd()                   {}
func (HUD) isCmd()                     {}
func (HUDList) isCmd()                 {}
func (HUDCategory) isCmd()             {}
func (Labels) isCmd()                  {}
func (Infra) isCmd()                   {}
func (Sync) isCmd()                    {}
func (RecordStart) isCmd()             {}
func (RecordPause) isCmd()             {}
func (RecordStop) isCmd()              {}
func (RecordDelete) isCmd()            {}
func (ConfigReloadKeybindings) isCmd() {}
func (HelpKeys) isCmd()                {}
func (SessionRegister) isCmd()         {}
func (SessionUnregister) isCmd()       {}
func (SessionList) isCmd()             {}
func (SessionKick) isCmd()             {}
func (SessionTeleport) isCmd()         {}

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

// tokenize splits a command line into fields, respecting double-quoted strings.
// Quoted tokens have their surrounding quotes stripped.
// Example: tokenize(`orbit "S/2019 S 1" 15 2`) → ["orbit", "S/2019 S 1", "15", "2"]
func tokenize(line string) []string {
	var fields []string
	i := 0
	for i < len(line) {
		// skip whitespace
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			break
		}
		if line[i] == '"' {
			i++ // skip opening quote
			start := i
			for i < len(line) && line[i] != '"' {
				i++
			}
			fields = append(fields, line[start:i])
			if i < len(line) {
				i++ // skip closing quote
			}
		} else {
			start := i
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			fields = append(fields, line[start:i])
		}
	}
	return fields
}

// Parse parses one trimmed input line into a Cmd. It returns
// ErrUnknownCommand or ErrUsage on parse failure. Empty/comment lines return
// nil, nil — the caller should skip them.
func Parse(line string) (Cmd, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := tokenize(line)
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
		if len(args) > 0 && strings.EqualFold(args[0], "keys") {
			return HelpKeys{}, nil
		}
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
			return nil, fmt.Errorf("window: subcommand required (get|size|maximize|restore|full)")
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
		case "full":
			if len(args) < 2 {
				return nil, fmt.Errorf("window full: on|off required")
			}
			switch strings.ToLower(args[1]) {
			case "on":
				return WindowFullscreen{On: true}, nil
			case "off":
				return WindowFullscreen{On: false}, nil
			default:
				return nil, fmt.Errorf("window full: expected on|off, got %q", args[1])
			}
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

	// ── Track (shorthand for camera track) ──────────────────────────────────
	case "track":
		if len(args) == 0 {
			return nil, ErrUsage{Cmd: "track", Detail: "expected body name or stop", Example: "track Earth | track stop"}
		}
		if strings.EqualFold(args[0], "stop") {
			return CameraTrack{Name: ""}, nil
		}
		return CameraTrack{Name: args[0]}, nil

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
		if len(args) == 0 {
			return nil, ErrUsage{Cmd: "hud", Detail: "expected on, off, list, or a category name", Example: "hud on"}
		}
		switch strings.ToLower(args[0]) {
		case "on":
			return HUD{Visible: true}, nil
		case "off":
			return HUD{Visible: false}, nil
		case "list":
			return HUDList{}, nil
		case "debug", "info", "help", "player":
			cat := strings.ToLower(args[0])
			if len(args) < 2 {
				return nil, ErrUsage{Cmd: "hud", Detail: fmt.Sprintf("%s requires on or off", cat), Example: fmt.Sprintf("hud %s on", cat)}
			}
			switch strings.ToLower(args[1]) {
			case "on":
				return HUDCategory{Category: cat, Visible: true}, nil
			case "off":
				return HUDCategory{Category: cat, Visible: false}, nil
			default:
				return nil, ErrUsage{Cmd: "hud", Detail: fmt.Sprintf("expected on or off, got %q", args[1]), Example: fmt.Sprintf("hud %s on", cat)}
			}
		default:
			return nil, ErrUsage{Cmd: "hud", Detail: fmt.Sprintf("unknown argument %q", args[0]), Example: "hud on"}
		}

	// ── Labels ───────────────────────────────────────────────────────────────
	case "label", "labels":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "labels", Detail: "expected on, off, or nearest", Example: "labels on"}
		}
		mode := strings.ToLower(args[0])
		switch mode {
		case "on", "off", "nearest":
			return Labels{Mode: mode}, nil
		default:
			return nil, ErrUsage{Cmd: "labels", Detail: fmt.Sprintf("unknown mode %q", args[0]), Example: "labels on|off|nearest"}
		}

	// ── Infra ────────────────────────────────────────────────────────────────
	case "infra":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "infra", Detail: "expected mode 0, 1, or 2", Example: "infra 1"}
		}
		switch args[0] {
		case "0":
			return Infra{Mode: 0}, nil
		case "1":
			return Infra{Mode: 1}, nil
		case "2":
			return Infra{Mode: 2}, nil
		default:
			return nil, ErrUsage{Cmd: "infra", Detail: fmt.Sprintf("unknown mode %q", args[0]), Example: "infra 0|1|2"}
		}

	// ── Sync ────────────────────────────────────────────────────────────────
	case "sync":
		if len(args) != 1 {
			return nil, ErrUsage{Cmd: "sync", Detail: "expected on or off", Example: "sync on"}
		}
		switch strings.ToLower(args[0]) {
		case "on":
			return Sync{On: true}, nil
		case "off":
			return Sync{On: false}, nil
		default:
			return nil, ErrUsage{Cmd: "sync", Detail: fmt.Sprintf("unknown value %q", args[0]), Example: "sync on|off"}
		}

	// ── Record ───────────────────────────────────────────────────────────────
	case "record":
		if len(args) == 0 {
			return nil, ErrUsage{Cmd: "record", Detail: "expected sub-command", Example: "record start <filename>|pause|stop|delete <filename>"}
		}
		switch strings.ToLower(args[0]) {
		case "start":
			if len(args) < 2 {
				return nil, ErrUsage{Cmd: "record start", Detail: "expected filename", Example: "record start inner-tour.mp4"}
			}
			return RecordStart{Filename: args[1]}, nil
		case "pause":
			return RecordPause{}, nil
		case "stop":
			return RecordStop{}, nil
		case "delete":
			if len(args) < 2 {
				return nil, ErrUsage{Cmd: "record delete", Detail: "expected filename", Example: "record delete inner-tour.mp4"}
			}
			return RecordDelete{Filename: args[1]}, nil
		default:
			return nil, ErrUsage{Cmd: "record", Detail: fmt.Sprintf("unknown sub-command %q", args[0]), Example: "record start|pause|stop|delete"}
		}

	// ── Config ───────────────────────────────────────────────────────────────
	case "config":
		if len(args) < 2 {
			return nil, fmt.Errorf("config: subcommand required (reload keybindings)")
		}
		if strings.EqualFold(args[0], "reload") && strings.EqualFold(args[1], "keybindings") {
			return ConfigReloadKeybindings{}, nil
		}
		return nil, fmt.Errorf("config: unknown subcommand %q %q", args[0], args[1])

	// ── Session ──────────────────────────────────────────────────────────────
	case "session":
		if len(args) == 0 {
			return nil, fmt.Errorf("session: subcommand required (register|unregister|list|kick|teleport)")
		}
		switch args[0] {
		case "register":
			label := ""
			if len(args) > 1 {
				label = strings.Join(args[1:], " ")
			}
			return SessionRegister{Label: label}, nil
		case "unregister":
			return SessionUnregister{}, nil
		case "list":
			return SessionList{}, nil
		case "kick":
			if len(args) < 2 {
				return nil, ErrUsage{Cmd: "session kick", Detail: "session_id required",
					Example: "session kick <session_id>"}
			}
			return SessionKick{TargetSessionID: args[1]}, nil
		case "teleport":
			if len(args) < 3 {
				return nil, ErrUsage{Cmd: "session teleport",
					Detail:  "session_id and body name required",
					Example: "session teleport <session_id> <body_name>"}
			}
			return SessionTeleport{
				TargetSessionID: args[1],
				Body:            strings.Join(args[2:], " "),
			}, nil
		default:
			return nil, fmt.Errorf("session: unknown subcommand %q", args[0])
		}

	default:
		return nil, ErrUnknownCommand{Input: fields[0]}
	}
}
