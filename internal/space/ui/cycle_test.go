package ui

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/space/engine"
)

// testCategoryOrder mirrors the SOL cycle order used in production.
var testCategoryOrder = []engine.ObjectCategory{
	engine.CategoryStar,
	engine.CategoryPlanet,
	engine.CategoryDwarfPlanet,
	engine.CategoryMoon,
	engine.CategoryBelt,
}

// TestCycleCategoryForward verifies the full forward rotation:
// Star -> Planet -> DwarfPlanet -> Moon -> Belt -> Star
func TestCycleCategoryForward(t *testing.T) {
	steps := []struct {
		start engine.ObjectCategory
		want  engine.ObjectCategory
	}{
		{engine.CategoryStar, engine.CategoryPlanet},
		{engine.CategoryPlanet, engine.CategoryDwarfPlanet},
		{engine.CategoryDwarfPlanet, engine.CategoryMoon},
		{engine.CategoryMoon, engine.CategoryBelt},
		{engine.CategoryBelt, engine.CategoryStar},
	}
	for _, tt := range steps {
		inp := NewInputState(engine.CategoryStar)
		inp.SelectedCategory = tt.start
		inp.CycleCategory(testCategoryOrder)
		if inp.SelectedCategory != tt.want {
			t.Errorf("CycleCategory from %v: got %v, want %v", tt.start, inp.SelectedCategory, tt.want)
		}
	}
}

// TestCycleCategoryBack verifies the full backward rotation:
// Star -> Belt -> Moon -> DwarfPlanet -> Planet -> Star
func TestCycleCategoryBack(t *testing.T) {
	steps := []struct {
		start engine.ObjectCategory
		want  engine.ObjectCategory
	}{
		{engine.CategoryStar, engine.CategoryBelt},
		{engine.CategoryBelt, engine.CategoryMoon},
		{engine.CategoryMoon, engine.CategoryDwarfPlanet},
		{engine.CategoryDwarfPlanet, engine.CategoryPlanet},
		{engine.CategoryPlanet, engine.CategoryStar},
	}
	for _, tt := range steps {
		inp := NewInputState(engine.CategoryStar)
		inp.SelectedCategory = tt.start
		inp.CycleCategoryBack(testCategoryOrder)
		if inp.SelectedCategory != tt.want {
			t.Errorf("CycleCategoryBack from %v: got %v, want %v", tt.start, inp.SelectedCategory, tt.want)
		}
	}
}

// TestCycleCategoryResetsIndex confirms that cycling always resets SelectedIndex to 0.
func TestCycleCategoryResetsIndex(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	inp.SelectedIndex = 42
	inp.CycleCategory(testCategoryOrder)
	if inp.SelectedIndex != 0 {
		t.Errorf("CycleCategory did not reset SelectedIndex: got %d, want 0", inp.SelectedIndex)
	}

	inp.SelectedIndex = 42
	inp.CycleCategoryBack(testCategoryOrder)
	if inp.SelectedIndex != 0 {
		t.Errorf("CycleCategoryBack did not reset SelectedIndex: got %d, want 0", inp.SelectedIndex)
	}
}

// TestCycleCategoryUnknownFallback verifies that an unrecognised category falls back to CategoryStar.
func TestCycleCategoryUnknownFallback(t *testing.T) {
	for _, unknown := range []engine.ObjectCategory{engine.CategoryAsteroid, engine.CategoryRing} {
		fwd := NewInputState(engine.CategoryStar)
		fwd.SelectedCategory = unknown
		fwd.CycleCategory(testCategoryOrder)
		if fwd.SelectedCategory != engine.CategoryStar {
			t.Errorf("CycleCategory fallback from %v: got %v, want CategoryStar", unknown, fwd.SelectedCategory)
		}

		bck := NewInputState(engine.CategoryStar)
		bck.SelectedCategory = unknown
		bck.CycleCategoryBack(testCategoryOrder)
		if bck.SelectedCategory != engine.CategoryStar {
			t.Errorf("CycleCategoryBack fallback from %v: got %v, want CategoryStar", unknown, bck.SelectedCategory)
		}
	}
}

// TestCycleCategoryFullRound verifies that five forward steps return to the start.
func TestCycleCategoryFullRound(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	const steps = 5
	for range steps {
		inp.CycleCategory(testCategoryOrder)
	}
	if inp.SelectedCategory != engine.CategoryStar {
		t.Errorf("after %d CycleCategory calls, got %v, want CategoryStar", steps, inp.SelectedCategory)
	}
}

// TestCycleCategoryFullRoundBack verifies that five backward steps return to the start.
func TestCycleCategoryFullRoundBack(t *testing.T) {
	inp := NewInputState(engine.CategoryStar)
	const steps = 5
	for range steps {
		inp.CycleCategoryBack(testCategoryOrder)
	}
	if inp.SelectedCategory != engine.CategoryStar {
		t.Errorf("after %d CycleCategoryBack calls, got %v, want CategoryStar", steps, inp.SelectedCategory)
	}
}
