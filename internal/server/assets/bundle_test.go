package assets

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// writeTree materializes a data directory from a map of relative path to
// contents and returns its root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

// entryNames returns the sorted entry paths inside a built bundle.
func entryNames(t *testing.T, b *Bundle) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b.Data), b.Size())
	if err != nil {
		t.Fatalf("open bundle as zip: %v", err)
	}
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func sampleTree() map[string]string {
	return map[string]string{
		"systems/solar_system.json":      `{"schema_version":1}`,
		"bodies/earth.json":              `{"name":"Earth"}`,
		"ships/scout.json":               `{"name":"Scout"}`,
		"assets/textures/earthmap1k.jpg": "\xff\xd8\xff binary-ish",
		"profiles/laptop.json":           `{"profile":"laptop"}`,
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	root := writeTree(t, sampleTree())

	first, err := Build(root)
	if err != nil {
		t.Fatalf("first Build: %v", err)
	}
	second, err := Build(root)
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}

	if first.Hash != second.Hash {
		t.Errorf("hash not reproducible across builds of identical content:\n first=%s\nsecond=%s", first.Hash, second.Hash)
	}
	if !bytes.Equal(first.Data, second.Data) {
		t.Error("archive bytes differ across builds of identical content")
	}
}

func TestBuildHashChangesWhenIncludedContentChanges(t *testing.T) {
	tree := sampleTree()
	root := writeTree(t, tree)

	before, err := Build(root)
	if err != nil {
		t.Fatalf("Build before: %v", err)
	}

	target := filepath.Join(root, "systems", "solar_system.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":2}`), 0o644); err != nil {
		t.Fatalf("rewrite %s: %v", target, err)
	}

	after, err := Build(root)
	if err != nil {
		t.Fatalf("Build after: %v", err)
	}
	if before.Hash == after.Hash {
		t.Error("hash unchanged after an included file was modified; cache would serve stale content")
	}
}

// Shipped keybinding profiles MUST be bundled. input.LoadKeyMap treats the
// profile file as mandatory, so a client without one cannot start at all — which
// is exactly how an isolated remote client failed before this was corrected.
// User customization lives in the separate keybindings config file, which is
// never bundled. See spec D5.
func TestBuildIncludesProfiles(t *testing.T) {
	root := writeTree(t, sampleTree())

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var found bool
	for _, name := range entryNames(t, b) {
		if filepath.ToSlash(name) == "data/profiles/laptop.json" {
			found = true
		}
	}
	if !found {
		t.Error("shipped keybinding profiles must be in the bundle; a remote client cannot start without one")
	}
}

func TestBuildProfilesChangeAffectsHash(t *testing.T) {
	root := writeTree(t, sampleTree())

	before, err := Build(root)
	if err != nil {
		t.Fatalf("Build before: %v", err)
	}

	target := filepath.Join(root, "profiles", "laptop.json")
	if err := os.WriteFile(target, []byte(`{"profile":"changed"}`), 0o644); err != nil {
		t.Fatalf("rewrite %s: %v", target, err)
	}

	after, err := Build(root)
	if err != nil {
		t.Fatalf("Build after: %v", err)
	}
	if before.Hash == after.Hash {
		t.Error("profiles are bundled content, so changing one must change the hash")
	}
}

func TestBuildEntryPathsAreDataPrefixed(t *testing.T) {
	root := writeTree(t, sampleTree())

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []string{
		"data/assets/textures/earthmap1k.jpg",
		"data/bodies/earth.json",
		"data/profiles/laptop.json",
		"data/ships/scout.json",
		"data/systems/solar_system.json",
	}
	got := entryNames(t, b)

	if len(got) != len(want) {
		t.Fatalf("entry count = %d, want %d\ngot: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if b.EntryCount != len(want) {
		t.Errorf("EntryCount = %d, want %d", b.EntryCount, len(want))
	}
}

func TestBuildContentRoundTrips(t *testing.T) {
	tree := sampleTree()
	root := writeTree(t, tree)

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(b.Data), b.Size())
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}

	for _, f := range zr.File {
		rel := f.Name[len("data/"):]
		want, ok := tree[rel]
		if !ok {
			t.Errorf("unexpected entry %q", f.Name)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %s: %v", f.Name, err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read entry %s: %v", f.Name, err)
		}
		rc.Close()
		if buf.String() != want {
			t.Errorf("entry %s content = %q, want %q", f.Name, buf.String(), want)
		}
	}
}

func TestBuildSkipsMissingSubdirs(t *testing.T) {
	// Only systems/ exists — a minimal deployment with no ships or assets.
	root := writeTree(t, map[string]string{
		"systems/solar_system.json": `{"schema_version":1}`,
	})

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build with missing subdirs must not error: %v", err)
	}
	if b.EntryCount != 1 {
		t.Errorf("EntryCount = %d, want 1", b.EntryCount)
	}
}

func TestBuildEmptyDataDirProducesValidBundle(t *testing.T) {
	root := t.TempDir()

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build on empty dir: %v", err)
	}
	if b.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0", b.EntryCount)
	}
	if b.Hash == "" {
		t.Error("empty bundle must still carry a hash")
	}
	if _, err := zip.NewReader(bytes.NewReader(b.Data), b.Size()); err != nil {
		t.Errorf("empty bundle is not a readable archive: %v", err)
	}
}

// A bundle must not carry links that could resolve outside the client's cache
// root once unpacked, so symlinks are skipped at build time rather than stored.
func TestBuildSkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}
	root := writeTree(t, map[string]string{
		"systems/solar_system.json": `{"schema_version":1}`,
		"outside.txt":               "should never be reachable",
	})

	link := filepath.Join(root, "systems", "escape.json")
	if err := os.Symlink(filepath.Join(root, "outside.txt"), link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, name := range entryNames(t, b) {
		if name == "data/systems/escape.json" {
			t.Error("symlink was stored in the bundle; it must be skipped")
		}
	}
	if b.EntryCount != 1 {
		t.Errorf("EntryCount = %d, want 1 (symlink excluded)", b.EntryCount)
	}
}

// TestBuildAgainstRealDataDir exercises the builder against this repository's
// actual data tree rather than a synthetic fixture. Skipped when the tree is
// absent so the package still tests cleanly outside a checkout.
func TestBuildAgainstRealDataDir(t *testing.T) {
	root := filepath.Join("..", "..", "..", "data")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("real data dir unavailable: %v", err)
	}

	b, err := Build(root)
	if err != nil {
		t.Fatalf("Build against real data dir: %v", err)
	}
	if b.EntryCount == 0 {
		t.Fatal("real data dir produced an empty bundle")
	}

	var sawSystem, sawTexture, sawProfile bool
	for _, name := range entryNames(t, b) {
		switch {
		case name == "data/profiles/laptop.json":
			sawProfile = true
		// F-034 stores each system as a directory of per-category files,
		// not a single JSON document.
		case name == "data/systems/solar_system/system.json":
			sawSystem = true
		case name == "data/assets/textures/starfield_8k.jpg":
			sawTexture = true
		}
	}
	if !sawProfile {
		t.Error("bundle is missing data/profiles/laptop.json — a remote client cannot start without a keybinding profile")
	}
	if !sawSystem {
		t.Error("bundle is missing data/systems/solar_system/system.json")
	}
	if !sawTexture {
		t.Error("bundle is missing the skysphere texture")
	}

	t.Logf("real bundle: %d entries, %.2f MB, hash %s", b.EntryCount, float64(b.Size())/(1024*1024), b.Hash[:16])
}
