package sim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

func TestLoadSystemFromFileCreatesRingSystemFeature(t *testing.T) {
	json := `{
	  "name": "Test System",
	  "version": "1.0",
	  "scale_factor": 1,
	  "bodies": [
	    {
	      "type": "star",
	      "name": "Sun",
	      "orbit": {"semi_major_axis": 0, "eccentricity": 0, "inclination": 0, "orbital_period": 0},
	      "physical": {"radius": 10, "mass": 1.0e30, "color": [255,255,255,255]},
	      "rendering": {"material": "emissive"},
	      "importance": 100
	    },
	    {
	      "type": "planet",
	      "name": "Saturn",
	      "parent": "Sun",
	      "orbit": {"semi_major_axis": 100, "eccentricity": 0, "inclination": 0, "orbital_period": 1000},
	      "physical": {"radius": 5, "mass": 1.0e26, "axial_tilt": 26.7, "color": [200,180,120,255]},
	      "rendering": {"material": "diffuse"},
	      "importance": 90
	    }
	  ],
	  "features": [
	    {
	      "type": "ring_system",
	      "name": "Saturn-Ring-A",
	      "parent": "Saturn",
	      "distribution": {"inner_radius": 4.9, "outer_radius": 5.6},
	      "physical": {"mass": 1.5e19, "axial_tilt": 26.7, "color": [240,220,175,220]},
	      "rendering": {"material": "diffuse"},
	      "importance": 30
	    }
	  ]
	}`

	path := filepath.Join(t.TempDir(), "system.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatalf("write temp system: %v", err)
	}

	state, err := LoadSystemFromFile(path)
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	ring := state.GetObject("Saturn-Ring-A")
	if ring == nil {
		t.Fatal("expected ring object to be created from ring_system feature")
	}
	if ring.Meta.Category != engine.CategoryRing {
		t.Fatalf("expected ring category, got %v", ring.Meta.Category)
	}
	if ring.Meta.ParentName != "Saturn" {
		t.Fatalf("expected parent Saturn, got %q", ring.Meta.ParentName)
	}
	if ring.Meta.InnerRadius != 4.9 || ring.Meta.PhysicalRadius != 5.6 {
		t.Fatalf("unexpected ring radii: inner=%v outer=%v", ring.Meta.InnerRadius, ring.Meta.PhysicalRadius)
	}
	if ring.Meta.AxialTilt != 26.7 {
		t.Fatalf("expected axial tilt 26.7, got %v", ring.Meta.AxialTilt)
	}
	if ring.Meta.Color.A != 220 {
		t.Fatalf("expected ring alpha 220, got %d", ring.Meta.Color.A)
	}
	if len(state.NavigationOrder) == 0 || state.NavigationOrder[len(state.NavigationOrder)-1] != engine.CategoryRing {
		found := false
		for _, cat := range state.NavigationOrder {
			if cat == engine.CategoryRing {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("expected navigation order to include ring category")
		}
	}
}

func TestLoadSystemFromFileRejectsInvalidRingSystemFeature(t *testing.T) {
	json := `{
	  "name": "Test System",
	  "version": "1.0",
	  "scale_factor": 1,
	  "bodies": [
	    {
	      "type": "star",
	      "name": "Sun",
	      "orbit": {"semi_major_axis": 0, "eccentricity": 0, "inclination": 0, "orbital_period": 0},
	      "physical": {"radius": 10, "mass": 1.0e30, "color": [255,255,255,255]},
	      "rendering": {"material": "emissive"},
	      "importance": 100
	    }
	  ],
	  "features": [
	    {
	      "type": "ring_system",
	      "name": "Broken-Ring",
	      "parent": "Sun",
	      "distribution": {"inner_radius": 5, "outer_radius": 4}
	    }
	  ]
	}`

	path := filepath.Join(t.TempDir(), "invalid-system.json")
	if err := os.WriteFile(path, []byte(json), 0644); err != nil {
		t.Fatalf("write temp system: %v", err)
	}

	if _, err := LoadSystemFromFile(path); err == nil {
		t.Fatal("expected invalid ring_system feature to fail loading")
	}
}
