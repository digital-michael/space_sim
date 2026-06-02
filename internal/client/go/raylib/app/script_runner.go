package app

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/digital-michael/space_sim/internal/client/commands"
	"github.com/digital-michael/space_sim/internal/client/go/raylib/ui"
)

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
// execute the given script file. Lines are fed to the main thread via
// ScriptLineCmd; sleep commands are honoured in the goroutine itself.
func (a *App) startScript(ctx context.Context, path string) {
	// Cancel any previous script.
	if a.scriptCancel != nil {
		a.scriptCancel()
	}
	scriptCtx, cancel := context.WithCancel(ctx)
	a.scriptCancel = cancel

	go func() {
		defer cancel()
		a.runScript(scriptCtx, path)
	}()
}

// runScript reads a script file line by line and dispatches commands.
// Block comments (/* ... */) and line comments (// ...) are stripped.
// Variable substitution ($name) mirrors the space-sim-repl behaviour:
//   set $name value  — stores a variable locally in the goroutine
//   $name            — expanded in subsequent lines before dispatch
//
// sleep <N> is handled locally; all other lines are forwarded as ScriptLineCmd.
func (a *App) runScript(ctx context.Context, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	vars := make(map[string]string)
	inBlock := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()

		// Strip block comments.
		for {
			if inBlock {
				if idx := strings.Index(line, "*/"); idx >= 0 {
					line = line[idx+2:]
					inBlock = false
				} else {
					line = ""
					break
				}
			} else {
				if idx := strings.Index(line, "/*"); idx >= 0 {
					line = line[:idx]
					inBlock = true
				} else {
					break
				}
			}
		}

		// Strip line comment.
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle set $var before expansion so the RHS is stored literally.
		if name, val, ok := scriptParseSetVar(line); ok {
			vars[name] = val
			continue
		}

		// Expand $vars before parsing or forwarding.
		line = scriptExpandVars(line, vars)

		// Handle sleep locally so it doesn't block the main thread.
		cmd, err := commands.Parse(line)
		if err != nil {
			continue
		}
		if s, ok := cmd.(commands.Sleep); ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(float64(time.Second) * s.Seconds)):
			}
			continue
		}

		// Forward everything else to the main thread.
		if !a.SendCmd(ScriptLineCmd{Line: line}) {
			// Channel full — brief backoff then retry once.
			time.Sleep(16 * time.Millisecond)
			a.SendCmd(ScriptLineCmd{Line: line})
		}

		// Small yield between commands so the render loop can process each one.
		select {
		case <-ctx.Done():
			return
		case <-time.After(8 * time.Millisecond):
		}
	}
}

// scriptParseSetVar parses "set $name value" lines.
// Returns (name, value, true) on success; ("", "", false) otherwise.
func scriptParseSetVar(line string) (name, value string, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(line))
	if !strings.HasPrefix(lower, "set ") {
		return "", "", false
	}
	rest := strings.TrimSpace(line[4:])
	if !strings.HasPrefix(rest, "$") {
		return "", "", false
	}
	var rawName, rawVal string
	if idx := strings.Index(rest, ":="); idx > 0 {
		rawName = rest[:idx]
		rawVal = strings.TrimSpace(rest[idx+2:])
	} else if idx := strings.IndexAny(rest, "= \t"); idx > 0 {
		rawName = rest[:idx]
		rawVal = strings.TrimSpace(strings.TrimLeft(rest[idx:], "= \t"))
	} else {
		rawName = rest
		rawVal = ""
	}
	if len(rawVal) >= 2 && rawVal[0] == '"' && rawVal[len(rawVal)-1] == '"' {
		rawVal = rawVal[1 : len(rawVal)-1]
	}
	return rawName, rawVal, true
}

// scriptExpandVars replaces every $name occurrence in line with its stored value.
// Longer names are substituted first to avoid partial matches.
func scriptExpandVars(line string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(line, "$") {
		return line
	}
	// Sort keys longest-first so $orbspd is matched before $orbs.
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		line = strings.ReplaceAll(line, k, vars[k])
	}
	return line
}
