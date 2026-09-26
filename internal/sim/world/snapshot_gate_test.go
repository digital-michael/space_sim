package world

import (
	"sync/atomic"
	"testing"
)

// The gate exists to skip an O(objects) deep clone when nothing consumes it, so
// the contract that matters is: unset or nil means always build, and a false
// predicate suppresses building.
func TestSnapshotWantedDefaultsToTrue(t *testing.T) {
	var w World
	if !w.snapshotWanted() {
		t.Error("an unset gate must build snapshots; standalone mode relies on this")
	}
}

func TestSnapshotWantedRespectsGate(t *testing.T) {
	var w World
	var open atomic.Bool

	w.SetSnapshotGate(func() bool { return open.Load() })

	if w.snapshotWanted() {
		t.Error("closed gate must suppress snapshot building")
	}
	open.Store(true)
	if !w.snapshotWanted() {
		t.Error("open gate must allow snapshot building")
	}
}

// Passing nil must restore the always-build default rather than panicking or
// latching the previous predicate.
func TestSnapshotGateNilRestoresDefault(t *testing.T) {
	var w World
	w.SetSnapshotGate(func() bool { return false })
	if w.snapshotWanted() {
		t.Fatal("precondition: gate should be closed")
	}

	w.SetSnapshotGate(nil)
	if !w.snapshotWanted() {
		t.Error("nil gate must restore the always-build default")
	}
}

func TestSnapshotGateIsCalledEachTime(t *testing.T) {
	var w World
	var calls atomic.Int32
	w.SetSnapshotGate(func() bool {
		calls.Add(1)
		return true
	})

	for i := 0; i < 3; i++ {
		w.snapshotWanted()
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("gate called %d times, want 3 — the result must not be cached", got)
	}
}
