package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/client/commands"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
	"github.com/digital-michael/space_sim/internal/client/script"
	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	protocol "github.com/digital-michael/space_sim/internal/protocol"
)

// scriptsDir is the directory scanned for runnable script files.
const scriptsDir = "scripts"

// discoverScripts returns a sorted list of .txt script paths under scripts/.
func discoverScripts() []string {
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".txt" {
			paths = append(paths, filepath.Join(scriptsDir, e.Name()))
		}
	}
	sort.Strings(paths)
	return paths
}

// openScriptSelector populates and activates the script browser dialog.
func (a *App) openScriptSelector(inputState *ui.InputState) {
	inputState.OpenScriptSelector(discoverScripts())
}

// startScript cancels any running script and launches a new goroutine to
// execute the given script file. snap is used to resolve body group names
// (stars, planets, etc.) for for-loop expansion at script start time.
func (a *App) startScript(ctx context.Context, path string, snap protocol.WorldSnapshot) {
	if a.scriptCancel != nil {
		a.scriptCancel()
	}
	scriptCtx, cancel := context.WithCancel(ctx)
	a.scriptCancel = cancel

	bodies := scriptBodiesFromSnapshot(snap)
	go func() {
		defer cancel()
		a.runScript(scriptCtx, path, bodies)
	}()
}

// runScript expands the script file using script.Expand, then dispatches each
// concrete line to the main thread as a ScriptLineCmd. sleep lines are handled
// locally to avoid blocking the render loop.
//
// Add new script language features to internal/client/script/script.go —
// not here. This function only handles execution timing.
func (a *App) runScript(ctx context.Context, path string, bodies func(string) []string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	lines, err := script.Expand(f, bodies)
	if err != nil {
		return
	}

	for _, line := range lines {
		if ctx.Err() != nil {
			return
		}

		// Handle sleep locally — don't block the main thread.
		cmd, err := commands.Parse(line)
		if err == nil {
			if s, ok := cmd.(commands.Sleep); ok {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(float64(time.Second) * s.Seconds)):
				}
				continue
			}
		}

		// Forward to main thread.
		if !a.SendCmd(ScriptLineCmd{Line: line}) {
			time.Sleep(16 * time.Millisecond)
			a.SendCmd(ScriptLineCmd{Line: line})
		}

		// Small yield so the render loop can process each command.
		select {
		case <-ctx.Done():
			return
		case <-time.After(8 * time.Millisecond):
		}
	}
}

// scriptBodiesFromSnapshot returns a bodies resolver that filters objects from
// the given snapshot by group name (e.g. "stars", "planets").
//
// Add new group name mappings to internal/client/script/script.go (GroupToCategory)
// — not here.
func scriptBodiesFromSnapshot(snap protocol.WorldSnapshot) func(string) []string {
	state := snap.State
	return func(group string) []string {
		if state == nil {
			return nil
		}
		allBodies := strings.ToLower(group) == "bodies"
		category := script.GroupToCategory[strings.ToLower(group)]
		if !allBodies && category == "" {
			return nil
		}
		var names []string
		seen := make(map[string]struct{})
		for _, obj := range state.Objects {
			cg := objectCategoryGroup(obj.Meta.Category)
			if cg == "" {
				continue
			}
			if !allBodies && cg != category {
				continue
			}
			if obj.Meta.Name == "" {
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

// objectCategoryGroup maps an engine ObjectCategory to the category string
// used in script group names and GroupToCategory (e.g. "star", "planet").
//
// When new ObjectCategory values are added to the engine, add their mapping
// here so scripts can iterate them via for-loop groups.
func objectCategoryGroup(cat engine.ObjectCategory) string {
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
