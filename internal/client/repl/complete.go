package repl

import "strings"

// ─── Completion table ─────────────────────────────────────────────────────────

// topVerbs is the set of first-word commands the REPL accepts.
var topVerbs = []string{
	"setspeed", "getspeed", "pause", "resume",
	"setdataset", "getdataset",
	"gettime", "bodies", "inspect", "status", "stream",
	"system", "window", "camera", "nav", "perf",
	"orbit", "sleep", "hud", "labels", "set",
	"shutdown", "clear", "help", "quit", "exit",
}

// subCmds maps a multi-word verb to its valid sub-commands.
var subCmds = map[string][]string{
	"system": {"list", "get", "load"},
	"window": {"get", "size", "maximize", "restore", "full"},
	"camera": {"get", "center", "orient", "position", "track"},
	"nav":    {"stop", "velocity", "forward", "back", "left", "right", "up", "down", "jump"},
	"perf":   {"get", "set"},
	"hud":    {"on", "off", "list", "debug", "info", "help", "player"},
	"labels": {"on", "off", "nearest"},
}

// perfFields is the sorted list of valid perf set field names.
var perfFields = []string{
	"camera_speed",
	"frustum_culling",
	"importance_threshold",
	"instanced_rendering",
	"lod_enabled",
	"point_rendering",
	"spatial_partition",
	"use_in_place_swap",
	"workers",
}

// ─── Completer function ───────────────────────────────────────────────────────

// replComplete returns the list of completions for a partial REPL input line.
// Every returned string is a complete replacement for the whole input line —
// the caller (lineReader.replaceLine) writes the result verbatim.
//
// bodies is the cached list of known body names (e.g. "Earth", "Sun"); it may
// be nil or empty when body-name completions are not yet available.
//
// Rules:
//   - Single token (no space yet): complete against topVerbs
//   - "verb <partial>": complete the sub-command, return "verb subCmd"
//   - "nav jump " / "nav jump <partial>": complete against bodies (+ "clear")
//   - "camera track " / "camera track <partial>": complete against bodies
//   - "perf set <partial>": complete against perfFields, return "perf set field"
//   - All other multi-token positions: no completions
func replComplete(partial string, bodies []string) []string {
	fields := strings.Fields(partial)

	switch len(fields) {
	case 0:
		// Empty input: offer all verbs.
		return topVerbs

	case 1:
		if strings.HasSuffix(partial, " ") {
			// "verb ": offer full-line "verb subcmd" completions.
			verb := fields[0]
			// "orbit ": complete against body names for the first positional arg.
			if verb == "orbit" {
				return fullLine("orbit ", bodies)
			}
			return fullLine(verb+" ", subCmds[verb])
		}
		return prefixMatch(fields[0], topVerbs)

	case 2:
		verb := fields[0]
		partialSub := fields[1]
		if strings.HasSuffix(partial, " ") {
			// "nav jump ": offer "clear" then all known bodies.
			if verb == "nav" && partialSub == "jump" {
				return fullLine("nav jump ", append([]string{"clear"}, bodies...))
			}
			// "camera track ": offer all known bodies.
			if verb == "camera" && partialSub == "track" {
				return fullLine("camera track ", bodies)
			}
			// "perf set ": offer "perf set <field>" completions.
			if verb == "perf" && partialSub == "set" {
				return fullLine("perf set ", perfFields)
			}
			return nil
		}
		// "orbit <partial>": complete the body name.
		if verb == "orbit" {
			return fullLine("orbit ", prefixMatch(partialSub, bodies))
		}
		// Completing the sub-command token — return full lines.
		subs := subCmds[verb]
		return fullLine(verb+" ", prefixMatch(partialSub, subs))

	case 3:
		// "nav jump <partial>": complete against bodies + "clear".
		if fields[0] == "nav" && fields[1] == "jump" {
			if strings.HasSuffix(partial, " ") {
				return nil
			}
			candidates := append([]string{"clear"}, bodies...)
			return fullLine("nav jump ", prefixMatch(fields[2], candidates))
		}
		// "camera track <partial>": complete against bodies.
		if fields[0] == "camera" && fields[1] == "track" {
			if strings.HasSuffix(partial, " ") {
				return nil
			}
			return fullLine("camera track ", prefixMatch(fields[2], bodies))
		}
		// "perf set <partial>"
		if fields[0] == "perf" && fields[1] == "set" {
			if strings.HasSuffix(partial, " ") {
				return nil
			}
			return fullLine("perf set ", prefixMatch(fields[2], perfFields))
		}
		return nil

	default:
		return nil
	}
}

// needsBodyNames reports whether completing partial requires the body name
// cache. Called before replComplete to decide whether to refresh the cache.
func needsBodyNames(partial string) bool {
	fields := strings.Fields(partial)
	switch len(fields) {
	case 1:
		if !strings.HasSuffix(partial, " ") {
			return false
		}
		return fields[0] == "orbit"
	case 2:
		if !strings.HasSuffix(partial, " ") {
			// "orbit <partial>" — need bodies for prefix match.
			return fields[0] == "orbit" ||
				(fields[0] == "nav" && fields[1] == "jump") ||
				(fields[0] == "camera" && fields[1] == "track")
		}
		return (fields[0] == "nav" && fields[1] == "jump") ||
			(fields[0] == "camera" && fields[1] == "track")
	case 3:
		return (fields[0] == "nav" && fields[1] == "jump") ||
			(fields[0] == "camera" && fields[1] == "track")
	}
	return false
}

// fullLine prepends prefix to each element in items and returns the result.
// Returns nil when items is empty.
func fullLine(prefix string, items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = prefix + item
	}
	return out
}

// prefixMatch returns elements from candidates that start with prefix.
func prefixMatch(prefix string, candidates []string) []string {
	var out []string
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}
