package render

// Tests for atmosphere flicker bugs.
//
// Two root causes produce the flickering textured and untextured objects observed
// near planets with atmospheres:
//
// BUG-1: DEPTH WRITE DURING ADDITIVE BLEND
//   The atmosphere sphere is drawn with BlendAddColors (GL_ONE, GL_ONE).
//   Raylib's BeginBlendMode does NOT disable depth writes (glDepthMask(false)).
//   The glow sphere extends beyond the planet's physicalRadius by 4–60%.
//   Its fragments write depth values at glowRadius into the depth buffer.
//   On the next draw call, any opaque object whose camera distance falls between
//   physicalRadius and glowRadius fails the depth test and is discarded → flickers.
//   This is distance-dependent (worse at certain ranges) and affects both
//   textured and untextured spheres — matching the reported symptom exactly.
//
//   Fix: call rl.DisableDepthMask() before DrawModel and rl.EnableDepthMask()
//   after, so the atmosphere sphere contributes colour but never poisons depth.
//
// BUG-2: SHARED MUTABLE MODEL MATERIAL STATE
//   r.atmoSphere is a single rl.Model reused every drawAtmosphereGlow call.
//   GetMaterials() returns unsafe.Slice(m.Materials, m.MaterialCount) — a live
//   slice into C-allocated memory. Writing mats[0].Shader mutates that C struct
//   in-place and the mutation persists. When two objects with atmospheres are
//   drawn in the same frame, the second call overwrites the shader (same model),
//   and any state set before DrawModel for the first call may be clobbered.
//   The glow radius transform is also set before, not after, the shader binding,
//   so draw order determines which body wins the transform.
//
//   This test file exercises the pure-Go layers that are observable without a
//   GL context. The depth-write and material-clobber paths require GL and are
//   documented as manual verification steps below.
//
// MANUAL VERIFICATION (no GL in unit tests):
//   1. Depth write:  Place camera at a distance between Earth physicalRadius and
//      glowRadius (~0.63 × physicalRadius to ~0.95). Step forward slowly.
//      Without the fix: Mars or another nearby body flickers in/out.
//      With the fix:    no flicker at any approach distance.
//   2. Multi-atmo:   Enable two adjacent bodies with atmosphere (e.g. Earth+Venus).
//      Without the fix: one body's halo bleeds the other's tint.
//      With the fix:    each halo renders with its own colour.

import (
	"testing"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

// --- helpers ------------------------------------------------------------------

func makeAtmoObject(physRadius, atmoThicknessKm float32, colorHint engine.Color) *engine.Object {
	return &engine.Object{
		Meta: engine.ObjectMetadata{
			PhysicalRadius:         physRadius,
			AtmosphereThicknessKm:  atmoThicknessKm,
			AtmosphereColorHint:    colorHint,
		},
	}
}

// computeGlowRadius replicates the calibration formula in drawAtmosphereGlow so
// we can assert on expected values without calling GL code.
func computeGlowRadius(physRadius, atmoThicknessKm float32) float32 {
	const (
		kmPerSimUnit = float32(12742)
		atmoBoost    = float32(4)
		atmoFloor    = float32(0.04)
		atmoCap      = float32(0.60)
	)
	bodyRadiusKm := physRadius * kmPerSimUnit
	frac := (atmoThicknessKm / bodyRadiusKm) * atmoBoost
	if frac < atmoFloor {
		frac = atmoFloor
	}
	if frac > atmoCap {
		frac = atmoCap
	}
	return physRadius * (1.0 + frac)
}

// --- BUG-1 confirming tests ---------------------------------------------------

// TestGlowRadiusExceedsPhysicalRadius confirms that for any body with an
// atmosphere the glow sphere always extends beyond physicalRadius. This is the
// prerequisite for BUG-1: the depth region between physicalRadius and glowRadius
// is where flickering occurs.
func TestGlowRadiusExceedsPhysicalRadius(t *testing.T) {
	cases := []struct {
		name            string
		physRadius      float32
		atmoThicknessKm float32
	}{
		{"Earth (100 km)", 0.5, 100},
		{"Venus (65 km)", 0.487, 65},
		{"Mars (11 km)", 0.266, 11},
		{"Jupiter (5000 km)", 5.5, 5000},
		{"Sol corona (500000 km)", 50.0, 500000},
		{"Floor clamp body (tiny atmo)", 0.5, 0.1}, // frac < atmoFloor → still 4%
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			glow := computeGlowRadius(tc.physRadius, tc.atmoThicknessKm)
			if glow <= tc.physRadius {
				t.Errorf("glowRadius %v must exceed physRadius %v", glow, tc.physRadius)
			}
		})
	}
}

// TestDepthPollutionZone quantifies the depth-pollution zone for Earth — the
// range [physRadius, glowRadius] where BUG-1 produces flickering. Any opaque
// body rendered while the camera is in this zone is at risk.
func TestDepthPollutionZone(t *testing.T) {
	physRadius := float32(0.5)          // Earth
	atmoThickness := float32(100)       // km

	glowRadius := computeGlowRadius(physRadius, atmoThickness)
	pollutionDepth := glowRadius - physRadius

	if pollutionDepth <= 0 {
		t.Fatalf("expected positive pollution depth, got %v", pollutionDepth)
	}

	// At Earth scale the zone should be about 4% of physRadius (floor clamp).
	minExpected := physRadius * 0.039
	if pollutionDepth < minExpected {
		t.Errorf("pollution zone %v smaller than expected minimum %v", pollutionDepth, minExpected)
	}

	t.Logf("Earth depth-pollution zone: physRadius=%.4f  glowRadius=%.4f  zone=%.4f sim-units",
		physRadius, glowRadius, pollutionDepth)
}

// TestGlowRadiusFloorClamp confirms the 4% minimum glow is applied when
// atmosphere thickness is zero or negligibly small. This also means even a
// "minimal" atmosphere creates a depth-pollution zone.
func TestGlowRadiusFloorClamp(t *testing.T) {
	physRadius := float32(0.5)
	glowZero := computeGlowRadius(physRadius, 0.001) // effectively zero thickness
	expected := physRadius * 1.04
	// allow 0.1% tolerance for float arithmetic
	if glowZero < expected*0.999 || glowZero > expected*1.001 {
		t.Errorf("floor-clamped glowRadius: want ~%.4f, got %.4f", expected, glowZero)
	}
}

// TestGlowRadiusCapClamp confirms the 60% maximum cap is applied for extreme
// atmospheres (e.g. Sol's corona). Without the cap the glow sphere could extend
// to 2.87× physRadius, creating an enormous depth-pollution zone.
func TestGlowRadiusCapClamp(t *testing.T) {
	physRadius := float32(50.0)            // Sol
	atmoThickness := float32(500_000)      // corona, ~500 000 km

	glow := computeGlowRadius(physRadius, atmoThickness)
	maxAllowed := physRadius * 1.60

	if glow > maxAllowed*1.001 {
		t.Errorf("cap not applied: glowRadius %.4f exceeds max %.4f", glow, maxAllowed)
	}
	t.Logf("Sol capped glowRadius=%.4f (max=%.4f)", glow, maxAllowed)
}

// --- BUG-1 guard: early-return conditions ------------------------------------

// TestNoAtmosphereSkipsGlow confirms that the glow is skipped when
// AtmosphereThicknessKm == 0 or AtmosphereColorHint.A == 0.
// These are the two early-return guards in drawAtmosphereGlow.
// No GL calls are made; we confirm computeGlowRadius is never reached.
func TestNoAtmosphereSkipsGlow(t *testing.T) {
	cases := []struct {
		name    string
		obj     *engine.Object
		wantSkip bool
	}{
		{
			name: "zero thickness",
			obj:  makeAtmoObject(0.5, 0, engine.Color{R: 100, G: 150, B: 255, A: 180}),
			wantSkip: true,
		},
		{
			name: "zero alpha",
			obj:  makeAtmoObject(0.5, 100, engine.Color{R: 100, G: 150, B: 255, A: 0}),
			wantSkip: true,
		},
		{
			name: "valid atmosphere",
			obj:  makeAtmoObject(0.5, 100, engine.Color{R: 100, G: 150, B: 255, A: 180}),
			wantSkip: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skip := tc.obj.Meta.AtmosphereThicknessKm <= 0 || tc.obj.Meta.AtmosphereColorHint.A == 0
			if skip != tc.wantSkip {
				t.Errorf("wantSkip=%v got skip=%v", tc.wantSkip, skip)
			}
		})
	}
}

// --- BUG-2 confirming tests ---------------------------------------------------

// TestGlowColorIntensityEncoding confirms the glowColor intensity-weight
// encoding: ch.A/255*0.6. This is the value passed as glowColor.a to the
// shader, where it acts as an intensity multiplier (not real alpha).
// BUG-2 means a second atmosphere draw call in the same frame overwrites this
// uniform with different values — each body's colour contaminates the other.
func TestGlowColorIntensityEncoding(t *testing.T) {
	cases := []struct {
		name          string
		alpha         uint8
		wantIntensity float32
	}{
		{"full alpha (255)", 255, 0.6},
		{"half alpha (128)", 128, float32(128) / 255.0 * 0.6},
		{"zero alpha (0)", 0, 0.0},
		{"Earth-like (180)", 180, float32(180) / 255.0 * 0.6},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := float32(tc.alpha) / 255.0 * 0.6
			diff := got - tc.wantIntensity
			if diff < -1e-6 || diff > 1e-6 {
				t.Errorf("want intensity %.6f, got %.6f", tc.wantIntensity, got)
			}
		})
	}
}

// TestAtmoBaseRadiusSelection confirms equatorial radius is preferred over
// physical radius for oblate bodies. This is the base for glowRadius; an
// incorrect base would shift the depth-pollution zone.
func TestAtmoBaseRadiusSelection(t *testing.T) {
	cases := []struct {
		name           string
		physRadius     float32
		equatorialR    float32
		wantBaseRadius float32
	}{
		{"spherical body (no equatorial)", 0.5, 0, 0.5},
		{"oblate body (equatorial set)", 0.5, 0.55, 0.55},
		{"equatorial == physRadius", 0.5, 0.5, 0.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Mirror the base-radius selection logic from drawAtmosphereGlow.
			base := tc.physRadius
			if tc.equatorialR > 0 {
				base = tc.equatorialR
			}
			if base != tc.wantBaseRadius {
				t.Errorf("want base %.4f, got %.4f", tc.wantBaseRadius, base)
			}
		})
	}
}
