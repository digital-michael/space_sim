// Package decompose splits a system definition into smaller, independently
// loadable system definitions — one per gravitationally meaningful region.
//
// It exists so a large system such as Sol can be partitioned into local
// environments (inner system, Jupiter system, Saturn system, …), each cheap
// enough to run on its own. See docs/wip/f043 tracking in Ledger.
//
// This package is deliberately free of any CLI or transport dependency so it can
// be reused if runtime partitioning is ever wanted. It operates on generic JSON
// rather than typed schema structs, so fields it does not understand survive the
// round trip untouched.
package decompose

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Options controls how a system is partitioned.
type Options struct {
	// MinSatellites is how many descendants a body needs before it earns its own
	// chunk. Bodies below the threshold stay in an inner or outer band.
	MinSatellites int

	// Split forces these body names into their own chunk regardless of
	// satellite count.
	Split []string

	// OutDir is where chunk directories are written. Empty means alongside the
	// source system, which is what authoring into data/systems/ wants; tests
	// redirect it to a temporary directory.
	OutDir string

	// Prefix is prepended to every chunk directory name. Defaults to the source
	// directory's base name.
	Prefix string
}

// Chunk is one partition of the source system.
type Chunk struct {
	// Dir is the emitted directory name, e.g. "solar_system_02_jupiter". The
	// numeric segment makes chunks sort in orbital order in a system selector,
	// and this name becomes the simulation's runtime identity.
	Dir string

	// Label is the short region name, e.g. "Jupiter".
	Label string

	// Display is written to the chunk's system.json "name" field. The system
	// selector sorts and displays by THAT field, not by directory name, so the
	// ordering and family grouping have to live here or they have no effect on
	// what a user sees.
	Display string

	// Roots are the star-orbiting bodies this chunk owns, excluding the star
	// itself, which is injected into every chunk.
	Roots []string

	// Members is every body in the chunk, including satellites and rings.
	Members []string

	// Anchor is the semi-major axis the chunk is ordered by.
	Anchor float64
}

// body is the minimum we need to interpret from an element; the raw form is
// what gets written back out.
type body struct {
	name   string
	parent string
	semi   float64
	raw    json.RawMessage
}

// categoryFile is one of the per-category JSON documents (planets, moons, …).
type categoryFile struct {
	key      string // the "files" key in system.json, e.g. "moons"
	filename string
	arrayKey string         // the document's array property: bodies / belts / rogues / artifacts
	envelope map[string]any // everything except the array, preserved verbatim
	elems    []body
}

// HillRadius returns the radius within which a body's gravity dominates its
// parent star's, in the same length unit as semiMajorAxis.
//
// Kept exported because it is the physically principled way to size a chunk's
// spatial extent. It is deliberately NOT used to decide chunk membership: in
// authored datasets satellite distances are commonly inflated for visibility, so
// real satellites fall outside a Hill radius computed from real masses. The
// declared parent field is authoritative for membership.
func HillRadius(semiMajorAxis, bodyMass, starMass float64) float64 {
	if starMass <= 0 {
		return 0
	}
	if bodyMass <= 0 {
		return 0
	}
	return semiMajorAxis * math.Cbrt(bodyMass/(3*starMass))
}

// Decompose reads the system at srcDir and writes one directory per chunk into
// opts.OutDir, defaulting to the source system's parent. It returns the chunks
// it produced.
func Decompose(srcDir string, opts Options) ([]Chunk, error) {
	if opts.MinSatellites <= 0 {
		opts.MinSatellites = 4
	}
	if opts.Prefix == "" {
		opts.Prefix = filepath.Base(filepath.Clean(srcDir))
	}

	sysDoc, files, err := readSystemDoc(srcDir)
	if err != nil {
		return nil, err
	}

	cats, err := readCategories(srcDir, files)
	if err != nil {
		return nil, err
	}

	all := map[string]body{}
	var starNames []string
	for _, c := range cats {
		for _, b := range c.elems {
			if b.name == "" {
				continue
			}
			all[b.name] = b
		}
		if c.key == "stars" {
			for _, b := range c.elems {
				starNames = append(starNames, b.name)
			}
		}
	}
	if len(starNames) == 0 {
		return nil, fmt.Errorf("%s: no stars found; cannot partition a system without a primary", srcDir)
	}
	stars := map[string]bool{}
	for _, n := range starNames {
		stars[n] = true
	}

	children := map[string][]string{}
	for _, b := range all {
		if b.parent != "" {
			children[b.parent] = append(children[b.parent], b.name)
		}
	}

	// Roots are bodies that orbit a star, plus parentless bodies (probes and
	// comets are authored without a parent), excluding the stars themselves.
	var roots []string
	for name, b := range all {
		if stars[name] {
			continue
		}
		if b.parent == "" || stars[b.parent] {
			roots = append(roots, name)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return all[roots[i]].semi < all[roots[j]].semi })

	forced := map[string]bool{}
	for _, n := range opts.Split {
		forced[n] = true
	}

	split := map[string]bool{}
	for _, r := range roots {
		if forced[r] || len(descendants(children, r)) >= opts.MinSatellites {
			split[r] = true
		}
	}

	// The innermost split body is the inner/outer band boundary. With no splits
	// the whole system is one chunk, which is a valid if useless result.
	innerMax := 0.0
	first := true
	for r := range split {
		if a := all[r].semi; first || a < innerMax {
			innerMax, first = a, false
		}
	}

	outDir := opts.OutDir
	if outDir == "" {
		outDir = filepath.Dir(filepath.Clean(srcDir))
	}

	sourceName, _ := sysDoc["name"].(string)
	if sourceName == "" {
		sourceName = opts.Prefix
	}

	chunks := buildChunks(all, children, roots, split, innerMax, opts.Prefix, sourceName)
	if err := writeChunks(srcDir, outDir, sysDoc, cats, chunks, stars); err != nil {
		return nil, err
	}
	return chunks, nil
}

func buildChunks(
	all map[string]body,
	children map[string][]string,
	roots []string,
	split map[string]bool,
	innerMax float64,
	prefix string,
	sourceName string,
) []Chunk {
	var inner, outer []string
	var solo []string

	for _, r := range roots {
		switch {
		case split[r]:
			solo = append(solo, r)
		case all[r].semi <= innerMax:
			inner = append(inner, r)
		default:
			outer = append(outer, r)
		}
	}

	var out []Chunk
	add := func(label string, slug string, rs []string, anchor float64) {
		if len(rs) == 0 {
			return
		}
		members := map[string]bool{}
		for _, r := range rs {
			members[r] = true
			for _, d := range descendants(children, r) {
				members[d] = true
			}
		}
		out = append(out, Chunk{
			Label:   label,
			Roots:   rs,
			Members: sortedKeys(members),
			Anchor:  anchor,
			Dir:     slug, // numbered below
		})
	}

	// The bands bracket the split bodies by intent, not by radius: the outer
	// band is a catch-all whose innermost member can orbit closer than a split
	// body (Salacia at 2100 su sits inside Pluto at 2460), so ordering it by
	// anchor would place it mid-list and break selector adjacency.
	add("Inner", "inner", inner, math.Inf(-1))
	for _, r := range solo {
		add(r, slug(r), []string{r}, all[r].semi)
	}
	add("Outer", "outer", outer, math.Inf(1))

	sort.SliceStable(out, func(i, j int) bool { return out[i].Anchor < out[j].Anchor })
	for i := range out {
		// Display carries the ordering because the selector sorts on it. Prefixing
		// with the source system's name also groups the family together and places
		// it immediately after the undivided system alphabetically.
		out[i].Display = fmt.Sprintf("%s %02d — %s", sourceName, i+1, out[i].Label)
		out[i].Dir = fmt.Sprintf("%s_%02d_%s", prefix, i+1, out[i].Dir)
	}
	return out
}

func writeChunks(
	srcDir string,
	outDir string,
	sysDoc map[string]any,
	cats []categoryFile,
	chunks []Chunk,
	stars map[string]bool,
) error {
	for _, ch := range chunks {
		dir := filepath.Join(outDir, ch.Dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}

		member := map[string]bool{}
		for _, m := range ch.Members {
			member[m] = true
		}

		kept := map[string]string{}
		for _, c := range cats {
			var sel []json.RawMessage
			for _, b := range c.elems {
				// Stars are injected into every chunk: needed for lighting, and
				// the primary must appear in every local sky.
				if member[b.name] || stars[b.name] {
					sel = append(sel, b.raw)
				}
			}
			if len(sel) == 0 {
				continue // omit empty category files rather than writing stubs
			}
			doc := map[string]any{}
			for k, v := range c.envelope {
				doc[k] = v
			}
			doc[c.arrayKey] = sel
			if err := writeJSON(filepath.Join(dir, c.filename), doc); err != nil {
				return err
			}
			kept[c.key] = c.filename
		}

		sys := map[string]any{}
		for k, v := range sysDoc {
			sys[k] = v
		}
		sys["name"] = ch.Display
		sys["files"] = kept
		sys["decomposed_from"] = filepath.Base(filepath.Clean(srcDir))
		if err := writeJSON(filepath.Join(dir, "system.json"), sys); err != nil {
			return err
		}
	}
	return nil
}

// ── reading ───────────────────────────────────────────────────────────────────

func readSystemDoc(srcDir string) (map[string]any, map[string]string, error) {
	path := filepath.Join(srcDir, "system.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	filesAny, ok := doc["files"].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("%s: missing or malformed \"files\" map", path)
	}
	files := map[string]string{}
	for k, v := range filesAny {
		if s, ok := v.(string); ok {
			files[k] = s
		}
	}
	return doc, files, nil
}

func readCategories(srcDir string, files map[string]string) ([]categoryFile, error) {
	keys := sortedKeys(toSet(files))
	var out []categoryFile

	for _, key := range keys {
		fname := files[key]
		path := filepath.Join(srcDir, fname)
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // a declared but absent category is not an error
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		arrayKey := ""
		var elems []json.RawMessage
		for k, v := range doc {
			var probe []json.RawMessage
			if json.Unmarshal(v, &probe) == nil {
				arrayKey, elems = k, probe
				break
			}
		}
		if arrayKey == "" {
			return nil, fmt.Errorf("%s: no array property found", path)
		}

		envelope := map[string]any{}
		for k, v := range doc {
			if k == arrayKey {
				continue
			}
			var anyv any
			if err := json.Unmarshal(v, &anyv); err != nil {
				return nil, fmt.Errorf("%s: parse %q: %w", path, k, err)
			}
			envelope[k] = anyv
		}

		cf := categoryFile{key: key, filename: fname, arrayKey: arrayKey, envelope: envelope}
		for _, e := range elems {
			cf.elems = append(cf.elems, parseBody(e))
		}
		out = append(out, cf)
	}
	return out, nil
}

func parseBody(raw json.RawMessage) body {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	b := body{raw: raw}
	if s, ok := m["name"].(string); ok {
		b.name = s
	}
	if s, ok := m["parent"].(string); ok {
		b.parent = s
	}
	if orbit, ok := m["orbit"].(map[string]any); ok {
		if f, ok := orbit["semi_major_axis"].(float64); ok {
			b.semi = f
		}
	}
	if dist, ok := m["distribution"].(map[string]any); ok {
		if f, ok := dist["inner_radius"].(float64); ok && b.semi == 0 {
			b.semi = f
		}
	}
	return b
}

// ── helpers ───────────────────────────────────────────────────────────────────

func descendants(children map[string][]string, root string) []string {
	var out []string
	seen := map[string]bool{root: true}
	queue := append([]string{}, children[root]...)
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
		queue = append(queue, children[n]...)
	}
	return out
}

func minSemi(all map[string]body, names []string) float64 {
	m, first := 0.0, true
	for _, n := range names {
		if a := all[n].semi; first || a < m {
			m, first = a, false
		}
	}
	return m
}

func slug(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r == ' ', r == '-', r == '_':
			return '_'
		default:
			return -1
		}
	}, s)
	return strings.Trim(s, "_")
}

func writeJSON(path string, doc map[string]any) error {
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func toSet(m map[string]string) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range m {
		out[k] = struct{}{}
	}
	return out
}
