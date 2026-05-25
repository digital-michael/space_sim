package render

import (
	"testing"

	engine "github.com/digital-michael/space_sim/internal/sim/engine"
)

func selObj(name string, cat engine.ObjectCategory) *engine.Object {
	return &engine.Object{Meta: engine.ObjectMetadata{Name: name, Category: cat}}
}

// filterObjectsByCategory tests

func TestFilterByCategory_Empty(t *testing.T) {
	got := filterObjectsByCategory(nil, engine.CategoryPlanet)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestFilterByCategory_PlanetsOnly(t *testing.T) {
	objs := []*engine.Object{
		selObj("Earth", engine.CategoryPlanet),
		selObj("Sol", engine.CategoryStar),
		selObj("Mars", engine.CategoryPlanet),
	}
	got := filterObjectsByCategory(objs, engine.CategoryPlanet)
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Errorf("got %v, want [0 2]", got)
	}
}

func TestFilterByCategory_NoMatch(t *testing.T) {
	objs := []*engine.Object{selObj("Earth", engine.CategoryPlanet)}
	got := filterObjectsByCategory(objs, engine.CategoryStar)
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

func TestFilterByCategory_BeltAsteroidsOnly(t *testing.T) {
	objs := []*engine.Object{
		selObj("Earth", engine.CategoryPlanet),
		selObj("Asteroid-001", engine.CategoryAsteroid),
		selObj("Asteroid-002", engine.CategoryAsteroid),
	}
	got := filterObjectsByCategory(objs, engine.CategoryBelt)
	if len(got) != 1 || got[0] != -1 {
		t.Errorf("got %v, want [-1]", got)
	}
}

func TestFilterByCategory_BeltKuiperOnly(t *testing.T) {
	objs := []*engine.Object{selObj("KBO-001", engine.CategoryAsteroid)}
	got := filterObjectsByCategory(objs, engine.CategoryBelt)
	if len(got) != 1 || got[0] != -2 {
		t.Errorf("got %v, want [-2]", got)
	}
}

func TestFilterByCategory_BeltBoth(t *testing.T) {
	objs := []*engine.Object{
		selObj("Asteroid-001", engine.CategoryAsteroid),
		selObj("KBO-001", engine.CategoryAsteroid),
	}
	got := filterObjectsByCategory(objs, engine.CategoryBelt)
	if len(got) != 2 || got[0] != -1 || got[1] != -2 {
		t.Errorf("got %v, want [-1 -2]", got)
	}
}

// filterObjectsByCategoryAndText tests

func TestFilterByCatAndText_EmptyFilter(t *testing.T) {
	objs := []*engine.Object{
		selObj("Earth", engine.CategoryPlanet),
		selObj("Mars", engine.CategoryPlanet),
		selObj("Sol", engine.CategoryStar),
	}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryPlanet, "")
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Errorf("got %v, want [0 1]", got)
	}
}

func TestFilterByCatAndText_TextMatch(t *testing.T) {
	objs := []*engine.Object{
		selObj("Earth", engine.CategoryPlanet),
		selObj("Mars", engine.CategoryPlanet),
	}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryPlanet, "mars")
	if len(got) != 1 || got[0] != 1 {
		t.Errorf("got %v, want [1] (Mars)", got)
	}
}

func TestFilterByCatAndText_CaseInsensitive(t *testing.T) {
	objs := []*engine.Object{selObj("Earth", engine.CategoryPlanet)}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryPlanet, "EARTH")
	if len(got) != 1 || got[0] != 0 {
		t.Errorf("got %v, want [0]", got)
	}
}

func TestFilterByCatAndText_NoMatch(t *testing.T) {
	objs := []*engine.Object{selObj("Earth", engine.CategoryPlanet)}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryPlanet, "venus")
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}

func TestFilterByCatAndText_BeltEmptyFilter(t *testing.T) {
	objs := []*engine.Object{
		selObj("Asteroid-001", engine.CategoryAsteroid),
		selObj("KBO-001", engine.CategoryAsteroid),
	}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryBelt, "")
	if len(got) != 2 || got[0] != -1 || got[1] != -2 {
		t.Errorf("got %v, want [-1 -2]", got)
	}
}

func TestFilterByCatAndText_BeltAsteroidTextMatch(t *testing.T) {
	objs := []*engine.Object{
		selObj("Asteroid-001", engine.CategoryAsteroid),
		selObj("KBO-001", engine.CategoryAsteroid),
	}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryBelt, "asteroid")
	if len(got) != 1 || got[0] != -1 {
		t.Errorf("got %v, want [-1]", got)
	}
}

func TestFilterByCatAndText_BeltKuiperTextMatch(t *testing.T) {
	objs := []*engine.Object{
		selObj("Asteroid-001", engine.CategoryAsteroid),
		selObj("KBO-001", engine.CategoryAsteroid),
	}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryBelt, "kuiper")
	if len(got) != 1 || got[0] != -2 {
		t.Errorf("got %v, want [-2]", got)
	}
}

func TestFilterByCatAndText_BeltNoMatchText(t *testing.T) {
	objs := []*engine.Object{selObj("Asteroid-001", engine.CategoryAsteroid)}
	got := filterObjectsByCategoryAndText(objs, engine.CategoryBelt, "oort")
	if len(got) != 0 {
		t.Errorf("got %v, want []", got)
	}
}
