# Space Sim Work Queue

## Purpose
Track active and future work for Space Sim in one operational backlog. Keep this file focused on work that is not yet done.

## Last Updated
2026-04-03


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
**Status**: ✅ Complete — 2026-04-03
**Start Date**: 2026-04-03
**Depends on**: Phase 1, Phase 2

#### Work Items

- [x] Implement JSON definition save and load with atomic writes
- [x] Implement JSON snapshot save and load
- [x] Implement append and replay for the event log
- [x] Add non-blocking autosave subscriber
- [x] Add round-trip, replay, and corrupt-file tests

#### Decisions

- Event log format: **JSON lines** (one event per line)
- Persistence backend: **file-only** with atomic rename (no SQLite)

### 4.4 Pre-Phase-6 Gate - Client/App Package Split

**Value**: Establishes a clean `internal/client/` vs `internal/server/` import graph before gRPC handlers are wired, avoiding a forced mid-Phase-6 restructure.
**Status**: ✅ Complete — 2026-04-03
**Start Date**: 2026-04-03
**Depends on**: Phase 5 complete or in final stabilization

#### What Was Done

- Created `internal/api/` as the transport-agnostic contract layer (ports-and-adapters).
  - `client.go`: `CameraController`, `PlayerView` interfaces with TODO stubs for Phase 6 (Zoom, Pan, Orbit, CameraPosition, etc.)
  - `server.go`: `SimulationControl`, `AnimationControl` interfaces with TODO stubs (Pause, Resume, LoadWorld, SeekToTime, etc.)
  - `doc.go`: package rationale
- Confirmed import boundary: `internal/api` carries no deps on `internal/sim`, `internal/client`, or `internal/server`.
- Updated `agent-readme.md`: Repository Map, Package Doc Index, Layered View, Architectural Boundaries, Preserved Refactor Intent, and Startup Flow — all stale `internal/space/` paths replaced with actual paths.
- `go vet ./internal/api/...` passes clean.

### 4.5 Phase 6 - gRPC Integration

**Value**: Connects live server components to client-facing transport via ConnectRPC (Apache 2.0, v1.19.1).
**Status**: ✅ Complete — 2026-04-03
**Start Date**: 2026-04-03
**Depends on**: Phase 1 through Phase 5

#### Binary Model

- `space-sim-direct` — Raylib client + in-process server. No network transport. Current working binary.
- `space-sim-grpc` (Phase 6 target, Option A) — Raylib client + embedded ConnectRPC server in one process. Client dials `localhost`. Full wire path without two processes.
- Option B (future) — Split into `space-sim-server` and `space-sim-client`. Player identification and registration handled on gRPC connection. JS/browser client connects to the same server binary.

#### Decisions

- **Command RPCs acknowledge queueing immediately** and return an `event_id`. Events are async; clients query state separately (CQRS pattern).
- **Transport**: ConnectRPC. Server natively speaks gRPC + gRPC-Web + Connect protocols. No proxy needed for browser clients.
- **Proto/generated code location**: `api/proto/spacesim/v1/` (public, importable by 3rd parties); generated Go at `api/gen/spacesim/v1/`.
- **`internal/api/`** remains internal-only; it defines Go interface ports, not the wire contract.

#### Sub-phases

- **6a**: Toolchain + proto. Add `connectrpc.com/connect` and `google.golang.org/protobuf` to `go.mod`. Write `api/proto/spacesim/v1/simulation.proto`. Add `buf.yaml` + `buf.gen.yaml`. Generate Go stubs into `api/gen/`.
- **6b**: Server scaffold. `internal/transport/grpc/` package. Start/Stop lifecycle. DI wiring: inject `internal/api/` interfaces into Raylib app constructor; provide direct adapter (in-process) and ConnectRPC adapter.
- **6c**: Handler implementations. `SimulationService` and `WorldService` handlers delegating to `eventqueue` and `runtime`.
- **6d**: Connection limit + idle timeout interceptors.
- **6e**: Integration tests (bufconn). Command round-trip, over-limit rejection, snapshot stream.

#### Work Items

- [x] 6a: Add ConnectRPC + protobuf deps to go.mod
- [x] 6a: Write simulation.proto (SimulationService, WorldService); all messages carry version field
- [x] 6a: Add buf.yaml + buf.gen.yaml; generate Go stubs
- [x] 6b: Create internal/transport/grpc/ scaffold with Start/Stop
- [x] 6b: Wire internal/api/ interfaces into Raylib app constructor (direct adapter)
- [x] 6c: Implement SimulationService handler (SetSpeed, GetSpeed, SetDataset, GetDataset, GetSimulationTime)
- [x] 6c: Implement WorldService handler (StreamSnapshot from protocol.Broadcaster)
- [x] 6d: Connection limit + idle timeout interceptors
- [x] 6e: Integration tests (transport routing, connection limit, WorldHandler fan-out)

#### Acceptance Criteria

- All intended REPL commands map cleanly to transport handlers
- Command RPCs return queued ack with event_id immediately
- Over-limit connections are rejected with ResourceExhausted
- Idle clients are disconnected as configured
- Snapshot stream delivers WorldSnapshot to connected clients
- `space-sim-grpc` builds and runs end-to-end against the embedded server

### 4.6 Phase 7 - Additional Pool Types

**Value**: Adds specialized pool strategies after the main server path is stable.
**Status**: ✅ Complete — 2026-04-03
**Start Date**: 2026-04-03
**Depends on**: Phase 1 through Phase 6

#### Work Items

- [x] Add `SimplePool` (`internal/server/pool/simple/`)
- [x] Add `DistributedPool` stub (`internal/server/pool/distributed/`)
- [x] Add pool factory (`internal/server/pool/factory/`)
- [x] Benchmark alternative pool strategies (SimplePool 388 ns/op, GroupPool 397 ns/op — equivalent)
- [x] Update docs after implementation

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