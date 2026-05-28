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
	// CategoryRing is an internal sentinel (value 99) and is intentionally
	// excluded from NavigationOrder — ring systems are not shown as a UI tab.
	for _, cat := range state.NavigationOrder {
		if cat == engine.CategoryRing {
			t.Fatal("CategoryRing should not appear in NavigationOrder")
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

// systemJSON builds a minimal valid system JSON with two bodies: a star and one planet.
// The caller may pass extra star/planet JSON fragments to inject additional fields.
func systemJSON(starExtra, planetExtra string) string {
	return `{
	  "name": "Test System",
	  "version": "1.0",
	  "scale_factor": 1,
	  "bodies": [
	    {
	      "type": "star",
	      "name": "Sol",
	      "orbit": {"semi_major_axis": 0, "eccentricity": 0, "inclination": 0, "orbital_period": 0},
	      "physical": {"radius": 27.25, "mass": 1.989e30, "albedo": 1.0},
	      "rendering": {"material": "emissive", "texture_image": "data/assets/textures/sunmap.jpg", "fallback_color": [245,245,255,255]},
	      "luminosity": {"self_luminous": true, "solar_luminosity": 1.0, "surface_temperature_k": 5778, "emission_color": [255,244,220,255], "emission_intensity": 1.0},
	      "importance": 100` + starExtra + `
	    },
	    {
	      "type": "planet",
	      "name": "Earth",
	      "parent": "Sol",
	      "orbit": {"semi_major_axis": 100, "eccentricity": 0.017, "inclination": 0, "orbital_period": 365.25},
	      "physical": {"radius": 0.637, "mass": 5.972e24, "albedo": 0.367},
	      "rendering": {"material": "diffuse", "texture_image": "data/assets/textures/earthmap1k.jpg", "fallback_color": [100,149,237,255]},
	      "atmosphere": {"thickness_km": 100, "principal_chemicals": ["nitrogen","oxygen"], "cloud_coverage": 0.67, "color_hint": [120,180,255,160]},
	      "importance": 90` + planetExtra + `
	    }
	  ]
	}`
}

func writeSystem(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "system.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp system: %v", err)
	}
	return path
}

func TestLoaderPopulatesLuminosityFields(t *testing.T) {
	state, err := LoadSystemFromFile(writeSystem(t, systemJSON("", "")))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	sol := state.GetObject("Sol")
	if sol == nil {
		t.Fatal("Sol not found")
	}
	if !sol.Meta.SelfLuminous {
		t.Error("expected Sol.SelfLuminous = true")
	}
	if sol.Meta.SolarLuminosity != 1.0 {
		t.Errorf("expected SolarLuminosity=1.0, got %v", sol.Meta.SolarLuminosity)
	}
	if sol.Meta.SurfaceTemperatureK != 5778 {
		t.Errorf("expected SurfaceTemperatureK=5778, got %v", sol.Meta.SurfaceTemperatureK)
	}
	if sol.Meta.EmissionColor.R != 255 || sol.Meta.EmissionColor.A != 255 {
		t.Errorf("unexpected EmissionColor %+v", sol.Meta.EmissionColor)
	}
}

func TestLoaderPopulatesTexturePath(t *testing.T) {
	state, err := LoadSystemFromFile(writeSystem(t, systemJSON("", "")))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	sol := state.GetObject("Sol")
	if sol == nil {
		t.Fatal("Sol not found")
	}
	if sol.Meta.TexturePath != "data/assets/textures/sunmap.jpg" {
		t.Errorf("unexpected Sol TexturePath %q", sol.Meta.TexturePath)
	}

	earth := state.GetObject("Earth")
	if earth == nil {
		t.Fatal("Earth not found")
	}
	if earth.Meta.TexturePath != "data/assets/textures/earthmap1k.jpg" {
		t.Errorf("unexpected Earth TexturePath %q", earth.Meta.TexturePath)
	}
}

func TestLoaderPopulatesAtmosphereFields(t *testing.T) {
	state, err := LoadSystemFromFile(writeSystem(t, systemJSON("", "")))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	earth := state.GetObject("Earth")
	if earth == nil {
		t.Fatal("Earth not found")
	}
	if earth.Meta.AtmosphereThicknessKm != 100 {
		t.Errorf("expected AtmosphereThicknessKm=100, got %v", earth.Meta.AtmosphereThicknessKm)
	}
	if earth.Meta.CloudCoverage != 0.67 {
		t.Errorf("expected CloudCoverage=0.67, got %v", earth.Meta.CloudCoverage)
	}
	if earth.Meta.AtmosphereColorHint.A != 160 {
		t.Errorf("expected AtmosphereColorHint.A=160, got %v", earth.Meta.AtmosphereColorHint.A)
	}

	// Sol has no atmosphere — fields should be zero
	sol := state.GetObject("Sol")
	if sol.Meta.AtmosphereThicknessKm != 0 {
		t.Errorf("expected Sol AtmosphereThicknessKm=0, got %v", sol.Meta.AtmosphereThicknessKm)
	}
}

func TestLoaderPopulatesAlbedo(t *testing.T) {
	state, err := LoadSystemFromFile(writeSystem(t, systemJSON("", "")))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	earth := state.GetObject("Earth")
	if earth == nil {
		t.Fatal("Earth not found")
	}
	if earth.Meta.Albedo != 0.367 {
		t.Errorf("expected Albedo=0.367, got %v", earth.Meta.Albedo)
	}
}

func TestLoaderDiffuseThermalMaterial(t *testing.T) {
	json := systemJSON("", `,"rendering": {"material": "diffuse_thermal", "fallback_color": [240,120,40,255]}`)
	// The planet field in systemJSON already has rendering, so we need a fresh body instead.
	// Use a standalone system with one thermal-emitter planet.
	raw := `{
	  "name": "Thermal Test",
	  "version": "1.0",
	  "scale_factor": 1,
	  "bodies": [
	    {
	      "type": "planet",
	      "name": "HotJupiter",
	      "orbit": {"semi_major_axis": 50, "eccentricity": 0, "inclination": 0, "orbital_period": 10},
	      "physical": {"radius": 7.1, "mass": 1.9e27},
	      "rendering": {"material": "diffuse_thermal", "fallback_color": [240,120,40,255]},
	      "importance": 80
	    }
	  ]
	}`
	_ = json // unused helper call
	state, err := LoadSystemFromFile(writeSystem(t, raw))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}

	hj := state.GetObject("HotJupiter")
	if hj == nil {
		t.Fatal("HotJupiter not found")
	}
	if hj.Meta.Material != engine.MaterialDiffuseThermal {
		t.Errorf("expected MaterialDiffuseThermal, got %v", hj.Meta.Material)
	}
}

func TestLoaderFallbackColorPreferredOverPhysicalColor(t *testing.T) {
	// Body has both physical.color (old schema) and rendering.fallback_color (new schema).
	// fallback_color must win.
	raw := `{
	  "name": "Color Test",
	  "version": "1.0",
	  "scale_factor": 1,
	  "bodies": [
	    {
	      "type": "planet",
	      "name": "Tester",
	      "orbit": {"semi_major_axis": 100, "eccentricity": 0, "inclination": 0, "orbital_period": 365},
	      "physical": {"radius": 1, "mass": 1e24, "color": [10,20,30,255]},
	      "rendering": {"material": "diffuse", "fallback_color": [100,150,200,255]},
	      "importance": 50
	    }
	  ]
	}`
	state, err := LoadSystemFromFile(writeSystem(t, raw))
	if err != nil {
		t.Fatalf("load system: %v", err)
	}
	obj := state.GetObject("Tester")
	if obj == nil {
		t.Fatal("Tester not found")
	}
	if obj.Meta.Color.R != 100 || obj.Meta.Color.G != 150 || obj.Meta.Color.B != 200 {
		t.Errorf("expected fallback_color [100,150,200], got %+v", obj.Meta.Color)
	}
}
// ─── LoadSystemFromDir tests ────────────────────────────────────────────────

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoadSystemFromDir_Basic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{
		"name":"Test","system_version":"1.0","schema_version":"2.0","scale_factor":1,
		"files":{"stars":"stars.json","planets":"planets.json"}
	}`)
	writeFile(t, dir, "stars.json", `{
		"schema_version":"2.0","bodies":[{
			"type":"star","name":"Sol",
			"orbit":{"semi_major_axis":0,"eccentricity":0,"inclination":0,"orbital_period":0},
			"physical":{"mass":1.989e30,"radius":10,"color":[255,244,180,255]},
			"rendering":{"material":"emissive"},
			"luminosity":{"self_luminous":true,"solar_luminosity":1.0},
			"importance":100
		}]
	}`)
	writeFile(t, dir, "planets.json", `{
		"schema_version":"2.0","bodies":[{
			"type":"planet","name":"Earth","parent":"Sol",
			"orbit":{"semi_major_axis":1,"eccentricity":0,"inclination":0,"orbital_period":365},
			"physical":{"mass":5.972e24,"radius":0.04,"color":[100,149,237,255]},
			"rendering":{"material":"diffuse"},
			"importance":80
		}]
	}`)

	state, err := LoadSystemFromDir(dir)
	if err != nil {
		t.Fatalf("LoadSystemFromDir: %v", err)
	}
	if state.GetObject("Sol") == nil {
		t.Error("expected Sol to be loaded")
	}
	if state.GetObject("Earth") == nil {
		t.Error("expected Earth to be loaded")
	}
	if len(state.NavigationOrder) == 0 {
		t.Error("expected non-empty NavigationOrder")
	}
}

func TestLoadSystemFromDir_SchemaVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{
		"name":"Test","system_version":"1.0","schema_version":"9.9","scale_factor":1,
		"files":{}
	}`)

	_, err := LoadSystemFromDir(dir)
	if err == nil {
		t.Fatal("expected error for unsupported schema_version, got nil")
	}
}

func TestLoadSystemFromDir_MissingSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{"name":"Test","scale_factor":1,"files":{}}`)

	_, err := LoadSystemFromDir(dir)
	if err == nil {
		t.Fatal("expected error for missing schema_version, got nil")
	}
}

func TestLoadSystemFromDir_RoguesFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{
		"name":"Test","system_version":"1.0","schema_version":"2.0","scale_factor":1,
		"files":{"stars":"stars.json","rogues":"rogues.json"}
	}`)
	writeFile(t, dir, "stars.json", `{
		"schema_version":"2.0","bodies":[{
			"type":"star","name":"Sol",
			"orbit":{"semi_major_axis":0,"eccentricity":0,"inclination":0,"orbital_period":0},
			"physical":{"mass":1.989e30,"radius":10,"color":[255,255,255,255]},
			"rendering":{"material":"emissive"},"importance":100
		}]
	}`)
	writeFile(t, dir, "rogues.json", `{
		"schema_version":"2.0","rogues":[{
			"type":"rogue","name":"Comet X","subtype":"hyperbolic",
			"orbit":{"semi_major_axis":500,"eccentricity":0.99,"inclination":120,"orbital_period":99999},
			"physical":{"mass":1e12,"radius":0.005,"color":[200,220,255,200]},
			"rendering":{"material":"diffuse"},"importance":30
		}]
	}`)

	state, err := LoadSystemFromDir(dir)
	if err != nil {
		t.Fatalf("LoadSystemFromDir: %v", err)
	}
	comet := state.GetObject("Comet X")
	if comet == nil {
		t.Fatal("expected Comet X to be loaded")
	}
	if comet.Meta.Category != engine.CategoryRogue {
		t.Errorf("expected CategoryRogue (%d), got %d", engine.CategoryRogue, comet.Meta.Category)
	}
}

func TestLoadSystemFromDir_ArtifactsPositionOverride(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{
		"name":"Test","system_version":"1.0","schema_version":"2.0","scale_factor":1,
		"files":{"stars":"stars.json","artifacts":"artifacts.json"}
	}`)
	writeFile(t, dir, "stars.json", `{
		"schema_version":"2.0","bodies":[{
			"type":"star","name":"Sol",
			"orbit":{"semi_major_axis":0,"eccentricity":0,"inclination":0,"orbital_period":0},
			"physical":{"mass":1.989e30,"radius":10,"color":[255,255,255,255]},
			"rendering":{"material":"emissive"},"importance":100
		}]
	}`)
	writeFile(t, dir, "artifacts.json", `{
		"schema_version":"2.0","artifacts":[{
			"type":"artifact","name":"Voyager 1","subtype":"probe","faction":"nasa",
			"orbit":{
				"semi_major_axis":0,"eccentricity":0,"inclination":0,"orbital_period":0,
				"position_override":[100.0,5.0,200.0],
				"velocity_override":[0.1,0.0,0.05]
			},
			"physical":{"mass":721.9,"radius":0.0001,"color":[220,210,180,255]},
			"rendering":{"material":"diffuse"},"importance":80
		}]
	}`)

	state, err := LoadSystemFromDir(dir)
	if err != nil {
		t.Fatalf("LoadSystemFromDir: %v", err)
	}
	v1 := state.GetObject("Voyager 1")
	if v1 == nil {
		t.Fatal("expected Voyager 1 to be loaded")
	}
	if v1.Meta.Category != engine.CategoryArtifact {
		t.Errorf("expected CategoryArtifact (%d), got %d", engine.CategoryArtifact, v1.Meta.Category)
	}
	if v1.Anim.Position.X != 100.0 || v1.Anim.Position.Y != 5.0 || v1.Anim.Position.Z != 200.0 {
		t.Errorf("unexpected position: %+v", v1.Anim.Position)
	}
	if v1.Anim.Velocity.X != 0.1 || v1.Anim.Velocity.Z != 0.05 {
		t.Errorf("unexpected velocity: %+v", v1.Anim.Velocity)
	}
}

func TestLoadSystemFromDir_ParentResolutionError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "system.json", `{
		"name":"Test","system_version":"1.0","schema_version":"2.0","scale_factor":1,
		"files":{"planets":"planets.json"}
	}`)
	writeFile(t, dir, "planets.json", `{
		"schema_version":"2.0","bodies":[{
			"type":"planet","name":"Orphan","parent":"MissingParent",
			"orbit":{"semi_major_axis":1,"eccentricity":0,"inclination":0,"orbital_period":365},
			"physical":{"mass":5e24,"radius":0.04,"color":[100,149,237,255]},
			"rendering":{"material":"diffuse"},"importance":50
		}]
	}`)

	_, err := LoadSystemFromDir(dir)
	if err == nil {
		t.Fatal("expected error for unresolved parent reference, got nil")
	}
}
