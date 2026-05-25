package render

import (
	"fmt"
	"testing"
)

func TestImportanceThresholdLabelKnownValues(t *testing.T) {
	cases := []struct {
		v    int
		want string
	}{
		{0, "0 (All objects)"},
		{5, "5 (Hide tiny asteroids)"},
		{10, "10 (Hide medium asteroids)"},
		{15, "15 (Hide all asteroids)"},
		{30, "30 (Hide rings)"},
		{40, "40 (Hide small moons)"},
		{50, "50 (Hide dwarf planets)"},
		{60, "60 (Hide large moons)"},
		{70, "70 (Hide ice giants)"},
		{80, "80 (Hide rocky planets)"},
		{90, "90 (Hide gas giants)"},
	}
	for _, c := range cases {
		got := importanceThresholdLabel(c.v)
		if got != c.want {
			t.Errorf("importanceThresholdLabel(%d) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestImportanceThresholdLabelDefault(t *testing.T) {
	for _, v := range []int{1, 7, 25, 45, 100, 999} {
		got := importanceThresholdLabel(v)
		want := fmt.Sprintf("%d", v)
		if got != want {
			t.Errorf("importanceThresholdLabel(%d) = %q, want %q", v, got, want)
		}
	}
}
