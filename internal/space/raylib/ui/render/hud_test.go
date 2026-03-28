package render

import (
	"strings"
	"testing"
)

func TestFormatSimulationDateText(t *testing.T) {
	const simSeconds = 0

	tests := []struct {
		name             string
		secondsPerSecond float32
		wantDateOnly     bool
	}{
		{name: "paused", secondsPerSecond: 0, wantDateOnly: false},
		{name: "real time", secondsPerSecond: 1, wantDateOnly: false},
		{name: "hour per second", secondsPerSecond: 3600, wantDateOnly: false},
		{name: "day per second", secondsPerSecond: 86400, wantDateOnly: true},
		{name: "week per second", secondsPerSecond: 604800, wantDateOnly: true},
		{name: "month per second", secondsPerSecond: 2628000, wantDateOnly: true},
		{name: "year per second", secondsPerSecond: 31557600, wantDateOnly: true},
	}

	for _, tt := range tests {
		got := formatSimulationDateText(simSeconds, tt.secondsPerSecond)
		if !strings.HasPrefix(got, "Date: 2000/01/01") {
			t.Fatalf("%s: unexpected date prefix %q", tt.name, got)
		}
		hasTime := strings.Contains(got, " 00:") || strings.Contains(got, " 01:") || strings.Contains(got, " 02:") || strings.Contains(got, " 03:") || strings.Contains(got, " 04:") || strings.Contains(got, " 05:") || strings.Contains(got, " 06:") || strings.Contains(got, " 07:") || strings.Contains(got, " 08:") || strings.Contains(got, " 09:") || strings.Contains(got, " 10:") || strings.Contains(got, " 11:") || strings.Contains(got, " 12:") || strings.Contains(got, " 13:") || strings.Contains(got, " 14:") || strings.Contains(got, " 15:") || strings.Contains(got, " 16:") || strings.Contains(got, " 17:") || strings.Contains(got, " 18:") || strings.Contains(got, " 19:") || strings.Contains(got, " 20:") || strings.Contains(got, " 21:") || strings.Contains(got, " 22:") || strings.Contains(got, " 23:")
		if tt.wantDateOnly && hasTime {
			t.Fatalf("%s: got %q, want date-only output", tt.name, got)
		}
		if !tt.wantDateOnly && !hasTime {
			t.Fatalf("%s: got %q, want date and time output", tt.name, got)
		}
	}
}
