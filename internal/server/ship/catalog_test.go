package ship

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// writeShipJSON writes content to dir/<name>.json and returns the path.
func writeShipJSON(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeShipJSON: %v", err)
	}
	return path
}

// validScoutJSON returns minimal valid JSON for a ship named "test_ship".
func validScoutJSON(id string) string {
	return `{
  "version": 1,
  "id": "` + id + `",
  "name": "Test Ship",
  "description": "A test ship.",
  "class": "light",
  "model": { "path": "data/assets/models/test.iqm", "scale": 1.0 },
  "identification": { "transponder_prefix": "TS" },
  "engine_stages": [
    { "stage": 1, "label": "Maneuvering", "accel_min_ms2": 0.0, "accel_max_ms2": 1e5, "power_draw_w": 5e8 },
    { "stage": 2, "label": "Main Drive", "accel_min_ms2": 1e5, "accel_max_ms2": 5e7, "power_draw_w": 4e9 }
  ],
  "turning": { "rate_deg_per_s": 90.0, "power_draw_w": 1e8 },
  "power": { "available_w": 5e9, "system_draw_baseline_w": 2e8, "overload_policy": "degrade_non_critical" },
  "mass_kg": 12000.0,
  "max_speed_sim_units_per_s": 5.0,
  "superluminal_allowed": true
}`
}

// minimalDef returns a fully populated ShipDefinition for use in instance tests.
func minimalDef() *ShipDefinition {
	return &ShipDefinition{
		Version: 1,
		ID:      "scout_mk1",
		Name:    "Scout Mk I",
		Class:   ClassLight,
		Model:   ModelRef{Path: "data/assets/models/scout_mk1.iqm", Scale: 1.0},
		Identification: IdentificationSpec{TransponderPrefix: "SC"},
		EngineStages: []EngineStage{
			{Stage: 1, Label: "Maneuvering", AccelMinMS2: 0, AccelMaxMS2: 1e5, PowerDrawW: 5e8},
			{Stage: 2, Label: "Main Drive", AccelMinMS2: 1e5, AccelMaxMS2: 5e7, PowerDrawW: 4e9},
		},
		Turning: TurningSpec{RateDegreesPerS: 90.0, PowerDrawW: 1e8},
		Power: PowerSpec{
			AvailableW:          5e9,
			SystemDrawBaselineW: 2e8,
			OverloadPolicy:      OverloadDegrade,
		},
		MassKg:               12000,
		MaxSpeedSimUnitsPerS: 5.0,
		SuperluminalAllowed:  true,
	}
}

// ─── LoadCatalog ──────────────────────────────────────────────────────────────

func TestLoadCatalog_LoadsValidFile(t *testing.T) {
	dir := t.TempDir()
	writeShipJSON(t, dir, "test_ship", validScoutJSON("test_ship"))

	cat, err := LoadCatalog(dir, "test_ship")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 1 {
		t.Errorf("Len = %d, want 1", cat.Len())
	}
	def := cat.Get("test_ship")
	if def == nil {
		t.Fatal("Get(test_ship) = nil")
	}
	if def.Name != "Test Ship" {
		t.Errorf("Name = %q, want Test Ship", def.Name)
	}
}

func TestLoadCatalog_SkipsInvalidFile(t *testing.T) {
	dir := t.TempDir()
	writeShipJSON(t, dir, "good_ship", validScoutJSON("good_ship"))
	// bad file: version mismatch
	writeShipJSON(t, dir, "bad_ship", `{"version":2,"id":"bad_ship","name":"Bad"}`)

	cat, err := LoadCatalog(dir, "good_ship")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 1 {
		t.Errorf("Len = %d, want 1 (bad file should be skipped)", cat.Len())
	}
	if cat.Get("bad_ship") != nil {
		t.Error("bad_ship should not be in catalog")
	}
}

func TestLoadCatalog_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir, "scout_mk1")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 0 {
		t.Errorf("Len = %d, want 0", cat.Len())
	}
	if cat.Default() != nil {
		t.Error("Default() should be nil for empty catalog")
	}
}

func TestLoadCatalog_DefaultFallback(t *testing.T) {
	dir := t.TempDir()
	writeShipJSON(t, dir, "ship_a", validScoutJSON("ship_a"))

	// Request a defaultID that doesn't exist — should fall back to ship_a.
	cat, err := LoadCatalog(dir, "nonexistent")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Default() == nil {
		t.Error("Default() should not be nil when catalog has ships")
	}
}

func TestLoadCatalog_DefaultIDHonoured(t *testing.T) {
	dir := t.TempDir()
	writeShipJSON(t, dir, "ship_a", validScoutJSON("ship_a"))
	writeShipJSON(t, dir, "ship_b", validScoutJSON("ship_b"))

	cat, err := LoadCatalog(dir, "ship_b")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.DefaultID() != "ship_b" {
		t.Errorf("DefaultID = %q, want ship_b", cat.DefaultID())
	}
	if cat.Default().ID != "ship_b" {
		t.Errorf("Default().ID = %q, want ship_b", cat.Default().ID)
	}
}

func TestLoadCatalog_All(t *testing.T) {
	dir := t.TempDir()
	writeShipJSON(t, dir, "ship_a", validScoutJSON("ship_a"))
	writeShipJSON(t, dir, "ship_b", validScoutJSON("ship_b"))

	cat, err := LoadCatalog(dir, "ship_a")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if len(cat.All()) != 2 {
		t.Errorf("All() len = %d, want 2", len(cat.All()))
	}
}

// ─── validateDefinition ───────────────────────────────────────────────────────

func TestValidation_MismatchedID(t *testing.T) {
	dir := t.TempDir()
	// File is named "wrong.json" but id field says "other"
	writeShipJSON(t, dir, "wrong", validScoutJSON("other"))
	cat, err := LoadCatalog(dir, "")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 0 {
		t.Error("mismatched id/filename should be rejected")
	}
}

func TestValidation_NonContiguousStages(t *testing.T) {
	dir := t.TempDir()
	json := `{
  "version": 1, "id": "bad", "name": "Bad", "class": "light",
  "model": {"path": "x.iqm", "scale": 1.0},
  "identification": {"transponder_prefix": "BD"},
  "engine_stages": [
    {"stage": 1, "label": "A", "accel_min_ms2": 0, "accel_max_ms2": 1e5, "power_draw_w": 1e8},
    {"stage": 3, "label": "B", "accel_min_ms2": 1e5, "accel_max_ms2": 5e7, "power_draw_w": 1e9}
  ],
  "turning": {"rate_deg_per_s": 45.0, "power_draw_w": 1e7},
  "power": {"available_w": 1e9, "system_draw_baseline_w": 1e8, "overload_policy": "hard_cut"},
  "mass_kg": 5000, "max_speed_sim_units_per_s": 3.0, "superluminal_allowed": false
}`
	writeShipJSON(t, dir, "bad", json)
	cat, err := LoadCatalog(dir, "")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 0 {
		t.Error("non-contiguous stages should be rejected")
	}
}

func TestValidation_UnknownClass(t *testing.T) {
	dir := t.TempDir()
	json := `{
  "version": 1, "id": "cls", "name": "C", "class": "destroyer",
  "model": {"path": "x.iqm", "scale": 1.0},
  "identification": {"transponder_prefix": "CL"},
  "engine_stages": [{"stage": 1, "label": "A", "accel_min_ms2": 0, "accel_max_ms2": 1e5, "power_draw_w": 1e8}],
  "turning": {"rate_deg_per_s": 45.0, "power_draw_w": 1e7},
  "power": {"available_w": 1e9, "system_draw_baseline_w": 1e8, "overload_policy": "degrade_non_critical"},
  "mass_kg": 5000, "max_speed_sim_units_per_s": 3.0, "superluminal_allowed": false
}`
	writeShipJSON(t, dir, "cls", json)
	cat, err := LoadCatalog(dir, "")
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}
	if cat.Len() != 0 {
		t.Error("unknown class should be rejected")
	}
}

// ─── loadBundledShips integration ────────────────────────────────────────────

func TestBundledShipsLoad(t *testing.T) {
	dir := bundledShipDir(t)
	cat, err := LoadCatalog(dir, "scout_mk1")
	if err != nil {
		t.Fatalf("LoadCatalog(bundled): %v", err)
	}
	if cat.Len() < 3 {
		t.Errorf("expected at least 3 bundled ships, got %d", cat.Len())
	}
	for _, id := range []string{"scout_mk1", "freighter_t1", "explorer_x1"} {
		if cat.Get(id) == nil {
			t.Errorf("bundled ship %q missing from catalog", id)
		}
	}
	if cat.Default() == nil || cat.Default().ID != "scout_mk1" {
		t.Errorf("default ship should be scout_mk1, got %v", cat.Default())
	}
}

// bundledShipDir walks up from the test file to find data/ships/.
func bundledShipDir(t *testing.T) string {
	t.Helper()
	dir := "."
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "data", "ships")
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
		dir = filepath.Join(dir, "..")
	}
	t.Skip("data/ships not found — skipping bundled-ship integration test")
	return ""
}
