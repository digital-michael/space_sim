package render

import (
	"math"
	"testing"
)

// ── markerPhaseOffset ────────────────────────────────────────────────────────

func TestMarkerPhaseOffsetDeterministic(t *testing.T) {
	got1 := markerPhaseOffset("session-abc")
	got2 := markerPhaseOffset("session-abc")
	if got1 != got2 {
		t.Fatalf("phase offset not deterministic: %v != %v", got1, got2)
	}
}

func TestMarkerPhaseOffsetRange(t *testing.T) {
	ids := []string{"a", "abc-123", "session-xyz", ""}
	for _, id := range ids {
		offset := markerPhaseOffset(id)
		if offset < 0 || offset >= 2*math.Pi {
			t.Fatalf("phaseOffset(%q) = %v, want in [0, 2π)", id, offset)
		}
	}
}

func TestMarkerPhaseOffsetVaries(t *testing.T) {
	a := markerPhaseOffset("session-A")
	b := markerPhaseOffset("session-B")
	if a == b {
		t.Fatalf("different session IDs produced identical phase offset: %v", a)
	}
}

// ── markerBlinkAlpha ─────────────────────────────────────────────────────────

func TestMarkerBlinkAlphaRange(t *testing.T) {
	ids := []string{"alice", "bob", "carol", ""}
	times := []float64{0, 0.25, 0.5, 0.75, 1.0, 1.5, 3.0, 100.0}
	for _, id := range ids {
		for _, ti := range times {
			alpha := markerBlinkAlpha(id, ti)
			if alpha < 0 || alpha > 1 {
				t.Fatalf("blinkAlpha(%q, %v) = %v, want in [0, 1]", id, ti, alpha)
			}
		}
	}
}

func TestMarkerBlinkAlphaDifferentSessions(t *testing.T) {
	// Two sessions with the same time should produce different alphas because
	// their phase offsets differ (probabilistic — fails only on hash collision).
	t0 := 0.0
	a := markerBlinkAlpha("session-1", t0)
	b := markerBlinkAlpha("session-2", t0)
	if a == b {
		t.Fatalf("sessions produced identical blink alpha at t=0: %v", a)
	}
}

// ── markerSphereRadius ───────────────────────────────────────────────────────

func TestMarkerSphereRadiusNearCamera(t *testing.T) {
	// Within far threshold: should return markerNearRadius exactly.
	got := markerSphereRadius(0.5, 1080)
	if got != markerNearRadius {
		t.Fatalf("near radius: got %v, want %v", got, markerNearRadius)
	}
}

func TestMarkerSphereRadiusAtThreshold(t *testing.T) {
	// Exactly at threshold: should still return markerNearRadius (not > condition).
	got := markerSphereRadius(markerFarThresholdSU, 1080)
	if got != markerNearRadius {
		t.Fatalf("at-threshold radius: got %v, want %v", got, markerNearRadius)
	}
}

func TestMarkerSphereRadiusFarCamera(t *testing.T) {
	// Beyond far threshold: radius must exceed markerNearRadius.
	got := markerSphereRadius(50.0, 1080)
	if got <= markerNearRadius {
		t.Fatalf("far radius: got %v, want > %v (screen-space min should dominate)", got, markerNearRadius)
	}
}

func TestMarkerSphereRadiusScreenSpaceMinEnforced(t *testing.T) {
	// Verify the computed radius subtends at least markerScreenMinPx pixels
	// using the inverse of the derivation in markerSphereRadius.
	dist := 50.0
	screenH := 1080
	r := markerSphereRadius(dist, screenH)

	halfFovRad := 45.0 * math.Pi / 180.0 / 2.0
	screenPixels := r / dist * float64(screenH) / (2.0 * math.Tan(halfFovRad))
	if screenPixels < markerScreenMinPx-1e-9 {
		t.Fatalf("radius %v subtends %.4f px at dist %v, want >= %.1f",
			r, screenPixels, dist, markerScreenMinPx)
	}
}

func TestMarkerSphereRadiusZeroScreenHeight(t *testing.T) {
	// screenH=0 must not divide by zero; fallback to markerNearRadius.
	got := markerSphereRadius(50.0, 0)
	if got != markerNearRadius {
		t.Fatalf("zero screenH: got %v, want %v", got, markerNearRadius)
	}
}

// ── LOD / cull constants ─────────────────────────────────────────────────────

func TestMarkerCullDistanceSane(t *testing.T) {
	if markerCullDistanceSU <= 0 {
		t.Fatal("markerCullDistanceSU must be positive")
	}
	if markerCullDistanceSU > 200 {
		t.Fatalf("markerCullDistanceSU = %v unexpectedly large (spec max: 100)", markerCullDistanceSU)
	}
}

func TestMarkerOwnOpacityRange(t *testing.T) {
	if markerOwnOpacity <= 0 || markerOwnOpacity > 1 {
		t.Fatalf("markerOwnOpacity = %v, want in (0, 1]", markerOwnOpacity)
	}
}
