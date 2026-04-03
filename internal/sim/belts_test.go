package sim

import (
	"math"
	"math/rand"
	"testing"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// twoTypeConfig is shared by the determinism tests. It has two object types so
// that map-iteration order matters — a single-type config would not catch the bug.
func twoTypeConfig() BeltConfig {
	return BeltConfig{
		Name:              "Test Belt",
		NamePrefix:        "T-",
		InnerRadius:       200,
		OuterRadius:       400,
		Thickness:         10,
		DistanceToAURatio: 100,
		UseKeplerLaw:      true,
		EccentricityMin:   0,
		EccentricityMax:   0.1,
		InclinationMin:    0,
		InclinationMax:    0.1,
		ObjectTypes: map[string]BeltObjectTypeConfig{
			"rocky": {
				Count:   3,
				SizeMin: 1,
				SizeMax: 2,
			},
			"carbonaceous": {
				Count:   3,
				SizeMin: 3,
				SizeMax: 5,
			},
		},
	}
}

// TestCreateBelt_Deterministic verifies that two CreateBelt calls with the same
// seed and config produce identical object sequences regardless of Go map
// iteration randomisation. This is the regression test for the front/back
// double-buffer flicker bug caused by non-deterministic type key ordering.
func TestCreateBelt_Deterministic(t *testing.T) {
	cfg := twoTypeConfig()

	stateA := engine.NewSimulationState()
	CreateBelt(stateA, cfg, engine.AsteroidDatasetSmall, rand.New(rand.NewSource(42)))

	stateB := engine.NewSimulationState()
	CreateBelt(stateB, cfg, engine.AsteroidDatasetSmall, rand.New(rand.NewSource(42)))

	if len(stateA.Objects) != len(stateB.Objects) {
		t.Fatalf("object count mismatch: %d vs %d", len(stateA.Objects), len(stateB.Objects))
	}
	for i, a := range stateA.Objects {
		b := stateB.Objects[i]
		if a.Meta.Name != b.Meta.Name {
			t.Errorf("index %d: name %q != %q", i, a.Meta.Name, b.Meta.Name)
		}
		if a.Meta.PhysicalRadius != b.Meta.PhysicalRadius {
			t.Errorf("index %d: PhysicalRadius %f != %f", i, a.Meta.PhysicalRadius, b.Meta.PhysicalRadius)
		}
		if a.Anim.Position != b.Anim.Position {
			t.Errorf("index %d: Position %v != %v", i, a.Anim.Position, b.Anim.Position)
		}
	}
}

func TestCreateBelt_OrbitalPeriodUsesSeconds(t *testing.T) {
	state := engine.NewSimulationState()
	rng := rand.New(rand.NewSource(42))

	config := BeltConfig{
		Name:              "Asteroid Belt",
		NamePrefix:        "Asteroid-",
		InnerRadius:       280,
		OuterRadius:       280,
		Thickness:         0,
		DistanceToAURatio: 100,
		UseKeplerLaw:      true,
		EccentricityMin:   0,
		EccentricityMax:   0,
		InclinationMin:    0,
		InclinationMax:    0,
		ObjectTypes: map[string]BeltObjectTypeConfig{
			"rock": {
				Count:      1,
				SizeMin:    1,
				SizeMax:    1,
				Importance: 10,
			},
		},
	}

	CreateBelt(state, config, engine.AsteroidDatasetSmall, rng)

	if len(state.Objects) != 1 {
		t.Fatalf("expected 1 belt object, got %d", len(state.Objects))
	}

	obj := state.Objects[0]
	distanceAU := float64(obj.Meta.SemiMajorAxis) / float64(config.DistanceToAURatio)
	expectedSeconds := math.Pow(distanceAU, 1.5) * 365.256 * 86400.0

	if obj.Meta.OrbitalPeriod < 86400 {
		t.Fatalf("orbital period appears to be in days, got %.2f", obj.Meta.OrbitalPeriod)
	}

	tolerance := expectedSeconds * 0.001
	delta := math.Abs(float64(obj.Meta.OrbitalPeriod) - expectedSeconds)
	if delta > tolerance {
		t.Fatalf("orbital period mismatch: got %.2f sec, want %.2f sec (delta %.2f > tol %.2f)", obj.Meta.OrbitalPeriod, expectedSeconds, delta, tolerance)
	}
}
