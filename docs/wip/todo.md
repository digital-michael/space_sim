# Space Sim Work Queue

## Purpose
Track active and future work for Space Sim in one operational backlog. Keep this file focused on work that is not yet done.

## Last Updated
2026-03-30

## Table of Contents
1. How to Use This File
2. Status Key
3. Active Work
4. Planned Phases
	4.1 Phase 3 - Event Queue System
	4.2 Phase 4 - Event Loop and Worker Pool
	4.3 Phase 5 - Persistence
	4.4 Pre-Phase-6 Gate - Client/App Package Split
	4.5 Phase 6 - gRPC Integration
	4.6 Phase 7 - Additional Pool Types
5. Related Docs

## 1. How to Use This File

- Keep only active, queued, blocked, or deferred work here.
- Move finished work to [docs/history/changelog.md](../history/changelog.md) instead of leaving completed sections in the live queue.
- Add a `Start Date` when a work item or section actually begins.
- Use `YYYY-MM-DD` for all `Start Date` values.
- Keep each section execution-oriented: value, status, dependencies, concrete work items, and acceptance criteria.
- If a task needs a deep design write-up, put that write-up in a separate document under `docs/` and leave a short pointer here.

## 2. Status Key

| Symbol | Meaning |
|--------|---------|
| 📋 | Not started |
| 🔄 | In progress |
| ⏸ | Blocked or deferred |

## 3. Active Work

### Phase 3 (session) - SnapshotBroadcaster

**Value**: Wires the physics loop to the server broadcast layer. After each engine tick the snapshot flows from `World.Snapshot()` through `protocol.Broadcaster` to all registered subscribers, enabling future gRPC and multi-client fan-out without coupling the renderer to `*World`.
**Status**: ✅ Complete — 2026-04-02
**Depends on**: Phase 1, Phase 2

#### Work Items

- [x] Add `protocol.Broadcaster` and `protocol.Subscriber` types with thread-safe register/unregister/push
- [x] Add broadcaster tests (concurrent push, unregister, no-op unregister)
- [x] Add `broadcaster *protocol.Broadcaster` field to `App`; initialize in `New()`; expose `RegisterSubscriber`
- [x] Push each `WorldSnapshot` to `app.broadcaster` in the interactive render loop

## 4. Planned Phases

### 4.1 Phase 3 - Event Queue System

**Value**: Enables per-GUID FIFO ordering with cross-GUID parallelism.
**Status**: ✅ Complete
**Start Date**: Pre-existing
**Depends on**: Phase 1, Phase 2

#### Work Items

- [x] Add event and transaction types plus the core event envelope
- [x] Implement per-GUID FIFO queues with graceful queue-full handling
- [x] Implement queue manager fan-out, dequeue, and metrics
- [x] Add rollback, best-effort, and no-transaction execution modes
- [x] Add concurrency and rollback tests

#### Acceptance Criteria

- Per-GUID ordering holds under concurrent load ✓
- Cross-GUID work can execute in parallel ✓
- Queue-full returns an error instead of panicking ✓
- Rollback restores state on failure ✓
- Race-enabled tests pass ✓

### 4.2 Phase 4 - Event Loop and Worker Pool

**Value**: Turns the queue and runtime layers into a running server-side simulation loop.
**Status**: ✅ Complete
**Start Date**: Pre-existing
**Depends on**: Phase 1, Phase 2, Phase 3

#### Work Items

- [x] Add a multi-threaded worker pool with drain and shutdown support
- [x] Implement an event loop with runtime FPS control
- [x] Execute queued events before routine execution each frame
- [x] Add routine registration and removal APIs
- [x] Capture frame timing metrics and integration tests

#### Acceptance Criteria

- Target FPS stays within tolerance under normal load ✓
- `SetFPS` takes effect without restart ✓
- Frame timing metrics are queryable ✓
- Race-enabled tests pass ✓

### 4.3 Phase 5 - Persistence

**Value**: Enables save, restore, crash recovery, and deterministic replay.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 1, Phase 2

#### Work Items

- [ ] Implement JSON definition save and load with atomic writes
- [ ] Implement JSON snapshot save and load
- [ ] Implement append and replay for the event log
- [ ] Add non-blocking autosave subscriber
- [ ] Add round-trip, replay, and corrupt-file tests

#### Decisions

- Event log format: **JSON lines** (one event per line)
- Persistence backend: **file-only** with atomic rename (no SQLite)

#### Acceptance Criteria

- Definitions and snapshots round-trip correctly
- Event replay reproduces the same runtime state
- Auto-save does not stall the event loop
- Corrupt files fail with errors, not panics

### 4.4 Pre-Phase-6 Gate - Client/App Package Split

**Value**: Establishes a clean `internal/client/` vs `internal/server/` import graph before gRPC handlers are wired, avoiding a forced mid-Phase-6 restructure.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 5 complete or in final stabilization

#### Context

The Raylib rendering packages have already moved to `internal/client/go/raylib/` (done 2026-03-30). The remaining work is moving the application orchestration layer. See [docs/technical.md](../technical.md) section 2.6 for the full rationale.

Two open design questions to evaluate before starting:
- Whether `internal/space/` stays as shared domain logic or splits into server-domain and client-domain sub-packages.
- Whether `internal/space/app/` moves wholesale to `internal/client/` or refactors into a thinner client-side adapter over a shared domain layer.

#### Work Items

- [ ] Evaluate `internal/space/` split vs. keeping as shared domain (based on Phase 6 gRPC ownership needs)
- [ ] Move `internal/space/app/` to `internal/client/` and update all import sites
- [ ] Update `cmd/space-sim/main.go` imports accordingly
- [ ] Verify build, all tests, and runtime smoke test
- [ ] Update agent-readme.md package map and doc.go files

#### Acceptance Criteria

- `internal/client/` imports from `internal/space/` or shared domain only — never from `internal/server/`
- `internal/server/` has no imports from `internal/client/`
- All tests pass with race detector enabled
- `cmd/space-sim` builds and runs the interactive session correctly

### 4.5 Phase 6 - gRPC Integration

**Value**: Connects live server components to client-facing transport.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 1 through Phase 5

#### Work Items

- [ ] Add `version` fields to all proto messages
- [ ] Wire RPC handlers to queue and runtime APIs
- [ ] Add connection limit enforcement
- [ ] Add idle timeout handling
- [ ] Add end-to-end integration tests

#### Open Questions

- Should command RPCs acknowledge queueing immediately or wait for applied results

#### Acceptance Criteria

- All intended REPL commands map cleanly to transport handlers
- Over-limit connections are rejected gracefully
- Idle clients are disconnected as configured
- Query RPCs return current runtime state directly

### 4.6 Phase 7 - Additional Pool Types

**Value**: Adds specialized pool strategies after the main server path is stable.
**Status**: ⏸ Deferred until Phase 6 is stable
**Start Date**: Not started
**Depends on**: Phase 1 through Phase 6

#### Work Items

- [ ] Add `SimplePool`
- [ ] Add `DistributedPool` stub
- [ ] Add pool factory wiring
- [ ] Benchmark alternative pool strategies
- [ ] Update docs after implementation

### 4.7 Belt Generation Quality - Overlap and Speed Uniqueness

**Value**: Prevents near-coincident belt objects from strobing or appearing to flicker at high dataset counts due to two objects occupying the same orbital position and speed.
**Status**: ⏸ Deferred — low priority, cosmetic only
**Start Date**: Not started
**Depends on**: None (self-contained change to `internal/sim/belts.go`)

#### Context

`CreateBelt` draws `orbitAngle` and `distanceAU` uniformly at random with no exclusion zone around already-placed objects. At large datasets (1,200–24,000 objects) near-coincident pairs are statistically likely. Two objects at the same `(distance, orbitAngle)` have identical Keplerian periods so they track together forever, appearing as a single strobing object when rendered on top of each other.

#### Work Items

- [ ] Enforce a minimum angular separation per orbital shell in `CreateBelt` (retry or stratified placement)
- [ ] Ensure no two objects in the same shell share both distance and angle within a configurable tolerance
- [ ] Add a test asserting minimum separation across a large generated dataset
- [ ] Consider whether `MeanAnomalyAtEpoch` jitter alone is sufficient or structural placement is needed

## 5. Related Docs

- [docs/history/changelog.md](../history/changelog.md): completed work moved out of the live queue
- [docs/standards/guidance.md](../standards/guidance.md): workflow and work-tracking rules
- [internal/space/package.md](../../internal/space/package.md): current package and architecture context
- [docs/standards/agent-readme.md](../standards/agent-readme.md): repository orientation for agents