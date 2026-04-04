package repl

import (
	"reflect"
	"testing"
)

func TestReplComplete_EmptyInput_ReturnsAllVerbs(t *testing.T) {
	got := replComplete("", nil)
	if len(got) == 0 {
		t.Fatal("expected completions for empty input, got none")
	}
	// Spot-check a few expected verbs.
	want := map[string]bool{"system": true, "camera": true, "nav": true, "perf": true, "shutdown": true}
	for _, v := range got {
		delete(want, v)
	}
	if len(want) != 0 {
		t.Errorf("missing expected verbs: %v", want)
	}
}

func TestReplComplete_PartialVerb(t *testing.T) {
	cases := []struct {
		partial string
		want    []string
	}{
		{"sys", []string{"system"}},
		{"sh", []string{"shutdown"}},
		{"cam", []string{"camera"}},
		{"nav", []string{"nav"}},
		{"pe", []string{"perf"}},
		{"wi", []string{"window"}},
		{"xyz", nil},
	}
	for _, tc := range cases {
		got := replComplete(tc.partial, nil)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("replComplete(%q) = %v; want %v", tc.partial, got, tc.want)
		}
	}
}

func TestReplComplete_VerbWithTrailingSpace_ReturnsSubCmds(t *testing.T) {
	cases := []struct {
		partial string
		want    []string
	}{
		{"system ", []string{"system list", "system get", "system load"}},
		{"window ", []string{"window get", "window size", "window maximize", "window restore"}},
		{"camera ", []string{"camera get", "camera center", "camera orient", "camera position", "camera track"}},
		{"nav ", []string{"nav stop", "nav velocity", "nav forward", "nav back", "nav left", "nav right", "nav up", "nav down", "nav jump"}},
		{"perf ", []string{"perf get", "perf set"}},
	}
	for _, tc := range cases {
		got := replComplete(tc.partial, nil)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("replComplete(%q) = %v; want %v", tc.partial, got, tc.want)
		}
	}
}

func TestReplComplete_PartialSubCmd(t *testing.T) {
	cases := []struct {
		partial string
		want    []string
	}{
		{"system l", []string{"system list", "system load"}},
		{"system ge", []string{"system get"}},
		{"window s", []string{"window size"}},
		{"nav f", []string{"nav forward"}},
		{"nav j", []string{"nav jump"}},
		{"perf s", []string{"perf set"}},
	}
	for _, tc := range cases {
		got := replComplete(tc.partial, nil)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("replComplete(%q) = %v; want %v", tc.partial, got, tc.want)
		}
	}
}

func TestReplComplete_PerfSetField(t *testing.T) {
	// "perf set " with trailing space should return all perf fields.
	got := replComplete("perf set ", nil)
	if len(got) != len(perfFields) {
		t.Errorf("perf set <space>: want %d fields, got %d", len(perfFields), len(got))
	}
	for i, f := range perfFields {
		if got[i] != "perf set "+f {
			t.Errorf("perf set <space>[%d]: want %q, got %q", i, "perf set "+f, got[i])
		}
	}
	// "perf set frus" should return just frustum_culling.
	got = replComplete("perf set frus", nil)
	if !reflect.DeepEqual(got, []string{"perf set frustum_culling"}) {
		t.Errorf("perf set frus: want [perf set frustum_culling], got %v", got)
	}
}

func TestReplComplete_NoMatch_Nil(t *testing.T) {
	cases := []string{"system xyz", "window xyz", "perf xyz", "getspeed blah blah"}
	for _, tc := range cases {
		got := replComplete(tc, nil)
		if len(got) != 0 {
			t.Errorf("replComplete(%q): want nil, got %v", tc, got)
		}
	}
}

func TestReplComplete_NavJumpBodyNames(t *testing.T) {
	bodies := []string{"Earth", "Mars", "Sun"}

	// "nav jump " → clear + all bodies
	got := replComplete("nav jump ", bodies)
	want := []string{"nav jump clear", "nav jump Earth", "nav jump Mars", "nav jump Sun"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nav jump <space>: got %v; want %v", got, want)
	}

	// "nav jump E" → prefix-filtered
	got = replComplete("nav jump E", bodies)
	if !reflect.DeepEqual(got, []string{"nav jump Earth"}) {
		t.Errorf("nav jump E: got %v; want [nav jump Earth]", got)
	}

	// "nav jump cl" → clear only
	got = replComplete("nav jump cl", bodies)
	if !reflect.DeepEqual(got, []string{"nav jump clear"}) {
		t.Errorf("nav jump cl: got %v; want [nav jump clear]", got)
	}

	// "camera track " → all bodies (no clear)
	got = replComplete("camera track ", bodies)
	want = []string{"camera track Earth", "camera track Mars", "camera track Sun"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("camera track <space>: got %v; want %v", got, want)
	}

	// "camera track M" → prefix-filtered
	got = replComplete("camera track M", bodies)
	if !reflect.DeepEqual(got, []string{"camera track Mars"}) {
		t.Errorf("camera track M: got %v; want [camera track Mars]", got)
	}

	// nil bodies → "clear" only (it's always valid)
	got = replComplete("nav jump ", nil)
	if !reflect.DeepEqual(got, []string{"nav jump clear"}) {
		t.Errorf("nav jump with nil bodies: want [nav jump clear], got %v", got)
	}
}
