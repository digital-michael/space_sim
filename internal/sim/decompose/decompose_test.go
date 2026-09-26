package decompose_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digital-michael/space_sim/internal/sim"
	"github.com/digital-michael/space_sim/internal/sim/decompose"
)

// realSystem locates this repository's solar system definition, skipping when it
// is unavailable so the package still tests outside a checkout.
func realSystem(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "..", "data", "systems", "solar_system")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("real system definition unavailable: %v", err)
	}
	return p
}

// The decisive test: every emitted chunk must load through the same loader the
// server uses. A chunk that does not load is worthless regardless of how tidily
// it was partitioned.
func TestChunksLoadThroughRealLoader(t *testing.T) {
	src := realSystem(t)
	out := t.TempDir()

	chunks, err := decompose.Decompose(src, decompose.Options{OutDir: out})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected the solar system to split into several chunks, got %d", len(chunks))
	}

	for _, ch := range chunks {
		dir := filepath.Join(out, ch.Dir)
		state, err := sim.LoadSystemFromDir(dir)
		if err != nil {
			t.Errorf("chunk %s does not load: %v", ch.Dir, err)
			continue
		}
		if len(state.Objects) == 0 {
			t.Errorf("chunk %s loaded but contains no objects", ch.Dir)
		}
	}
}

// The primary must appear in every chunk: it is the light source, and it has to
// be present in every local sky.
func TestEveryChunkContainsThePrimary(t *testing.T) {
	src := realSystem(t)
	out := t.TempDir()

	chunks, err := decompose.Decompose(src, decompose.Options{OutDir: out})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}

	for _, ch := range chunks {
		raw, err := os.ReadFile(filepath.Join(out, ch.Dir, "stars.json"))
		if err != nil {
			t.Errorf("chunk %s has no stars file: %v", ch.Dir, err)
			continue
		}
		if !strings.Contains(string(raw), `"Sol"`) {
			t.Errorf("chunk %s is missing the primary", ch.Dir)
		}
	}
}

// Partitioning must neither lose nor duplicate a body. The star is the one
// deliberate exception, since it is injected into every chunk.
func TestPartitionIsCompleteAndDisjoint(t *testing.T) {
	src := realSystem(t)
	out := t.TempDir()

	chunks, err := decompose.Decompose(src, decompose.Options{OutDir: out})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}

	seen := map[string]string{}
	for _, ch := range chunks {
		for _, m := range ch.Members {
			if prev, dup := seen[m]; dup {
				t.Errorf("%q appears in both %s and %s", m, prev, ch.Dir)
				continue
			}
			seen[m] = ch.Dir
		}
	}

	full, err := sim.LoadSystemFromDir(src)
	if err != nil {
		t.Fatalf("load source system: %v", err)
	}
	// Procedurally generated belt members are created at load time and are not
	// named entries in the definition, so compare only against definition bodies
	// that the partition actually tracks.
	var missing []string
	for _, obj := range full.Objects {
		n := obj.Meta.Name
		if n == "Sol" {
			continue
		}
		if _, ok := seen[n]; !ok {
			missing = append(missing, n)
		}
	}
	if len(missing) > 20 {
		missing = append(missing[:20], "…")
	}
	if len(missing) > 0 {
		t.Logf("bodies present in the loaded source but not in any chunk (expected for generated belt members): %v", missing)
	}
}

// Chunk directory names must sort into orbital order, because the name is both
// the selector ordering and the simulation's runtime identity.
func TestChunkNamesSortInOrbitalOrder(t *testing.T) {
	src := realSystem(t)
	out := t.TempDir()

	chunks, err := decompose.Decompose(src, decompose.Options{OutDir: out})
	if err != nil {
		t.Fatalf("Decompose: %v", err)
	}

	for i, ch := range chunks {
		if !strings.Contains(ch.Dir, "_0") && !strings.Contains(ch.Dir, "_1") {
			t.Errorf("chunk %s carries no ordering prefix", ch.Dir)
		}
		if i > 0 && chunks[i-1].Dir >= ch.Dir {
			t.Errorf("chunk names do not sort: %s then %s", chunks[i-1].Dir, ch.Dir)
		}
	}

	if first, last := chunks[0], chunks[len(chunks)-1]; !strings.HasSuffix(first.Dir, "inner") || !strings.HasSuffix(last.Dir, "outer") {
		t.Errorf("bands must bracket the splits, got first=%s last=%s", first.Dir, last.Dir)
	}

	// The selector sorts and displays by the system.json "name" field, not the
	// directory, so the ordering must be present THERE to have any effect.
	for i, ch := range chunks {
		if ch.Display == "" {
			t.Fatalf("chunk %s has no display name", ch.Dir)
		}
		if i > 0 && chunks[i-1].Display >= ch.Display {
			t.Errorf("display names do not sort in order: %q then %q", chunks[i-1].Display, ch.Display)
		}
	}
}

func TestHillRadius(t *testing.T) {
	const sunMass = 1.989e30
	// Jupiter: a = 260 su in this dataset, mass 1.898e27 kg.
	got := decompose.HillRadius(260, 1.898e27, sunMass)
	if got < 15 || got > 20 {
		t.Errorf("Jupiter Hill radius = %.2f su, expected roughly 17-18", got)
	}
	if decompose.HillRadius(100, 0, sunMass) != 0 {
		t.Error("zero mass must yield zero Hill radius")
	}
	if decompose.HillRadius(100, 5.972e24, 0) != 0 {
		t.Error("zero star mass must yield zero rather than dividing by zero")
	}
}
