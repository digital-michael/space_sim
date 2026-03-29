package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSystemOptionsFromDirFiltersAndSorts(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{"zeta_system.json", "alpha_system.JSON", "notes.txt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("{}"), 0644); err != nil {
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
	if options[1].Label != "zeta_system.json" {
		t.Fatalf("options[1].Label = %q, want zeta_system.json", options[1].Label)
	}
	for _, option := range options {
		if !filepath.IsAbs(option.Path) {
			t.Fatalf("option.Path = %q, want absolute path", option.Path)
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
