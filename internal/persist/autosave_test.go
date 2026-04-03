package persist_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/digital-michael/space_sim/internal/persist"
)

func TestAutosave_WritesSomething(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "autosave.json")

	as := persist.NewAutosave(path)
	as.Start()

	snap := buildTestSnapshot()
	as.Receive(snap)

	// Give the background writer time to flush.
	time.Sleep(50 * time.Millisecond)

	as.Stop()

	// The file must exist and be loadable.
	loaded, err := persist.LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot after autosave: %v", err)
	}
	if loaded.Speed != snap.Speed {
		t.Errorf("Speed: got %v, want %v", loaded.Speed, snap.Speed)
	}
}

func TestAutosave_NonBlockingUnderLoad(t *testing.T) {
	dir := t.TempDir()
	as := persist.NewAutosave(filepath.Join(dir, "autosave.json"))
	as.Start()

	snap := buildTestSnapshot()

	// Flood Receive — none must block.
	start := time.Now()
	for i := 0; i < 1000; i++ {
		as.Receive(snap)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("1000 Receive calls took %v; expected non-blocking behaviour", elapsed)
	}

	as.Stop()
}

func TestAutosave_StopWaitsForCurrentWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "autosave.json")
	as := persist.NewAutosave(path)
	as.Start()

	as.Receive(buildTestSnapshot())
	// Stop must not return until the pending write completes.
	as.Stop()

	// If Stop returned before the write finished the file would be missing.
	if _, err := persist.LoadSnapshot(path); err != nil {
		t.Fatalf("snapshot not durable after Stop: %v", err)
	}
}
