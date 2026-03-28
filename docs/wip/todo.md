# Space Sim Work Queue

## Purpose
Track active and future work for Space Sim in one operational backlog. Keep this file focused on work that is not yet done.

## Last Updated
2026-03-28

## Table of Contents
1. How to Use This File
2. Status Key
3. Active Work
	3.1 Runtime System Selector
4. Planned Phases
	4.1 Phase 3 - Event Queue System
	4.2 Phase 4 - Event Loop and Worker Pool
	4.3 Phase 5 - Persistence
	4.4 Phase 6 - gRPC Integration
	4.5 Phase 7 - Additional Pool Types
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

### 3.1 Runtime System Selector

**Value**: Enable fast manual testing by allowing the running app to switch between JSON systems from `data/systems/` without a restart.
**Status**: 📋 Not started
**Start Date**: Not started
**Ref**: [data/systems/](../../data/systems)

#### Work Items

- [ ] Add a fixed modal selector listing JSON files from `data/systems/`
- [ ] Bind the selector to `Cmd+L`
- [ ] Close the selector without action when the selected system is already loaded
- [ ] Reload the session cleanly when a different system is selected
- [ ] Add focused tests for selector behavior where practical and stop for manual runtime testing

#### Acceptance Criteria

- `Cmd+L` opens a system selector from the running app
- Selecting the already-loaded system behaves like cancel and closes the selector
- Selecting a different system reloads the active session cleanly
- Work stops after implementation for manual runtime verification

## 4. Planned Phases

### 4.1 Phase 3 - Event Queue System

**Value**: Enables per-GUID FIFO ordering with cross-GUID parallelism.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 1, Phase 2

#### Work Items

- [ ] Add event and transaction types plus the core event envelope
- [ ] Implement per-GUID FIFO queues with graceful queue-full handling
- [ ] Implement queue manager fan-out, dequeue, and metrics
- [ ] Add rollback, best-effort, and no-transaction execution modes
- [ ] Add concurrency and rollback tests

#### Acceptance Criteria

- Per-GUID ordering holds under concurrent load
- Cross-GUID work can execute in parallel
- Queue-full returns an error instead of panicking
- Rollback restores state on failure
- Race-enabled tests pass

### 4.2 Phase 4 - Event Loop and Worker Pool

**Value**: Turns the queue and runtime layers into a running server-side simulation loop.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 1, Phase 2, Phase 3

#### Work Items

- [ ] Add a multi-threaded worker pool with drain and shutdown support
- [ ] Implement an event loop with runtime FPS control
- [ ] Execute queued events before routine execution each frame
- [ ] Add routine registration and removal APIs
- [ ] Capture frame timing metrics and integration tests

#### Acceptance Criteria

- Target FPS stays within tolerance under normal load
- `SetFPS` takes effect without restart
- Frame timing metrics are queryable
- Race-enabled tests pass

### 4.3 Phase 5 - Persistence

**Value**: Enables save, restore, crash recovery, and deterministic replay.
**Status**: 📋 Not started
**Start Date**: Not started
**Depends on**: Phase 1, Phase 2

#### Work Items

- [ ] Implement JSON definition save and load with atomic writes
- [ ] Implement protobuf snapshot save and load
- [ ] Implement append and replay for the event log
- [ ] Add non-blocking autosave
- [ ] Add round-trip, replay, and corrupt-file tests

#### Open Questions

- Event log format: JSON or protobuf
- Persistence backend: file-only or optional SQLite

#### Acceptance Criteria

- Definitions and snapshots round-trip correctly
- Event replay reproduces the same runtime state
- Auto-save does not stall the event loop
- Corrupt files fail with errors, not panics

### 4.4 Phase 6 - gRPC Integration

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

### 4.5 Phase 7 - Additional Pool Types

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

## 5. Related Docs

- [docs/history/changelog.md](../history/changelog.md): completed work moved out of the live queue
- [docs/standards/guidance.md](../standards/guidance.md): workflow and work-tracking rules
- [internal/space/package.md](../../internal/space/package.md): current package and architecture context
- [docs/standards/agent-readme.md](../standards/agent-readme.md): repository orientation for agents