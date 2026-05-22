// Package protocol defines the wire contract between the simulation server
// and all clients, regardless of transport (in-process, gRPC, WebSocket, etc.).
package protocol

import "github.com/digital-michael/space_sim/internal/sim/engine"

// SnapshotSource is the read side of the snapshot pipeline. Any type that
// can serve a ready-to-render WorldSnapshot without locking implements this
// interface — both *world.World (local sim) and a gRPC stream consumer
// (remote client) satisfy it.
type SnapshotSource interface {
	LatestSnapshot() WorldSnapshot
}

// ClientSessionSnapshot is a point-in-time copy of a registered session's
// public state. Populated by the session registry; consumed by the Raylib
// renderer to draw per-client position markers (F-021 hook).
type ClientSessionSnapshot struct {
	SessionID string
	Label     string
	Color     [3]uint8
	Position  [3]float64
	POV       [3]float32
}

// WorldSnapshot is a point-in-time, lock-free copy of simulation state.
// Build one with (*world.World).Snapshot(); consume it in any client
// without holding any simulation lock.
type WorldSnapshot struct {
	// State is a value-cloned snapshot taken while holding the front-buffer
	// read lock. Safe to read after Snapshot() returns.
	State *engine.SimulationState

	// Speed is the physics animation multiplier at snapshot time.
	Speed float64

	// ClientSessions lists all registered sessions at snapshot time.
	// Populated by the gRPC WorldHandler from the session registry.
	// The Raylib renderer stores this for future F-021 marker rendering.
	ClientSessions []ClientSessionSnapshot
}
