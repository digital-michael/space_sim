package persist_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/digital-michael/space_sim/internal/persist"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// buildTestState creates a minimal SimulationState with two objects.
func buildTestState() *engine.SimulationState {
	state := engine.NewSimulationState()
	state.SecondsPerSecond = 3600.0
	state.NumWorkers = 2
	state.NavigationOrder = []engine.ObjectCategory{engine.CategoryStar, engine.CategoryPlanet}
	state.AllocatedDatasets[engine.AsteroidDatasetSmall] = true
	state.CurrentDataset = engine.AsteroidDatasetSmall

	star := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           "Sol",
			Category:       engine.CategoryStar,
			Mass:           1.989e30,
			PhysicalRadius: 696_000,
			Color:          engine.Color{R: 255, G: 255, B: 0, A: 255},
			Material:       engine.MaterialEmissive,
			Importance:     100,
		},
		Visible: true,
		Dataset: engine.AsteroidDatasetSmall,
	}

	planet := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           "Earth",
			Category:       engine.CategoryPlanet,
			Mass:           5.972e24,
			PhysicalRadius: 6_371,
			Color:          engine.Color{R: 0, G: 100, B: 255, A: 255},
			Material:       engine.MaterialDiffuse,
			Importance:     80,
			SemiMajorAxis:  1.0,
			OrbitalPeriod:  31_557_600,
			ParentName:     "Sol",
		},
		Visible: true,
		Dataset: engine.AsteroidDatasetSmall,
	}

	state.AddObject(star)
	state.AddObject(planet)
	return state
}

func TestDefinitionsRoundTrip(t *testing.T) {
	original := buildTestState()

	dir := t.TempDir()
	path := filepath.Join(dir, "defs.json")

	if err := persist.SaveDefinitions(path, original); err != nil {
		t.Fatalf("SaveDefinitions: %v", err)
	}

	loaded, err := persist.LoadDefinitions(path)
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}

	if got, want := len(loaded.Objects), len(original.Objects); got != want {
		t.Fatalf("object count: got %d, want %d", got, want)
	}

	for i, orig := range original.Objects {
		got := loaded.Objects[i]
		if got.Meta.Name != orig.Meta.Name {
			t.Errorf("Objects[%d].Meta.Name: got %q, want %q", i, got.Meta.Name, orig.Meta.Name)
		}
		if got.Meta.Category != orig.Meta.Category {
			t.Errorf("Objects[%d].Meta.Category: got %v, want %v", i, got.Meta.Category, orig.Meta.Category)
		}
		if got.Meta.Mass != orig.Meta.Mass {
			t.Errorf("Objects[%d].Meta.Mass: got %g, want %g", i, got.Meta.Mass, orig.Meta.Mass)
		}
		if got.Meta.PhysicalRadius != orig.Meta.PhysicalRadius {
			t.Errorf("Objects[%d].Meta.PhysicalRadius: got %g, want %g", i, got.Meta.PhysicalRadius, orig.Meta.PhysicalRadius)
		}
		if got.Meta.Color != orig.Meta.Color {
			t.Errorf("Objects[%d].Meta.Color: got %v, want %v", i, got.Meta.Color, orig.Meta.Color)
		}
		if got.Meta.OrbitalPeriod != orig.Meta.OrbitalPeriod {
			t.Errorf("Objects[%d].Meta.OrbitalPeriod: got %g, want %g", i, got.Meta.OrbitalPeriod, orig.Meta.OrbitalPeriod)
		}
		if got.Meta.ParentName != orig.Meta.ParentName {
			t.Errorf("Objects[%d].Meta.ParentName: got %q, want %q", i, got.Meta.ParentName, orig.Meta.ParentName)
		}
		if got.Visible != orig.Visible {
			t.Errorf("Objects[%d].Visible: got %v, want %v", i, got.Visible, orig.Visible)
		}
		if got.Dataset != orig.Dataset {
			t.Errorf("Objects[%d].Dataset: got %v, want %v", i, got.Dataset, orig.Dataset)
		}
		// AnimationState must be zero on load (recalculated by physics).
		if got.Anim != (engine.AnimationState{}) {
			t.Errorf("Objects[%d].Anim should be zero after LoadDefinitions", i)
		}
	}

	if loaded.CurrentDataset != original.CurrentDataset {
		t.Errorf("CurrentDataset: got %v, want %v", loaded.CurrentDataset, original.CurrentDataset)
	}
	if loaded.SecondsPerSecond != original.SecondsPerSecond {
		t.Errorf("SecondsPerSecond: got %v, want %v", loaded.SecondsPerSecond, original.SecondsPerSecond)
	}
	if loaded.NumWorkers != original.NumWorkers {
		t.Errorf("NumWorkers: got %v, want %v", loaded.NumWorkers, original.NumWorkers)
	}
	if len(loaded.NavigationOrder) != len(original.NavigationOrder) {
		t.Errorf("NavigationOrder length: got %d, want %d", len(loaded.NavigationOrder), len(original.NavigationOrder))
	}
	if _, ok := loaded.AllocatedDatasets[engine.AsteroidDatasetSmall]; !ok {
		t.Error("AllocatedDatasets[Small] missing after round-trip")
	}
}

func TestLoadDefinitions_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")

	if err := persist.SaveDefinitions(path, buildTestState()); err != nil {
		t.Fatal(err)
	}
	// Overwrite with garbage after a successful save.
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := persist.LoadDefinitions(path)
	if err == nil {
		t.Error("expected error loading corrupt file, got nil")
	}
}

func TestSaveDefinitions_NoTempSiblings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defs.json")

	if err := persist.SaveDefinitions(path, buildTestState()); err != nil {
		t.Fatalf("SaveDefinitions: %v", err)
	}

	// No .tmp siblings should remain after a successful write.
	matches, _ := filepath.Glob(filepath.Join(dir, ".persist-*.tmp"))
	if len(matches) != 0 {
		t.Errorf("temp file(s) not cleaned up: %v", matches)
	}
}
