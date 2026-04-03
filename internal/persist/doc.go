// Package persist provides Save, Load, and replay functions for Space Sim
// simulation state. It writes to the local filesystem using atomic
// rename-over-temp semantics so partial writes never corrupt saved state.
//
// Sub-files:
//
//   - definitions.go  Meta-only layer (name, orbital elements, etc.)
//   - snapshot.go     Full state including per-frame AnimationState
//   - eventlog.go     Append-only JSON-lines event log + Replay
//   - autosave.go     Non-blocking single-slot autosave
package persist
