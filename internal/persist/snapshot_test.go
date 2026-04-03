package persist_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/digital-michael/space_sim/internal/persist"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

func buildTestSnapshot() protocol.WorldSnapshot {
	state := buildTestState()
	state.Time = 12345.678
	state.DeltaTime = 0.016

	// Give objects non-zero animation state so round-trip coverage is real.
	state.Objects[0].Anim = engine.AnimationState{
		Position:    engine.Vector3{X: 0, Y: 0, Z: 0},
		MeanAnomaly: 0.5,
		TrueAnomaly: 0.51,
		OrbitAngle:  0.51,
		OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
	}
	state.Objects[1].Anim = engine.AnimationState{
		Position:     engine.Vector3{X: 100, Y: 0, Z: 50},
		Velocity:     engine.Vector3{X: -0.1, Y: 0, Z: 0.2},
		OrbitCenter:  engine.Vector3{X: 0, Y: 0, Z: 0},
		MeanAnomaly:  1.2,
		TrueAnomaly:  1.21,
		OrbitAngle:   1.21,
		OrbitAxis:    engine.Vector3{X: 0, Y: 1, Z: 0},
		OrbitYOffset: 0.5,
	}

	return protocol.WorldSnapshot{State: state, Speed: 3600}
}

func TestSnapshotRoundTrip(t *testing.T) {
	original := buildTestSnapshot()

	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	if err := persist.SaveSnapshot(path, original); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	loaded, err := persist.LoadSnapshot(path)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}

	// Top-level snapshot fields.
	if loaded.Speed != original.Speed {
		t.Errorf("Speed: got %v, want %v", loaded.Speed, original.Speed)
	}
	if loaded.State.Time != original.State.Time {
		t.Errorf("Time: got %v, want %v", loaded.State.Time, original.State.Time)
	}
	if loaded.State.DeltaTime != original.State.DeltaTime {
		t.Errorf("DeltaTime: got %v, want %v", loaded.State.DeltaTime, original.State.DeltaTime)
	}

	// Object count.
	if got, want := len(loaded.State.Objects), len(original.State.Objects); got != want {
		t.Fatalf("object count: got %d, want %d", got, want)
	}

	for i, orig := range original.State.Objects {
		got := loaded.State.Objects[i]
		if got.Meta.Name != orig.Meta.Name {
			t.Errorf("Objects[%d].Meta.Name: got %q, want %q", i, got.Meta.Name, orig.Meta.Name)
		}
		// AnimationState must survive round-trip.
		if got.Anim != orig.Anim {
			t.Errorf("Objects[%d].Anim: got %+v, want %+v", i, got.Anim, orig.Anim)
		}
	}
}

func TestLoadSnapshot_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	if err := persist.SaveSnapshot(path, buildTestSnapshot()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := persist.LoadSnapshot(path)
	if err == nil {
		t.Error("expected error loading corrupt snapshot, got nil")
	}
}

func TestSaveSnapshot_NilState(t *testing.T) {
	snap := protocol.WorldSnapshot{State: nil, Speed: 1.0}
	err := persist.SaveSnapshot(t.TempDir()+"/snap.json", snap)
	if err == nil {
		t.Error("expected error for nil State, got nil")
	}
}
