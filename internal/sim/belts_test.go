package sim

import (
	"math"
	"math/rand"
	"testing"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

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
