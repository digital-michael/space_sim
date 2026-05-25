package render

import (
	"math"
	"testing"
)

func TestLodScaleEarthRef(t *testing.T) {
	if got := lodScale(0.5); got != 1.0 {
		t.Errorf("lodScale(0.5) = %f, want 1.0", got)
	}
}

func TestLodScaleSolCapped(t *testing.T) {
	if got := lodScale(27.25); got != 10.0 {
		t.Errorf("lodScale(27.25) = %f, want 10.0 (max cap)", got)
	}
}

func TestLodScaleJupiter(t *testing.T) {
	got := lodScale(2.8)
	want := float64(2.8 / 0.5)
	if math.Abs(got-want) > 1e-5 {
		t.Errorf("lodScale(2.8) = %f, want %f", got, want)
	}
}

func TestLodScaleAsteroidMinClamp(t *testing.T) {
	if got := lodScale(0.1); got != 1.0 {
		t.Errorf("lodScale(0.1) = %f, want 1.0 (min clamp)", got)
	}
}

func TestLodScaleZeroRadius(t *testing.T) {
	if got := lodScale(0.0); got != 1.0 {
		t.Errorf("lodScale(0) = %f, want 1.0", got)
	}
}

func TestLodScaleAtExactCap(t *testing.T) {
	// physRadius == 10 * lodScaleRef => scale = exactly 10.0
	if got := lodScale(5.0); got != 10.0 {
		t.Errorf("lodScale(5.0) = %f, want 10.0", got)
	}
}

func TestLodScaleJustBelowCap(t *testing.T) {
	got := lodScale(4.9)
	want := 4.9 / 0.5
	if math.Abs(got-want) > 1e-5 {
		t.Errorf("lodScale(4.9) = %f, want %f", got, want)
	}
}
