package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/client/commands"
	"github.com/digital-michael/space_sim/internal/client/script"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	world "github.com/digital-michael/space_sim/internal/sim/world"
)

// runAdminREPL runs the server-side administration REPL on the provided reader
// (normally os.Stdin). It blocks until ctx is cancelled, the reader returns
// EOF, or a "shutdown" / "quit" command is received.
//
// If scriptPath is non-empty the script is executed first; the REPL then falls
// through to interactive mode (stdin).
//
// Add new script language features to internal/client/script — not here.
// Add new dispatchable commands to dispatchAdminCmd — not to this loop.
func runAdminREPL(ctx context.Context, w *world.World, cancel context.CancelFunc, scriptPath string) {
	lastSpeed := w.GetSpeed()

	dispatch := func(line string) (quit bool) {
		cmd, err := commands.Parse(line)
		if err != nil {
			fmt.Printf("[admin] error: %v\n", err)
			return false
		}
		return dispatchAdminCmd(ctx, w, cancel, &lastSpeed, cmd)
	}

	// Execute startup script first.
	if scriptPath != "" {
		bodies := adminBodiesFromWorld(w)
		if err := runAdminScript(ctx, scriptPath, bodies, dispatch); err != nil {
			fmt.Fprintf(os.Stderr, "[admin] script error: %v\n", err)
		}
	}

	// Interactive stdin loop.
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("[admin] ready — type 'help' for commands")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		fmt.Print("> ")
		if !scanner.Scan() {
			return // EOF or error
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if dispatch(line) {
			return
		}
	}
}

// dispatchAdminCmd executes one parsed command against the local world.
// Returns true if the REPL should exit (shutdown/quit).
//
// Extend this function when new simulation-control commands are needed.
func dispatchAdminCmd(ctx context.Context, w *world.World, cancel context.CancelFunc, lastSpeed *float64, cmd commands.Cmd) bool {
	switch v := cmd.(type) {
	case commands.SetSpeed:
		w.SetSpeed(float64(v.SecondsPerSecond))
		*lastSpeed = float64(v.SecondsPerSecond)
		fmt.Printf("[admin] speed = %.1f sim-s/real-s\n", float64(v.SecondsPerSecond))

	case commands.Pause:
		*lastSpeed = w.GetSpeed()
		w.SetSpeed(0)
		fmt.Println("[admin] paused")

	case commands.Resume:
		if *lastSpeed == 0 {
			*lastSpeed = 3600
		}
		w.SetSpeed(*lastSpeed)
		fmt.Printf("[admin] resumed at %.1f sim-s/real-s\n", *lastSpeed)

	case commands.SetDataset:
		var d engine.AsteroidDataset
		switch strings.ToLower(v.Level) {
		case "small":
			d = engine.AsteroidDatasetSmall
		case "medium":
			d = engine.AsteroidDatasetMedium
		case "large":
			d = engine.AsteroidDatasetLarge
		case "huge":
			d = engine.AsteroidDatasetHuge
		default:
			fmt.Printf("[admin] unknown dataset %q — use small/medium/large/huge\n", v.Level)
			return false
		}
		w.SetAsteroidDataset(d)
		fmt.Printf("[admin] dataset = %s\n", v.Level)

	case commands.Bodies:
		snap := w.LatestSnapshot()
		if snap.State == nil || len(snap.State.Objects) == 0 {
			fmt.Println("[admin] no bodies loaded")
			return false
		}
		for _, obj := range snap.State.Objects {
			if obj.Meta.Category != engine.CategoryAsteroid && obj.Meta.Category != engine.CategoryRing {
				fmt.Printf("  %-30s  %s\n", obj.Meta.Name, engineCategoryString(obj.Meta.Category))
			}
		}

	case commands.Status:
		snap := w.LatestSnapshot()
		speed := w.GetSpeed()
		bodies := 0
		if snap.State != nil {
			bodies = len(snap.State.Objects)
		}
		fmt.Printf("[admin] speed=%.1f  bodies=%d  sim-time=%.0f\n", speed, bodies, snap.Speed)

	case commands.Sleep:
		select {
		case <-ctx.Done():
			return true
		case <-time.After(time.Duration(float64(time.Second) * v.Seconds)):
		}

	case commands.Shutdown, commands.Quit:
		fmt.Println("[admin] shutting down")
		cancel()
		return true

	case commands.Help:
		fmt.Print(`[admin] available commands:
  setspeed <n>       — set simulation speed (sim-sec / real-sec)
  pause              — freeze simulation
  resume             — resume at last speed
  setdataset <level> — small | medium | large | huge
  bodies             — list named bodies in current system
  status             — show speed, body count, sim time
  sleep <n>          — pause script execution for n seconds
  shutdown / quit    — stop the server
  help               — this message
`)

	default:
		fmt.Printf("[admin] command not supported in server mode: %T\n", cmd)
	}
	return false
}

// runAdminScript expands a script file and executes each line via dispatch.
// Uses the shared script.Expand function so all script language features
// (variables, for loops, slice notation) work identically here and in
// space-sim-direct / space-sim-repl.
func runAdminScript(ctx context.Context, path string, bodies func(string) []string, dispatch func(string) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	lines, err := script.Expand(f, bodies)
	if err != nil {
		return err
	}

	for _, line := range lines {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// sleep is dispatched through commands.Parse in dispatchAdminCmd.
		if dispatch(line) {
			return nil
		}
	}
	return nil
}

// adminBodiesFromWorld returns a bodies resolver that reads body names from
// the world's latest snapshot — the same group names used by space-sim-direct
// and space-sim-repl scripts.
func adminBodiesFromWorld(w *world.World) func(string) []string {
	return func(group string) []string {
		snap := w.LatestSnapshot()
		if snap.State == nil {
			return nil
		}
		allBodies := strings.ToLower(group) == "bodies"
		category := script.GroupToCategory[strings.ToLower(group)]
		if !allBodies && category == "" {
			return nil
		}
		var names []string
		seen := make(map[string]struct{})
		for _, obj := range snap.State.Objects {
			if obj.Meta.Name == "" {
				continue
			}
			catStr := engineCategoryString(obj.Meta.Category)
			if !allBodies && catStr != category {
				continue
			}
			if _, dup := seen[obj.Meta.Name]; dup {
				continue
			}
			seen[obj.Meta.Name] = struct{}{}
			names = append(names, obj.Meta.Name)
		}
		return names
	}
}

// engineCategoryString maps an ObjectCategory to the category string used in
// GroupToCategory (e.g. "star", "planet"). Mirrors objectCategoryGroup in
// internal/client/go/raylib/app/script_runner.go — keep in sync.
func engineCategoryString(cat engine.ObjectCategory) string {
	switch cat {
	case engine.CategoryStarPreMain, engine.CategoryStarMainSequence,
		engine.CategoryStarEvolved, engine.CategorySubstellar, engine.CategoryStellarRemnant:
		return "star"
	case engine.CategoryPlanet:
		return "planet"
	case engine.CategoryDwarfPlanet:
		return "dwarf_planet"
	case engine.CategoryMoon:
		return "moon"
	case engine.CategoryAsteroid:
		return "asteroid"
	default:
		return ""
	}
}

// adminStdinIsTerminal is a thin shim so the scanner can handle both pipe
// input (non-interactive) and a real terminal gracefully.
func adminStdinIsTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// adminReadLine reads one line from r, trimming CR/LF. Returns ("", io.EOF)
// on EOF.
func adminReadLine(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		_, err := r.Read(b)
		if err != nil {
			return strings.TrimSpace(string(buf)), err
		}
		if b[0] == '\n' {
			break
		}
		if b[0] != '\r' {
			buf = append(buf, b[0])
		}
	}
	return strings.TrimSpace(string(buf)), nil
}
