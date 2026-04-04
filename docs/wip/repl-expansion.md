# REPL Expansion — Implementation Plan

## Purpose
Extend the Space Sim REPL with six new command categories and context-aware TAB completion.
Each category maps to a new gRPC service, a new transport handler, new `commands.go` types, and new `repl.go` exec cases.

## Last Updated
2026-04-03

## Status Key

| Symbol | Meaning |
|--------|---------|
| 📋 | Not started |
| 🔄 | In progress |
| ✅ | Complete |
| ⏸ | Blocked |

---

## Design Decisions (locked)

| Topic | Decision |
|-------|----------|
| Main-thread safety | `AppCmd` channel in `App`; interactive loop drains it each frame (non-blocking). Same pattern as `speedChangeCh` in physics. |
| Camera direction format | Degrees: `camera orient <yaw> <pitch>`. Yaw 0 = +Z, range 0–360. Pitch −85 to +85. Converted to radians at the wire boundary. |
| Jump sequence | Option B — server-side. `NavigationService.JumpTo` accepts a list of names; the app drains the queue internally, firing each jump when `JumpProgress >= 1.0`. |
| Coordinate space | AU simulation-space for all position/navigation coordinates. `CoordSpace` field added to position messages for future abstraction. |
| Window restore | Restores to pre-maximize size; falls back to config defaults if no prior windowed size exists. |
| Performance access | `GetPerformance` / `SetPerformance` pair; single message carries all nine knobs. Partial-update via explicit bool flags (`set_*`). |
| Mouse sensitivity | API only (no REPL command). |
| TAB completion source | Body names fetched on first connect, cached until shutdown. Player IDs fetched on-demand per TAB press. |
| System list | `SystemService.ListSystems` reads `data/systems/` at call time (lazy, no cache — list is stable). `LoadSystem` writes `PendingSystemPath` via `AppCmd`. |
| `start` command | Not in scope. Use `make run-grpc`. `shutdown` is in scope (maps to a `ShutdownService` RPC). |

---

## Dependency Graph

```
Phase A (AppCmd channel)
    └── Phase B  (Window)
    └── Phase C  (Camera)
    └── Phase D  (Navigation)
    └── Phase E  (Performance)

Phase F (TAB completion) — independent of A–E
Phase G (System)         — independent of A–E; needs proto + handler
Phase H (Shutdown)       — independent of A–E; simple RPC
```

---

## Phase 0 — Proto + buf generate

**Value**: Defines all wire contracts for phases A–H before implementation touches any Go source.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Nothing

### Work Items

- [ ] Add `SystemService` (ListSystems, LoadSystem) to `simulation.proto`
- [ ] Add `WindowService` (GetWindow, SetWindowSize, SetWindowMaximize, SetWindowRestore) to `simulation.proto`
- [ ] Add `CameraService` (GetCamera, SetCameraOrient, SetCameraPosition, SetCameraTrack) to `simulation.proto`
- [ ] Add `NavigationService` (MoveDir, SetVelocity, GetVelocity, JumpTo) to `simulation.proto`
- [ ] Add `PerformanceService` (GetPerformance, SetPerformance) to `simulation.proto`
- [ ] Add `ShutdownService` (Shutdown) to `simulation.proto`
- [ ] Run `make proto` — verify generated Go stubs compile clean

### Acceptance Criteria

- `go build ./api/gen/...` passes with no errors
- All six new service interfaces present in generated code
- No hand-edits to `api/gen/` (generated only)

---

## Phase A — AppCmd channel (main-thread gate)

**Value**: Enables gRPC handlers running on HTTP goroutines to safely mutate Raylib-owned state (window, camera, navigation, performance) without data races.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Nothing (code only, no proto needed)

### Work Items

- [ ] Define `AppCmd` interface in `internal/client/go/raylib/app/` (new file: `appcmd.go`)
- [ ] Add concrete `AppCmd` types: `WindowSizeCmd`, `WindowMaximizeCmd`, `WindowRestoreCmd`, `CameraOrientCmd`, `CameraPositionCmd`, `CameraTrackCmd`, `SetVelocityCmd`, `JumpToCmd`, `PerfSetCmd`, `LoadSystemCmd`, `ShutdownCmd`
- [ ] Add `cmdCh chan AppCmd` (buffer 32) to `App`; initialize in `New()`
- [ ] Add `App.SendCmd(AppCmd)` — non-blocking send, drops + logs on full
- [ ] In `interactive.go` update loop: drain `cmdCh` with a non-blocking `select` at the top of each frame; dispatch to concrete handlers
- [ ] Run `go test -race ./internal/client/go/raylib/...` — must pass

### Acceptance Criteria

- No new data races under `-race`
- Channel is non-blocking in both directions (handler never blocks; frame loop never blocks)
- Dropping a command when the channel is full logs a warning but does not panic

---

## Phase B — Window service

**Value**: REPL can query and change window size, maximize, and restore.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0, Phase A

### Work Items

- [ ] `internal/transport/grpc/window_handler.go` — `WindowHandler` implementing `WindowServiceHandler`
  - `GetWindow`: reads `App.runtime` (safe from any goroutine; values are int32 atomics or set only from main thread after init)
  - `SetWindowSize`, `SetWindowMaximize`, `SetWindowRestore`: push `AppCmd` via `App.SendCmd`
- [ ] Wire `WindowHandler` in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` types: `WindowGet`, `WindowSize{W, H int}`, `WindowMaximize`, `WindowRestore`
- [ ] Add parse cases and exec cases
- [ ] Tests: handler unit tests (stub App); REPL integration test

### REPL Commands

```
window get                    print current size and state
window size <W>x<H>           e.g.  window size 1920x1080
window maximize
window restore
```

### Acceptance Criteria

- `window get` returns current dimensions
- `window size` changes window on screen
- `window maximize` / `window restore` work as described
- Tests pass with `-race`

---

## Phase C — Camera service

**Value**: REPL can read and set camera orientation, position, and tracking target.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0, Phase A

### Work Items

- [ ] `internal/transport/grpc/camera_handler.go` — `CameraHandler`
  - `GetCamera`: reads `App.currentCameraState()` (snapshot via AppCmd round-trip or atomic — TBD during impl)
  - `SetCameraOrient(yaw_deg, pitch_deg)`: push `CameraOrientCmd`
  - `SetCameraPosition(x, y, z, coord_space)`: push `CameraPositionCmd`
  - `SetCameraTrack(name)`: push `CameraTrackCmd` (name="" → free-fly)
- [ ] Wire in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` types: `CameraGet`, `CameraOrient{Yaw, Pitch float32}`, `CameraPosition{X, Y, Z float64}`, `CameraTrack{Name string}`
- [ ] Add parse cases and exec cases
- [ ] Tests

### REPL Commands

```
camera get                    print position, yaw, pitch, mode, track target
camera orient <yaw> <pitch>   yaw 0–360°, pitch −85–+85°
camera position <x> <y> <z>   AU coordinates
camera track <name>           continuous follow; "camera track" with no name → free-fly
```

### Acceptance Criteria

- `camera get` returns accurate state
- `camera orient 90 0` faces east (+X)
- `camera track Earth` switches mode to `CameraModeTracking` targeting Earth
- `camera track` (no name) returns to free-fly
- Tests pass with `-race`

---

## Phase D — Navigation service

**Value**: REPL can drive camera movement (WASD/arrow equivalents with velocity) and queue multi-hop jumps.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0, Phase A

### Work Items

- [ ] Define `VelocityState` in `ui.CameraState` (or a new sibling struct): `{Forward, Right, Up float32}` — persistent per-frame deltas drained in `updateCameraState`
- [ ] `internal/transport/grpc/navigation_handler.go` — `NavigationHandler`
  - `MoveDir(direction, velocity)`: push `SetVelocityCmd`
  - `SetVelocity(vx, vy, vz)`: push `SetVelocityCmd` (all axes at once)
  - `GetVelocity`: return current velocity vector from App
  - `JumpTo(names[]string)`: push `JumpToCmd` (server-side sequencing)
- [ ] In `interactive.go` `JumpToCmd` handler: replace `inputState.PendingJump` logic with a queue drawn down as `JumpProgress >= 1.0`
- [ ] Wire in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` types: `NavForward{V}`, `NavBack{V}`, `NavLeft{V}`, `NavRight{V}`, `NavUp{V}`, `NavDown{V}`, `NavStop`, `NavGetVelocity`, `NavJumpTo{Names []string}`
- [ ] Add parse cases and exec cases
- [ ] Tests

### REPL Commands

```
nav forward <v>               move forward at velocity v (AU/s), 0 = stop axis
nav back <v>
nav left <v>
nav right <v>
nav up <v>
nav down <v>
nav stop                      zero all continuous velocities
nav velocity                  print current velocity vector
nav jump <name> [, <name> …]  queue multi-hop animated jump sequence
```

### Acceptance Criteria

- `nav forward 10` causes visible forward camera drift each frame
- `nav stop` halts all drift
- `nav jump Earth, Saturn` animates Earth jump, then Saturn jump sequentially
- `nav velocity` returns the active vector
- Tests pass with `-race`

---

## Phase E — Performance service

**Value**: REPL can read and toggle all nine rendering/physics knobs.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0, Phase A

### Work Items

- [ ] `internal/transport/grpc/performance_handler.go` — `PerformanceHandler`
  - `GetPerformance`: snapshot `PerfOptions` + `NumWorkers` + `CameraSpeed`
  - `SetPerformance`: partial update via `set_*` bool flags; push `PerfSetCmd`
- [ ] Wire in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` types: `PerfGet`, `PerfSet{Field string, Value string}`
- [ ] Add parse cases and exec cases
- [ ] Tests

### REPL Commands

```
perf get                                  print all performance knobs
perf set frustum_culling <true|false>
perf set spatial_partition <true|false>
perf set lod <true|false>
perf set instanced_rendering <true|false>
perf set point_rendering <true|false>
perf set importance_threshold <n>
perf set camera_speed <n>
perf set workers <n>
```

### Acceptance Criteria

- `perf get` lists all nine fields
- `perf set frustum_culling false` disables culling visibly
- `perf set workers 2` changes physics worker count
- Tests pass with `-race`

---

## Phase F — TAB completion

**Value**: Context-sensitive completion on body names, category filters, dataset levels, and system files; player IDs on-demand.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Nothing (self-contained readline change)

### Work Items

- [ ] Add `completionCache` to `REPL`: populated once at connect time (`bodies` snapshot → all names + categories + system list)
- [ ] Expose `TabCompleter` interface in `repl.go`: `Complete(partial string, context string) []string`
- [ ] In `readline.go` raw path: intercept `0x09` (TAB) and `shift+TAB` (CSI `Z`)
  - Parse partial line to determine context (verb + argument position)
  - Call `completionCache.Complete(partial, context)`, cycle through matches
  - Render inline: erase partial, write match, track cycle index
- [ ] Context table:
  | Verb | Arg position | Completion pool |
  |------|-------------|-----------------|
  | `inspect` | 1 | all body names |
  | `bodies` | 1 | category names |
  | `setdataset` | 1 | dataset levels |
  | `camera track` | 2 | all body names |
  | `nav jump` | 1..N | all body names |
  | `system load` | 1 | system file labels |
  | Any | 0 (verb) | all verbs |
- [ ] Shift+TAB cycles completion list in reverse
- [ ] No match → bell (print `\a`)
- [ ] Tests: `completionCache` unit tests; `readline` TAB dispatch tests with table-driven cases

### Acceptance Criteria

- `inspect E<TAB>` completes to `inspect Earth` (or cycles through Earth, Europa…)
- `setdataset <TAB>` cycles through `small medium large huge`
- `nav jump Ear<TAB>` completes `Earth`
- `system load <TAB>` lists system files
- Shift+TAB reverses cycle
- Non-TTY path unaffected (existing tests unchanged)

---

## Phase G — System service

**Value**: REPL can list available solar systems, query the active one, and switch to another at runtime.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0 (proto), Phase A (AppCmd for LoadSystem)

### Work Items

- [ ] `internal/transport/grpc/system_handler.go` — `SystemHandler`
  - `ListSystems`: calls `discoverRuntimeSystemOptions()` (reads `data/systems/` at call time); returns label + path pairs
  - `GetActiveSystem`: reads `App.cfg.SystemConfig`
  - `LoadSystem(path)`: validates path is within `data/systems/`; pushes `LoadSystemCmd` via `App.SendCmd`
- [ ] Wire in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` types: `SystemList`, `SystemGet`, `SystemLoad{Label string}`
- [ ] Add parse cases and exec cases
- [ ] TAB completion for `system load` (Phase F)
- [ ] Tests

### REPL Commands

```
system list                   list available system files with labels
system get                    print active system path/label
system load <label>           switch to that system (animated session reload)
```

### Acceptance Criteria

- `system list` shows `alpha_centauri_system.json` and `solar_system.json` (and any others in `data/systems/`)
- `system load alpha_centauri_system` triggers the same reload path as the in-app selector
- `system get` reflects the newly active system after a load
- Path traversal outside `data/systems/` is rejected with an error
- Tests pass with `-race`

---

## Phase H — Shutdown service

**Value**: REPL can gracefully stop the running server.
**Status**: 📋 Not started
**Start Date**: —
**Depends on**: Phase 0 (proto)

### Work Items

- [ ] `internal/transport/grpc/shutdown_handler.go` — `ShutdownHandler`
  - `Shutdown`: cancels the app context via a cancel func injected at construction
- [ ] Wire cancel func in `cmd/space-sim-grpc/main.go`
- [ ] Add `commands.go` type: `Shutdown`
- [ ] Add parse case and exec case (prompts "server shutting down")
- [ ] Test

### REPL Commands

```
shutdown                      gracefully stop the space-sim-grpc server
```

### Acceptance Criteria

- `shutdown` causes `space-sim-grpc` to exit cleanly (exit code 0)
- REPL prints confirmation and exits
- Test passes with `-race`

---

## Execution Order

```
Phase 0  →  Phase A  →  Phase B
                     →  Phase C
                     →  Phase D (requires VelocityState design)
                     →  Phase E
                     →  Phase G (LoadSystem needs AppCmd)
Phase 0  →  Phase H  (no AppCmd needed)
Phase F  (any time after REPL compile target exists)
```

Commit after each phase passes `go test -race ./...`.

---

## Open Items / Risks

| Item | Status | Notes |
|------|--------|-------|
| Camera read from gRPC goroutine | 🔄 Needs decision | Options: atomic snapshot struct in App, or AppCmd round-trip with response channel. Atomic snapshot preferred for reads. |
| `updateCameraState` velocity drain | 📋 Design needed | Where the persistent `VelocityState` is stored and how it integrates with the existing WASD/arrow frame logic. |
| Player ID / session layer | ⏸ Deferred | Design pass required before implementation. Not in current scope. |
| `data/systems/` path at runtime | 📋 Verify | Server binary must be run from repo root; `system_selector.go` already assumes this. Confirm in Phase G. |

---

## Related Docs

- [docs/wip/todo.md](todo.md): main backlog
- [docs/standards/agent-readme.md](../standards/agent-readme.md): architecture and package map
- [docs/standards/coding-standards.md](../standards/coding-standards.md): Definition of Done
- [docs/history/lessons-learned.md](../history/lessons-learned.md): anti-patterns
- [docs/history/lessons-learned-double-buffering.md](../history/lessons-learned-double-buffering.md): concurrency constraints
