// Package protocol defines the wire contract between the simulation server
// and all clients, regardless of transport (in-process, gRPC, WebSocket, etc.).
package protocol

import "github.com/digital-michael/space_sim/internal/sim/engine"

// WorldSnapshot is a point-in-time, lock-free copy of simulation state.
// Build one with (*world.World).Snapshot(); consume it in any client
// without holding any simulation lock.
type WorldSnapshot struct {
	// State is a value-cloned snapshot taken while holding the front-buffer
	// read lock. Safe to read after Snapshot() returns.
	State *engine.SimulationState

	// Speed is the physics animation multiplier at snapshot time.
	Speed float64
}
