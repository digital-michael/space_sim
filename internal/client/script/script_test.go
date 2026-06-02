package script_test

import (
	"strings"
	"testing"

	"github.com/digital-michael/space_sim/internal/client/script"
)

// ── ParseSetVar ───────────────────────────────────────────────────────────────

func TestParseSetVar(t *testing.T) {
	cases := []struct {
		in        string
		wantName  string
		wantVal   string
		wantOK    bool
	}{
		{"set $speed 20.0", "$speed", "20.0", true},
		{"set $speed=20.0", "$speed", "20.0", true},
		{"set $speed:=20.0", "$speed", "20.0", true},
		{`set $name "Sol"`, "$name", "Sol", true},      // quotes stripped
		{"SET $X 1", "$X", "1", true},                   // case-insensitive keyword
		{"set $empty", "$empty", "", true},               // no value
		{"set speed 20", "", "", false},                  // no $ prefix
		{"notset $x 1", "", "", false},                   // wrong keyword
		{"", "", "", false},
	}
	for _, c := range cases {
		name, val, ok := script.ParseSetVar(c.in)
		if ok != c.wantOK {
			t.Errorf("ParseSetVar(%q) ok=%v want %v", c.in, ok, c.wantOK)
			continue
		}
		if ok && (name != c.wantName || val != c.wantVal) {
			t.Errorf("ParseSetVar(%q) = (%q,%q) want (%q,%q)", c.in, name, val, c.wantName, c.wantVal)
		}
	}
}

// ── ExpandVars ────────────────────────────────────────────────────────────────

func TestExpandVars(t *testing.T) {
	vars := map[string]string{
		"$speed":     "20.0",
		"$speed_max": "100.0", // longer name must win over $speed prefix
		"$x":         "Earth",
	}
	cases := []struct {
		in   string
		want string
	}{
		{`nav jump "$x" $speed`, `nav jump "Earth" 20.0`},
		{"$speed_max", "100.0"},       // longest-first: $speed_max not $speed + _max
		{"no vars here", "no vars here"},
		{"$unknown stays", "$unknown stays"},
	}
	for _, c := range cases {
		got := script.ExpandVars(c.in, vars)
		if got != c.want {
			t.Errorf("ExpandVars(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestExpandVarsEmpty(t *testing.T) {
	if got := script.ExpandVars("hello $x", nil); got != "hello $x" {
		t.Errorf("got %q want %q", got, "hello $x")
	}
}

// ── StripLineComment ──────────────────────────────────────────────────────────

func TestStripLineComment(t *testing.T) {
	cases := []struct{ in, want string }{
		{"track Sol // go to star", "track Sol"},
		{"no comment", "no comment"},
		{`sleep 1 // wait`, "sleep 1"},
		{`nav jump "http://example.com" // oops`, `nav jump "http://example.com"`}, // inside quote preserved
		{"// full line comment", ""},
	}
	for _, c := range cases {
		got := strings.TrimRight(script.StripLineComment(c.in), " \t")
		if got != c.want {
			t.Errorf("StripLineComment(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

// ── ParseForHeader ────────────────────────────────────────────────────────────

func TestParseForHeader(t *testing.T) {
	cases := []struct {
		in      string
		wantOK  bool
		isCount bool
		count   int
		group   string
		varName string
		slice   string
	}{
		{"for $x in planets", true, false, 0, "planets", "$x", ""},
		{"for $p in moons[:10]", true, false, 0, "moons", "$p", "[:10]"},
		{"for $x in bodies", true, false, 0, "bodies", "$x", ""},
		{"for $x in bodies planets", true, false, 0, "planets", "$x", ""},
		{"for $x in dwarf_planets[3:]", true, false, 0, "dwarf_planets", "$x", "[3:]"},
		{"for 5", true, true, 5, "", "", ""},        // count loop
		{"for 1", true, true, 1, "", "", ""},
		{"for planets[2:] as $p:", true, false, 0, "planets", "$p", "[2:]"}, // legacy
		{"for $x", false, false, 0, "", "", ""},     // too few args
		{"notfor $x in planets", false, false, 0, "", "", ""},
		{"", false, false, 0, "", "", ""},
	}
	for _, c := range cases {
		fh, ok := script.ParseForHeader(c.in)
		if ok != c.wantOK {
			t.Errorf("ParseForHeader(%q) ok=%v want %v", c.in, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if fh.IsCount != c.isCount || fh.Count != c.count ||
			fh.Group != c.group || fh.VarName != c.varName || fh.SliceSpec != c.slice {
			t.Errorf("ParseForHeader(%q) = %+v want {IsCount:%v Count:%d Group:%q VarName:%q SliceSpec:%q}",
				c.in, fh, c.isCount, c.count, c.group, c.varName, c.slice)
		}
	}
}

// ── ApplyForSlice ─────────────────────────────────────────────────────────────

func TestApplyForSlice(t *testing.T) {
	names := []string{"a", "b", "c", "d", "e"}
	cases := []struct {
		spec    string
		want    []string
		wantErr bool
	}{
		{"", names, false},
		{"[1:]", []string{"b", "c", "d", "e"}, false},
		{"[:3]", []string{"a", "b", "c"}, false},
		{"[1:3]", []string{"b", "c"}, false},
		{"[-2:]", []string{"d", "e"}, false},
		{"[10:]", []string{}, false}, // beyond end → empty
		{"[bad]", nil, true},         // no colon
		{"[a:b]", nil, true},         // non-numeric
	}
	for _, c := range cases {
		got, err := script.ApplyForSlice(names, c.spec)
		if (err != nil) != c.wantErr {
			t.Errorf("ApplyForSlice(%q) err=%v wantErr=%v", c.spec, err, c.wantErr)
			continue
		}
		if !c.wantErr && !slicesEqual(got, c.want) {
			t.Errorf("ApplyForSlice(%q) = %v want %v", c.spec, got, c.want)
		}
	}
}

// ── Expand — unit ─────────────────────────────────────────────────────────────

func expand(t *testing.T, src string, bodies func(string) []string) []string {
	t.Helper()
	lines, err := script.Expand(strings.NewReader(src), bodies)
	if err != nil {
		t.Fatalf("Expand error: %v", err)
	}
	return lines
}

func TestExpand_empty(t *testing.T) {
	if got := expand(t, "", nil); len(got) != 0 {
		t.Errorf("got %v want []", got)
	}
}

func TestExpand_lineComments(t *testing.T) {
	src := "track Sol // go to star\nsleep 1"
	got := expand(t, src, nil)
	want := []string{"track Sol", "sleep 1"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_blockComment(t *testing.T) {
	src := "/* header */\ntrack Sol\n/* multi\nline */\nsleep 1"
	got := expand(t, src, nil)
	want := []string{"track Sol", "sleep 1"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_variables(t *testing.T) {
	src := "set $speed 20.0\nset $target Sol\nnav jump \"$target\" $speed"
	got := expand(t, src, nil)
	want := []string{`nav jump "Sol" 20.0`}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_longestVarFirst(t *testing.T) {
	src := "set $orbs 1\nset $orbspd 18\norbit X $orbspd $orbs"
	got := expand(t, src, nil)
	// $orbspd must expand before $orbs, so result is "orbit X 18 1" not "orbit X 181"
	want := []string{"orbit X 18 1"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_forLoop(t *testing.T) {
	src := "for $x in planets\nnav jump \"$x\"\nendfor"
	bodies := func(group string) []string {
		if group == "planets" {
			return []string{"Mercury", "Venus", "Earth"}
		}
		return nil
	}
	got := expand(t, src, bodies)
	want := []string{
		`nav jump "Mercury"`,
		`nav jump "Venus"`,
		`nav jump "Earth"`,
	}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_forLoopWithVarInBody(t *testing.T) {
	src := "set $speed 10\nfor $x in stars\nnav jump \"$x\" $speed\nendfor"
	bodies := func(group string) []string {
		if group == "stars" {
			return []string{"Sol"}
		}
		return nil
	}
	got := expand(t, src, bodies)
	want := []string{`nav jump "Sol" 10`}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_forLoopSlice(t *testing.T) {
	src := "for $x in moons[:2]\nnav jump \"$x\"\nendfor"
	bodies := func(group string) []string {
		if group == "moons" {
			return []string{"Moon", "Phobos", "Deimos", "Io"}
		}
		return nil
	}
	got := expand(t, src, bodies)
	want := []string{`nav jump "Moon"`, `nav jump "Phobos"`}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_countLoop(t *testing.T) {
	src := "for 3\nping\nendfor"
	got := expand(t, src, nil)
	want := []string{"ping", "ping", "ping"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_nestedForLoop(t *testing.T) {
	src := "for $x in groups\nfor $y in items\nvisit $x $y\nendfor\nendfor"
	bodies := func(group string) []string {
		switch group {
		case "groups":
			return []string{"A", "B"}
		case "items":
			return []string{"1", "2"}
		}
		return nil
	}
	got := expand(t, src, bodies)
	want := []string{"visit A 1", "visit A 2", "visit B 1", "visit B 2"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_sleepPassthrough(t *testing.T) {
	src := "sleep 1.5\ntrack Sol\nsleep 2"
	got := expand(t, src, nil)
	// sleep lines pass through unchanged for callers to handle
	want := []string{"sleep 1.5", "track Sol", "sleep 2"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_syncPassthrough(t *testing.T) {
	src := "sync on\ntrack Sol\nsync off"
	got := expand(t, src, nil)
	want := []string{"sync on", "track Sol", "sync off"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestExpand_nilBodiesForLoop(t *testing.T) {
	// With nil bodies, for loops over groups produce no lines (empty body list).
	src := "for $x in planets\nnav jump \"$x\"\nendfor\ntail"
	got := expand(t, src, nil)
	want := []string{"tail"}
	if !slicesEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

// ── Expand — cross-target consistency ────────────────────────────────────────
//
// These tests verify that the shared language semantics are identical for both
// the direct app and REPL consumers. If you change a parsing function, run
// these to confirm both targets would see the same expanded output.

func TestExpand_solarTourStyle(t *testing.T) {
	// Mirrors the structure of scripts/solar-tour.txt.
	src := `
/* header */
hud off
hud info on
sync on

set $orbits 1
set $speed 20.0
set $sleep 1.5

sleep $sleep

for $x in stars
   nav jump "$x" $speed
   orbit "$x" $speed $orbits
   sleep $sleep
endfor

nav jump Sol $speed
sleep 5
sync off
`
	bodies := func(group string) []string {
		if group == "stars" {
			return []string{"Sol"}
		}
		return nil
	}
	got := expand(t, src, bodies)

	// Verify key structural properties without hardcoding every line.
	assertContains(t, got, "hud off")
	assertContains(t, got, "hud info on")
	assertContains(t, got, `nav jump "Sol" 20.0`)
	assertContains(t, got, `orbit "Sol" 20.0 1`)
	assertContains(t, got, "sleep 1.5")
	assertContains(t, got, "nav jump Sol 20.0")
	assertContains(t, got, "sleep 5")
	assertNotContains(t, got, "for $x in stars") // for header must not appear
	assertNotContains(t, got, "endfor")
	assertNotContains(t, got, "$speed")  // variables fully expanded
	assertNotContains(t, got, "$orbits")
}

func TestExpand_groupToCategory_coverage(t *testing.T) {
	// Verify all GroupToCategory keys are recognised by ParseForHeader so
	// adding a new group to the map doesn't silently break script expansion.
	for group := range script.GroupToCategory {
		line := "for $x in " + group
		fh, ok := script.ParseForHeader(line)
		if !ok {
			t.Errorf("ParseForHeader(%q): group %q from GroupToCategory not recognised", line, group)
			continue
		}
		if fh.Group != group {
			t.Errorf("ParseForHeader(%q).Group = %q want %q", line, fh.Group, group)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func assertContains(t *testing.T, lines []string, want string) {
	t.Helper()
	for _, l := range lines {
		if l == want {
			return
		}
	}
	t.Errorf("output missing line %q\ngot: %v", want, lines)
}

func assertNotContains(t *testing.T, lines []string, substr string) {
	t.Helper()
	for _, l := range lines {
		if strings.Contains(l, substr) {
			t.Errorf("output should not contain %q but found %q", substr, l)
		}
	}
}
