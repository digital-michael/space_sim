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
//   setspeed <seconds_per_second>   e.g.  setspeed 10
type SetSpeed struct {
	SecondsPerSecond float32
}

// GetSpeed queries the current simulation speed.
//   getspeed
type GetSpeed struct{}

// SetDataset changes the active asteroid dataset.
//   setdataset <small|medium|large|huge>
type SetDataset struct {
	Level string // normalised to lower-case
}

// GetDataset queries the active asteroid dataset.
//   getdataset
type GetDataset struct{}

// GetTime queries the current simulation time (seconds since J2000).
//   gettime
type GetTime struct{}

// Stream opens a server-pushed snapshot stream and prints bodies until
// the user presses Ctrl-C.
//   stream
type Stream struct{}

// Help prints the command reference.
//   help
type Help struct{}

// Quit exits the REPL.
//   quit  |  exit
type Quit struct{}

func (SetSpeed) isCmd()   {}
func (GetSpeed) isCmd()   {}
func (SetDataset) isCmd() {}
func (GetDataset) isCmd() {}
func (GetTime) isCmd()    {}
func (Stream) isCmd()     {}
func (Help) isCmd()       {}
func (Quit) isCmd()       {}

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
		if err != nil || v <= 0 {
			return nil, ErrUsage{Cmd: "setspeed", Detail: "argument must be a positive number", Example: "setspeed 10"}
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

	case "stream":
		return Stream{}, nil

	case "help":
		return Help{}, nil

	case "quit", "exit":
		return Quit{}, nil

	default:
		return nil, ErrUnknownCommand{Input: fields[0]}
	}
}
