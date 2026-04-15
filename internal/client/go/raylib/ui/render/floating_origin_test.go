package render

// Tests for DEF-001: Floating-Point Precision Collapse at Extreme Camera Distances.
//
// The fix: all 3-D object positions passed to Raylib Draw* calls are expressed in
// camera-relative coordinates (worldPos - cameraPos). The Raylib camera is always
// placed at the origin {0,0,0}. This keeps GPU vertex inputs numerically small
// regardless of how large the absolute world coordinates grow.

import (
	"math"
	"testing"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

// cameraRelativePos mirrors the expression used by drawObject / drawObjectsInstanced
// in renders.go. It is replicated here as a pure function so the arithmetic can be
// covered without a Raylib context.
func cameraRelativePos(world, cam engine.Vector3) engine.Vector3 {
	return engine.Vector3{
		X: world.X - cam.X,
		Y: world.Y - cam.Y,
		Z: world.Z - cam.Z,
	}
}

// TestFloatingOriginCollocated checks that an object placed exactly at the
// camera position always produces a render-space coordinate of {0,0,0},
// at every scale used in the simulation (from near-origin Sun to far-field
// interstellar distances).
func TestFloatingOriginCollocated(t *testing.T) {
	scales := []engine.Vector3{
		{X: 0, Y: 0, Z: 0},         // solar origin
		{X: 100, Y: 0, Z: 0},       // ~1 AU (Earth)
		{X: 960, Y: 0, Z: 0},       // ~9.6 AU (Saturn)
		{X: 3000, Y: 0, Z: 0},      // outer solar system
		{X: 100000, Y: 0, Z: 0},    // extreme zoom-out
		{X: 27_000_000, Y: 0, Z: 0}, // interstellar: ~alpha Centauri
	}

	for _, pos := range scales {
		rel := cameraRelativePos(pos, pos)
		if rel.X != 0 || rel.Y != 0 || rel.Z != 0 {
			t.Errorf("collocated at %v: want {0,0,0}, got {%v,%v,%v}",
				pos, rel.X, rel.Y, rel.Z)
		}
	}
}

// TestFloatingOriginOffsetPreserved checks that the camera-relative offset
// equals the true world-space separation, within float32 precision.
func TestFloatingOriginOffsetPreserved(t *testing.T) {
	cases := []struct {
		name      string
		cam       engine.Vector3
		world     engine.Vector3
		wantDelta engine.Vector3
	}{
		{
			name:      "earth from sun",
			cam:       engine.Vector3{X: 0, Y: 0, Z: 0},
			world:     engine.Vector3{X: 100, Y: 0, Z: 0},
			wantDelta: engine.Vector3{X: 100, Y: 0, Z: 0},
		},
		{
			name:      "moon from earth orbit",
			cam:       engine.Vector3{X: 100, Y: 0, Z: 0},
			world:     engine.Vector3{X: 100, Y: 0.257, Z: 0}, // ~2.57 sim-units away
			wantDelta: engine.Vector3{X: 0, Y: 0.257, Z: 0},
		},
		{
			name:      "near object at far camera distance",
			cam:       engine.Vector3{X: 3000, Y: 0, Z: 0},
			world:     engine.Vector3{X: 3001.5, Y: 0, Z: 0},
			wantDelta: engine.Vector3{X: 1.5, Y: 0, Z: 0},
		},
	}

	const eps = 1e-4 // float32 tolerance

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cameraRelativePos(tc.world, tc.cam)
			if math.Abs(float64(got.X-tc.wantDelta.X)) > eps ||
				math.Abs(float64(got.Y-tc.wantDelta.Y)) > eps ||
				math.Abs(float64(got.Z-tc.wantDelta.Z)) > eps {
				t.Errorf("want %v, got %v", tc.wantDelta, got)
			}
		})
	}
}

// TestFloatingOriginZeroCamera verifies that a zero camera position is
// a transparent pass-through: render coords equal world coords.
func TestFloatingOriginZeroCamera(t *testing.T) {
	world := engine.Vector3{X: 152, Y: 10, Z: 5}
	rel := cameraRelativePos(world, engine.Vector3{})
	if rel.X != world.X || rel.Y != world.Y || rel.Z != world.Z {
		t.Errorf("zero-cam pass-through: want %v, got %v", world, rel)
	}
}
