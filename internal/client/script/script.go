// Package script provides the shared script language implementation for space-sim.
//
// # Extension point
//
// Add new script language features HERE and nowhere else:
//   - New directives (e.g. "repeat N", "if", "label")
//   - New comment styles
//   - New variable syntax
//   - New loop forms
//   - Changes to slice notation
//
// Both consumers use this package:
//   - space-sim-direct  → internal/client/go/raylib/app/script_runner.go
//                         calls Expand() once upfront, then iterates flat lines.
//   - space-sim-repl    → internal/client/repl/repl.go
//                         keeps its streaming execution model but delegates
//                         all language parsing to the exported functions below.
//
// Do NOT add script language logic to script_runner.go or repl.go.
package script

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// GroupToCategory maps human-friendly for-loop group names to the body category
// strings used in snapshot data. Both consumers reference this map so that group
// names stay consistent across direct and REPL targets.
var GroupToCategory = map[string]string{
	"stars":         "star",
	"star":          "star",
	"planets":       "planet",
	"planet":        "planet",
	"dwarf_planets": "dwarf_planet",
	"dwarf_planet":  "dwarf_planet",
	"moons":         "moon",
	"moon":          "moon",
	"asteroids":     "asteroid",
	"asteroid":      "asteroid",
	"systems":       "system",
	"system":        "system",
}

// ForHeader is the parsed result of a for-loop header line.
type ForHeader struct {
	// Count loop: "for <n>"
	IsCount bool
	Count   int
	// Iteration loop
	Group     string // normalised group name, e.g. "planets"
	VarName   string // e.g. "$p"
	SliceSpec string // e.g. "[3:]", empty means all
}

// ParseSetVar parses a "set $name value" line.
// Accepts: "set $name value", "set $name=value", "set $name:=value".
// The value has surrounding quotes stripped if present.
// Returns (name, value, true) on success; ("", "", false) if not a set line.
func ParseSetVar(line string) (name, value string, ok bool) {
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

// ExpandVars replaces every $name occurrence in line with its value from vars.
// Longer variable names are substituted first to avoid prefix collisions
// (e.g. $speed_max before $speed).
func ExpandVars(line string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(line, "$") {
		return line
	}
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

// StripLineComment removes everything from the first unquoted "//" to end of line.
// "//" inside a double-quoted token is preserved.
func StripLineComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			inQuote = !inQuote
		}
		if !inQuote && i+1 < len(line) && line[i] == '/' && line[i+1] == '/' {
			return strings.TrimRight(line[:i], " \t")
		}
	}
	return line
}

// ParseForHeader recognises all accepted for-loop header forms:
//
//	for $v in bodies           — all named bodies
//	for $v in bodies planets   — bodies of a specific group
//	for $v in planets          — shorthand: group only
//	for <group>[slice] as $v:  — legacy form
//	for <n>                    — count loop (n ≥ 1)
//
// Returns (ForHeader, true) on success.
func ParseForHeader(line string) (ForHeader, bool) {
	stripped := strings.TrimSuffix(strings.TrimSpace(line), ":")
	fields := strings.Fields(stripped)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "for") {
		return ForHeader{}, false
	}
	args := fields[1:]
	if len(args) == 0 {
		return ForHeader{}, false
	}

	// Count loop: "for <n>"
	if len(args) == 1 {
		n, err := strconv.Atoi(args[0])
		if err == nil && n >= 1 {
			return ForHeader{IsCount: true, Count: n}, true
		}
		return ForHeader{}, false
	}

	// "for $v in <group>" or "for $v in bodies [group]"
	if strings.EqualFold(args[1], "in") {
		varName := args[0]
		if len(args) < 3 {
			return ForHeader{}, false
		}
		groupToken := strings.ToLower(args[2])
		var sliceSpec string
		if idx := strings.IndexByte(groupToken, '['); idx != -1 {
			sliceSpec = groupToken[idx:]
			groupToken = groupToken[:idx]
		}
		if groupToken == "bodies" {
			if len(args) >= 4 {
				cat := strings.ToLower(args[3])
				if idx := strings.IndexByte(cat, '['); idx != -1 {
					sliceSpec = cat[idx:]
					cat = cat[:idx]
				}
				return ForHeader{Group: cat, VarName: varName, SliceSpec: sliceSpec}, true
			}
			return ForHeader{Group: "bodies", VarName: varName, SliceSpec: sliceSpec}, true
		}
		return ForHeader{Group: groupToken, VarName: varName, SliceSpec: sliceSpec}, true
	}

	// Legacy form: "for <group>[slice] as $v"
	if len(args) == 3 && strings.EqualFold(args[1], "as") {
		groupToken := strings.ToLower(args[0])
		var sliceSpec string
		if idx := strings.IndexByte(groupToken, '['); idx != -1 {
			sliceSpec = groupToken[idx:]
			groupToken = groupToken[:idx]
		}
		return ForHeader{Group: groupToken, VarName: args[2], SliceSpec: sliceSpec}, true
	}

	return ForHeader{}, false
}

// ApplyForSlice restricts names to the sub-range described by spec.
// Supported forms: [start:] [:end] [start:end] [-n:] (negative start = from end).
func ApplyForSlice(names []string, spec string) ([]string, error) {
	if spec == "" {
		return names, nil
	}
	n := len(names)
	if !strings.HasPrefix(spec, "[") || !strings.HasSuffix(spec, "]") {
		return nil, fmt.Errorf("for: invalid slice spec %q", spec)
	}
	inner := spec[1 : len(spec)-1]
	parts := strings.SplitN(inner, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("for: slice spec must contain ':' (got %q)", spec)
	}
	start, end := 0, n
	if parts[0] != "" {
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("for: invalid slice start %q", parts[0])
		}
		if v < 0 {
			start = n + v
		} else {
			start = v
		}
	}
	if parts[1] != "" {
		v, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("for: invalid slice end %q", parts[1])
		}
		if v < 0 {
			return nil, fmt.Errorf("for: negative end not supported (use negative start for last-N, e.g. [-5:])")
		}
		end = v
	}
	if start < 0 {
		start = 0
	}
	if start > n {
		start = n
	}
	if end > n {
		end = n
	}
	if end < start {
		end = start
	}
	return names[start:end], nil
}

// Expand reads a script from src, strips comments, expands variables and
// for/endfor loops, and returns a flat slice of concrete lines ready for
// commands.Parse. sleep and sync lines are passed through unchanged for
// callers to handle.
//
// bodies is called to resolve group names in for loops (e.g. "stars",
// "planets"). It receives the lower-cased group name and must return the
// ordered list of body names for that group. Pass nil to leave for-loop
// variable names un-substituted (useful when body names are not needed).
//
// Expand is suited for whole-file script processing where system state does
// not change during expansion. The direct binary (script_runner.go) uses it.
// The REPL uses the individual Parse*/Expand* functions for its streaming loop.
func Expand(src io.Reader, bodies func(string) []string) ([]string, error) {
	lines, err := readAndStrip(src)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]string)
	return expandLines(lines, 0, vars, bodies)
}

// readAndStrip reads all lines from src, strips // line comments and /* */
// block comments, and returns non-empty trimmed lines.
func readAndStrip(src io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(src)
	var out []string
	inBlock := false
	for scanner.Scan() {
		line := scanner.Text()
		line, inBlock = stripBlockState(line, inBlock)
		line = StripLineComment(line)
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out, scanner.Err()
}

// stripBlockState removes /* */ block comment content from a single line,
// updating the inBlock flag for multi-line blocks.
func stripBlockState(line string, inBlock bool) (string, bool) {
	for {
		if inBlock {
			if idx := strings.Index(line, "*/"); idx >= 0 {
				line = line[idx+2:]
				inBlock = false
			} else {
				return "", true
			}
		} else {
			if idx := strings.Index(line, "/*"); idx >= 0 {
				pre := line[:idx]
				line = line[idx+2:]
				rest, newBlock := stripBlockState(line, true)
				return strings.TrimSpace(pre + rest), newBlock
			}
			return line, false
		}
	}
}

// expandLines processes a pre-stripped line slice recursively, handling
// variable assignment, variable expansion, and for/endfor loop unrolling.
// start is the index to begin at; returns the expanded lines.
func expandLines(lines []string, start int, vars map[string]string, bodies func(string) []string) ([]string, error) {
	var result []string
	i := start
	for i < len(lines) {
		line := lines[i]

		// set $var — store before expansion so RHS is literal.
		if name, val, ok := ParseSetVar(line); ok {
			vars[name] = val
			i++
			continue
		}

		// Expand $vars.
		line = ExpandVars(line, vars)

		// For loop header.
		if fh, ok := ParseForHeader(line); ok {
			body, after, err := collectBodyLines(lines, i+1)
			if err != nil {
				return nil, err
			}
			i = after

			expanded, err := unrollLoop(fh, body, vars, bodies)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded...)
			continue
		}

		// Skip bare endfor that slipped through (shouldn't happen in valid scripts).
		if strings.EqualFold(line, "endfor") {
			i++
			continue
		}

		if line != "" {
			result = append(result, line)
		}
		i++
	}
	return result, nil
}

// collectBodyLines scans lines[start:] for the matching endfor at depth 0.
// Returns (body lines, index-after-endfor, error).
func collectBodyLines(lines []string, start int) ([]string, int, error) {
	depth := 0
	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if _, ok := ParseForHeader(line); ok {
			depth++
		}
		if strings.EqualFold(line, "endfor") {
			if depth == 0 {
				return lines[start:i], i + 1, nil
			}
			depth--
		}
	}
	// EOF without endfor — treat remainder as body.
	return lines[start:], len(lines), nil
}

// unrollLoop expands a for loop into concrete lines.
func unrollLoop(fh ForHeader, body []string, vars map[string]string, bodies func(string) []string) ([]string, error) {
	if fh.IsCount {
		var result []string
		for n := 0; n < fh.Count; n++ {
			// Copy vars so inner assignments don't leak across iterations.
			iterVars := copyVars(vars)
			lines, err := expandLines(body, 0, iterVars, bodies)
			if err != nil {
				return nil, err
			}
			result = append(result, lines...)
		}
		return result, nil
	}

	// Resolve body names.
	var names []string
	if bodies != nil {
		names = bodies(fh.Group)
	}
	names, err := ApplyForSlice(names, fh.SliceSpec)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, name := range names {
		iterVars := copyVars(vars)
		// Substitute the loop variable in each body line before further expansion.
		substituted := make([]string, len(body))
		for j, bl := range body {
			substituted[j] = strings.ReplaceAll(bl, fh.VarName, name)
		}
		lines, err := expandLines(substituted, 0, iterVars, bodies)
		if err != nil {
			return nil, err
		}
		result = append(result, lines...)
	}
	return result, nil
}

func copyVars(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
