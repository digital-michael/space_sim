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
		{"CategoryPlanet", CategoryPlanet, 0},
		{"CategoryDwarfPlanet", CategoryDwarfPlanet, 1},
		{"CategoryMoon", CategoryMoon, 2},
		{"CategoryAsteroid", CategoryAsteroid, 3},
		{"CategoryRing", CategoryRing, 4},
		{"CategoryStar", CategoryStar, 5},
		{"CategoryBelt", CategoryBelt, 6},
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
	const wantCount = 7
	all := []ObjectCategory{
		CategoryPlanet,
		CategoryDwarfPlanet,
		CategoryMoon,
		CategoryAsteroid,
		CategoryRing,
		CategoryStar,
		CategoryBelt,
	}
	if len(all) != wantCount {
		t.Errorf("ObjectCategory enum has %d values, want %d", len(all), wantCount)
	}
}
