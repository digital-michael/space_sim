package engine

import "testing"

// TestObjectCategoryValues locks in the exact integer values of the enum.
// If any value shifts, these tests will catch it before a refactor can
// silently corrupt saved state, serialized data, or switch-case logic.
func TestObjectCategoryValues(t *testing.T) {
	tests := []struct {
		name    string
		got     ObjectCategory
		wantInt int
	}{
		// Stellar group (0–9; 5–9 reserved)
		{"CategoryStarPreMain", CategoryStarPreMain, 0},
		{"CategoryStarMainSequence", CategoryStarMainSequence, 1},
		{"CategoryStarEvolved", CategoryStarEvolved, 2},
		{"CategorySubstellar", CategorySubstellar, 3},
		{"CategoryStellarRemnant", CategoryStellarRemnant, 4},
		// System body group (10–19; 17–19 reserved)
		{"CategoryPlanet", CategoryPlanet, 10},
		{"CategoryDwarfPlanet", CategoryDwarfPlanet, 11},
		{"CategoryMoon", CategoryMoon, 12},
		{"CategoryAsteroid", CategoryAsteroid, 13},
		{"CategoryBelt", CategoryBelt, 14},
		{"CategoryRogue", CategoryRogue, 15},
		{"CategoryArtifact", CategoryArtifact, 16},
		// Internal sentinel
		{"CategoryRing", CategoryRing, 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.got) != tt.wantInt {
				t.Errorf("%s = %d, want %d", tt.name, int(tt.got), tt.wantInt)
			}
		})
	}
}

// TestObjectCategoryCount ensures no new values were silently added or removed.
// Update this constant intentionally whenever the enum changes.
func TestObjectCategoryCount(t *testing.T) {
	const wantCount = 13
	all := []ObjectCategory{
		CategoryStarPreMain,
		CategoryStarMainSequence,
		CategoryStarEvolved,
		CategorySubstellar,
		CategoryStellarRemnant,
		CategoryPlanet,
		CategoryDwarfPlanet,
		CategoryMoon,
		CategoryAsteroid,
		CategoryBelt,
		CategoryRogue,
		CategoryArtifact,
		CategoryRing,
	}
	if len(all) != wantCount {
		t.Errorf("ObjectCategory enum has %d values, want %d", len(all), wantCount)
	}
}
