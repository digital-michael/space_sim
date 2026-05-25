package render

import (
	"testing"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// applyRingLightFactor tests

func TestApplyRingLightFactor_FullBrightness(t *testing.T) {
	c := rl.Color{R: 200, G: 150, B: 100, A: 255}
	got := applyRingLightFactor(c, 1.0)
	if got != c {
		t.Errorf("factor=1.0: got %+v, want %+v", got, c)
	}
}

func TestApplyRingLightFactor_AboveOne(t *testing.T) {
	c := rl.Color{R: 200, G: 150, B: 100, A: 255}
	got := applyRingLightFactor(c, 1.5)
	if got != c {
		t.Errorf("factor=1.5: got %+v, want %+v (no change expected)", got, c)
	}
}

func TestApplyRingLightFactor_Half(t *testing.T) {
	c := rl.Color{R: 200, G: 100, B: 50, A: 128}
	got := applyRingLightFactor(c, 0.5)
	want := rl.Color{R: 100, G: 50, B: 25, A: 128}
	if got != want {
		t.Errorf("factor=0.5: got %+v, want %+v", got, want)
	}
}

func TestApplyRingLightFactor_Zero(t *testing.T) {
	c := rl.Color{R: 200, G: 150, B: 100, A: 255}
	got := applyRingLightFactor(c, 0.0)
	if got.R != 0 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Errorf("factor=0: got %+v, want R=0 G=0 B=0 A=255", got)
	}
}

func TestApplyRingLightFactor_Negative(t *testing.T) {
	c := rl.Color{R: 200, G: 150, B: 100, A: 255}
	got := applyRingLightFactor(c, -1.0)
	if got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("factor=-1: got %+v, expected zero RGB", got)
	}
}

// spotlightFactor tests

func TestSpotlightFactor_ObjectAtCameraPos(t *testing.T) {
	pos := engine.Vector3{X: 1, Y: 1, Z: 1}
	fwd := engine.Vector3{X: 0, Y: 0, Z: 1}
	got := spotlightFactor(pos, pos, fwd) // zero dist
	if got != 1.0 {
		t.Errorf("zero-dist: got %f, want 1.0", got)
	}
}

func TestSpotlightFactor_DirectlyInFront(t *testing.T) {
	// Object at (0,0,5), camera at (0,0,0), looking forward (0,0,1)
	// cosAngle = 1.0, inside inner cone → returns 1
	got := spotlightFactor(
		engine.Vector3{X: 0, Y: 0, Z: 5},
		engine.Vector3{X: 0, Y: 0, Z: 0},
		engine.Vector3{X: 0, Y: 0, Z: 1},
	)
	if got != 1.0 {
		t.Errorf("direct front: got %f, want 1.0", got)
	}
}

func TestSpotlightFactor_ExactlyOnOuterEdge(t *testing.T) {
	// Build a vector at cos(40°) = 0.766 along camera forward (0,0,1).
	// cosAngle ≈ infraSpotOuterCos → t ≤ 0 → very close to 0.
	sin40 := float32(0.6428)
	cos40 := float32(0.7660)
	got := spotlightFactor(
		engine.Vector3{X: sin40, Y: 0, Z: cos40},
		engine.Vector3{X: 0, Y: 0, Z: 0},
		engine.Vector3{X: 0, Y: 0, Z: 1},
	)
	if got > 0.001 {
		t.Errorf("outer edge: got %f, want near 0", got)
	}
}

func TestSpotlightFactor_BeyondOuterEdge(t *testing.T) {
	// Object fully behind / to the side → cosAngle << outer → t < 0 → returns 0
	got := spotlightFactor(
		engine.Vector3{X: 10, Y: 0, Z: 0}, // 90° off-axis
		engine.Vector3{X: 0, Y: 0, Z: 0},
		engine.Vector3{X: 0, Y: 0, Z: 1},
	)
	if got != 0.0 {
		t.Errorf("beyond outer: got %f, want 0.0", got)
	}
}
