# Changelog

## Purpose
Capture completed work after it leaves the active backlog. This is a concise delivery history, not a full commit log.

## Last Updated
2026-03-30

## Table of Contents
1. How to Use This File
2. 2026-03 Delivered Work
	2.1 Quick Wins
	2.2 Runtime Loader and Engine Cleanup
	2.3 Runtime System Selector
	2.4 Phase 0 - Data-Driven Sol Outer Belt
	2.5 Phase 1 - Core GroupPool
	2.6 Phase 2 - Runtime Environment
	2.7 Phase 1+2 Integration into space_sim

## 1. How to Use This File

- Move finished work here from [docs/wip/todo.md](../wip/todo.md) once the work is complete.
- Record an `End Date` for each completed work item or section when it leaves the todo.
- Use `YYYY-MM-DD` for all `End Date` values.
- Record outcomes and validation, not full speculative planning detail.
- Keep entries concise enough to scan, but specific enough to recover what changed and why it mattered.

## 2. 2026-03 Delivered Work

### 2.1 Quick Wins

Completed early cleanup needed before larger workstreams.

**End Date**: 2026-03-26

| ID | Outcome |
|----|---------|
| Q1 | Deleted stale `internal/space/simulation.go.bak` |
| Q2 | Removed stale `Reserved for future dereference` comment from `cmd/space-sim/main.go` |
| Q3 | Made help layout constants responsive and closed the first fullscreen-related layout cleanup items |
| Q4 | Fullscreen and dynamic-resize implementation moved out of active backlog and treated as complete pending manual runtime verification |
| Q5 | Standardized dialog invocation keys so navigation dialogs stay on plain letters while system/config dialogs and actions move under a dedicated modifier, and the help/docs now match the live controls |
| Q6 | Standardized `Escape` as the close action for help, selection, and performance dialogs |
| Q7 | Moved system and display actions under `Ctrl+...`, blocked modified navigation keys from falling through, and made the help overlay modal like other dialogs |
| Q8 | Reduced tracking-menu invocation to a single `T` binding and removed the extra shifted tracking opener from the help/docs |

### 2.2 Runtime Loader and Engine Cleanup

**End Date**: 2026-03-28

Delivered outcomes:

- Implemented `ring_system` feature loading so rings can be defined through the feature pipeline instead of a silent stub path
- Converted the solar-system sample data from inline ring bodies to `ring_system` feature entries and removed duplicate representation
- Wired `ObjectPool` into clone-based double-buffer swaps so dynamic allocation mode now reuses transient front-buffer objects
- Removed unused ring helper constructors that were no longer part of the active loading path

Validation snapshot:

- Targeted and broader package tests passed for ring loading, object-pool clone mode, loader behavior, and existing UI/engine invariants
- The runtime queue no longer needs open backlog items for ring-system support or object-pool disposition

### 2.3 Runtime System Selector

**End Date**: 2026-03-29

Delivered outcomes:

- Added a modal runtime system selector that lists JSON system files from [data/systems/](../../data/systems)
- Bound the selector to `Cmd+S` in interactive mode without conflicting with the existing `Ctrl+...` system controls
- Implemented no-op close behavior when the currently loaded system is selected
- Added interactive-session replacement so a different selected system reloads cleanly without restarting the app window
- Added focused UI and app tests for selector state behavior, system discovery ordering/filtering, and safe session-creation failure handling

Validation snapshot:

- Manual runtime verification completed for open, cancel, no-op current-system confirmation, and switching between Solar and Alpha Centauri
- `go test ./internal/space/app ./internal/space/ui` passed after implementation

### 2.4 Phase 0 - Data-Driven Sol Outer Belt

**Status**: Complete
**End Date**: 2026-03-26

Delivered outcomes:

- Kept Sol outer-belt feature definition in `data/systems/solar_system.json`
- Used the shared belt creation and configuration path for both asteroid and Kuiper belt features
- Updated dataset switching so both `Asteroid-` and `KBO-` families allocate and toggle per dataset level
- Removed the assumption that a separate hardcoded Kuiper generator was still required
- Updated [internal/space/package.md](../../internal/space/package.md) to reflect the resolved gap

Validation snapshot:

- Kuiper belt object counts and orbital parameters are now JSON-driven
- Dataset switching affects both asteroid and Kuiper belt objects through the shared path

### 2.5 Phase 1 - Core GroupPool

**Status**: Complete
**End Date**: 2026-03-26

Delivered outcomes:

- Added pool type definitions and the shared `ObjectPool` interface
- Implemented object and group definitions with validation and cloning support
- Implemented DAG-based group hierarchy validation and locking support
- Implemented reverse membership lookup and synchronized parent-link handling
- Added CRUD, concurrency, cycle-detection, and depth-limit tests

Validation snapshot:

- `go test -race ./internal/server/pool/...` passed clean
- Reverse membership lookup is O(1) through `GroupsForMember`
- Group depth limit settled at 20

### 2.6 Phase 2 - Runtime Environment

**Status**: Complete
**End Date**: 2026-03-26

Delivered outcomes:

- Added runtime object and group state structures independent from definitions
- Added pluggable position initialization strategies
- Implemented group-state cache invalidation and recomputation flow
- Added query and list APIs for object and group state access
- Added benchmarks for cache hit, cache miss, invalidation, and batch query paths

Validation snapshot:

- Cache hit measured at 44.5 ns
- Cache miss measured at 2.57 us
- Cache invalidation measured at 620 ns for a 5-level hierarchy
- Reported test coverage was 87.2%

### 2.8 Raylib Package Relocation

**End Date**: 2026-03-30

Delivered outcomes:

- Moved `internal/space/raylib/spatial/` → `internal/client/go/raylib/spatial/`
- Moved `internal/space/raylib/ui/render/` → `internal/client/go/raylib/ui/render/`
- Updated the 3 import sites in `internal/space/app/` (app.go, interactive.go, performance.go)
- Deleted the now-empty `internal/space/raylib/` tree
- Established `internal/client/go/raylib/` as the canonical path for Raylib-specific rendering and spatial packages; future graphics backends can sit alongside it under `internal/client/go/`

Validation snapshot:

- `go build ./...` clean
- `go test ./internal/...` — all packages passed, including the relocated `client/go/raylib/ui/render` test

**End Date**: 2026-03-30

Delivered outcomes:

- Copied `internal/server/` (pool, pool/group, runtime, eventqueue, eventloop, routines) from the `raylibsim` workspace into this repository
- Copied `internal/client/` stub directories (commands, repl) — empty, reserved for Phase 6
- Rewrote all import paths from `github.com/digital-michael/raylibsim` to `github.com/digital-michael/space_sim`
- Added `github.com/google/uuid v1.6.0` to `go.mod` and `go.sum`

Validation snapshot:

- `go build ./internal/server/...` clean
- `go test -race ./internal/server/...` — all 6 packages passed: eventloop, eventqueue, pool, pool/group, routines, runtime
- `go test ./internal/space/...` — all existing space packages unaffected