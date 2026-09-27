package main

import (
	"math"
	"testing"

	"github.com/digital-michael/space_sim/internal/protocol"
)

// The curve's whole purpose is to cost almost nothing when nobody is near and to
// be smooth when someone is, so those two ends are what the tests pin down.
func TestPresenceRateCurve(t *testing.T) {
	tests := []struct {
		name  string
		dist  float64
		speed float64
		want  float64
	}{
		{"alone yields the floor", math.Inf(1), 0, presenceMinHz},
		{"touching yields full rate", 0, 0, presenceMaxHz},
		{"at the full-rate radius", presenceFullRateSU, 0, presenceMaxHz},
		{"inside the full-rate radius", 1, 0, presenceMaxHz},
		{"double the radius quarters the rate", 2 * presenceFullRateSU, 0, presenceMaxHz / 4},
		{"quadruple the radius is a sixteenth", 4 * presenceFullRateSU, 0, presenceMaxHz / 16},
		{"far away clamps to the floor", 10000, 0, presenceMinHz},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := presenceRateHz(tc.dist, tc.speed)
			if math.Abs(got-tc.want) > 0.01 {
				t.Errorf("presenceRateHz(%v, %v) = %.2f, want %.2f", tc.dist, tc.speed, got, tc.want)
			}
		})
	}
}

// Speed must pull the rate up before distance alone would, or two clients closing
// quickly are both throttled, both see stale positions, and both react late.
func TestPresenceRateAccountsForClosingSpeed(t *testing.T) {
	const dist = 40.0

	still := presenceRateHz(dist, 0)
	moving := presenceRateHz(dist, 20)

	if moving <= still {
		t.Errorf("a mover must publish more often: still=%.2f moving=%.2f", still, moving)
	}
	// Fast enough to cover the gap within the lookahead window means full rate,
	// regardless of current distance.
	if got := presenceRateHz(dist, dist); got != presenceMaxHz {
		t.Errorf("closing the whole gap within the lookahead should give full rate, got %.2f", got)
	}
}

func TestPresenceRateStaysInBounds(t *testing.T) {
	for _, d := range []float64{-5, 0, 0.001, 1, 5, 50, 1e6, math.Inf(1)} {
		for _, v := range []float64{0, 1, 1e6} {
			got := presenceRateHz(d, v)
			if got < presenceMinHz || got > presenceMaxHz {
				t.Errorf("presenceRateHz(%v, %v) = %v, outside [%v, %v]", d, v, got, presenceMinHz, presenceMaxHz)
			}
		}
	}
}

func TestNearestPeerDistanceIgnoresSelf(t *testing.T) {
	own := "me"
	sessions := []protocol.ClientSessionSnapshot{
		{SessionID: own, Position: [3]float64{0, 0, 0}},
		{SessionID: "other", Position: [3]float64{3, 4, 0}},
	}

	if got := nearestPeerDistance(sessions, own, [3]float64{0, 0, 0}); math.Abs(got-5) > 0.001 {
		t.Errorf("distance = %v, want 5 (self must be excluded)", got)
	}
}

func TestNearestPeerDistanceAloneIsInfinite(t *testing.T) {
	own := "me"
	sessions := []protocol.ClientSessionSnapshot{
		{SessionID: own, Position: [3]float64{10, 10, 10}},
	}

	got := nearestPeerDistance(sessions, own, [3]float64{10, 10, 10})
	if !math.IsInf(got, 1) {
		t.Errorf("distance = %v, want +Inf when no peers exist", got)
	}
	// And that must translate into the cheapest possible publish rate.
	if rate := presenceRateHz(got, 0); rate != presenceMinHz {
		t.Errorf("a client with no peers should publish at the floor, got %.2f Hz", rate)
	}
}

func TestNearestPeerDistancePicksClosest(t *testing.T) {
	sessions := []protocol.ClientSessionSnapshot{
		{SessionID: "far", Position: [3]float64{100, 0, 0}},
		{SessionID: "near", Position: [3]float64{7, 0, 0}},
		{SessionID: "mid", Position: [3]float64{30, 0, 0}},
	}

	if got := nearestPeerDistance(sessions, "me", [3]float64{0, 0, 0}); math.Abs(got-7) > 0.001 {
		t.Errorf("distance = %v, want 7", got)
	}
}
