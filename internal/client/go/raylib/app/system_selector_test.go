package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSystemOptionsFromDirFiltersAndSorts(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"zeta_system.json":  `{"name":"Zeta System"}`,
		"alpha_system.JSON": `{"name":"Alpha System"}`,
		"notes.txt":         "{}",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(tempDir, "nested"), 0755); err != nil {
		t.Fatalf("Mkdir(nested) failed: %v", err)
	}

	options, err := discoverSystemOptionsFromDir(tempDir)
	if err != nil {
		t.Fatalf("discoverSystemOptionsFromDir() error = %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if options[0].Label != "alpha_system.JSON" {
		t.Fatalf("options[0].Label = %q, want alpha_system.JSON", options[0].Label)
	}
	if options[0].DisplayName != "Alpha System" {
		t.Fatalf("options[0].DisplayName = %q, want Alpha System", options[0].DisplayName)
	}
	if options[1].Label != "zeta_system.json" {
		t.Fatalf("options[1].Label = %q, want zeta_system.json", options[1].Label)
	}
	if options[1].DisplayName != "Zeta System" {
		t.Fatalf("options[1].DisplayName = %q, want Zeta System", options[1].DisplayName)
	}
	for _, option := range options {
		if !filepath.IsAbs(option.Path) {
			t.Fatalf("option.Path = %q, want absolute path", option.Path)
		}
	}
}

func TestDiscoverSystemOptionsPartitionsTestSystems(t *testing.T) {
	tempDir := t.TempDir()

	// Create two real systems and two test systems as directories.
	for _, dir := range []string{"zeta_system", "alpha_system", "nbody_test", "demo_test"} {
		if err := os.Mkdir(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Mkdir(%q) failed: %v", dir, err)
		}
		manifest := filepath.Join(tempDir, dir, "system.json")
		content := `{"name":"` + dir + `"}`
		if err := os.WriteFile(manifest, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile(%q) failed: %v", manifest, err)
		}
	}

	options, err := discoverSystemOptionsFromDir(tempDir)
	if err != nil {
		t.Fatalf("discoverSystemOptionsFromDir() error = %v", err)
	}
	// Expect: alpha_system, zeta_system, separator, demo_test, nbody_test
	if len(options) != 5 {
		t.Fatalf("len(options) = %d, want 5", len(options))
	}
	if options[0].Label != "alpha_system" {
		t.Errorf("options[0].Label = %q, want alpha_system", options[0].Label)
	}
	if options[1].Label != "zeta_system" {
		t.Errorf("options[1].Label = %q, want zeta_system", options[1].Label)
	}
	if !options[2].IsSeparator {
		t.Errorf("options[2].IsSeparator = false, want true")
	}
	if options[3].Label != "demo_test" {
		t.Errorf("options[3].Label = %q, want demo_test", options[3].Label)
	}
	if options[4].Label != "nbody_test" {
		t.Errorf("options[4].Label = %q, want nbody_test", options[4].Label)
	}
}

func TestDiscoverSystemOptionsNoSeparatorWhenNoTestSystems(t *testing.T) {
	tempDir := t.TempDir()

	for _, dir := range []string{"beta_system", "alpha_system"} {
		if err := os.Mkdir(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("Mkdir(%q) failed: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, dir, "system.json"), []byte(`{"name":"`+dir+`"}`), 0644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	options, err := discoverSystemOptionsFromDir(tempDir)
	if err != nil {
		t.Fatalf("discoverSystemOptionsFromDir() error = %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2 (no separator)", len(options))
	}
	for _, opt := range options {
		if opt.IsSeparator {
			t.Error("unexpected separator when no test systems exist")
		}
	}
}

func TestNewRuntimeSessionReturnsErrorForMissingSystem(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing_system.json")
	a := &App{cfg: Config{SystemConfig: missingPath}}

	_, err := a.newRuntimeSession(missingPath)
	if err == nil {
		t.Fatal("newRuntimeSession() error = nil, want error for missing system config")
	}
}
