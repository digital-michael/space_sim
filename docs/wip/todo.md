# Space Sim Work Queue

## Purpose
Track active and future work for Space Sim in one operational backlog. Keep this file focused on work that is not yet done.

## Last Updated
2026-04-26

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
	4.7 Belt Generation Quality - Overlap and Speed Uniqueness
	4.8 UX Polish - Rendering, Camera, and Config
5. Defects
	DEF-001 Floating-Point Precision Collapse at Extreme Camera Distances
	DEF-002 Stars Disappear at Outer-Planet Camera Distances
	DEF-003 Sol Atmosphere Missing
	DEF-004 Objects Clip or Disappear at a Specific Camera Distance
6. Feature Backlog
	F-001 Camera Collision Prevention

Planning Documents
	[f013-nbody-plan.md](f013-nbody-plan.md) — F-013 implementation plan (phases, algorithms, validation)
	[f020-multi-client-spec.md](f020-multi-client-spec.md) — F-020 multi-client gRPC session layer spec
	[f021-physical-marker-spec.md](f021-physical-marker-spec.md) — F-021 client physical marker spec
	[f022-client-movement-spec.md](f022-client-movement-spec.md) — F-022 client locomotion and physics spec
	[f023-keyboard-config-spec.md](f023-keyboard-config-spec.md) — F-023 keyboard configuration spec
	[f024-multiplayer-hud-spec.md](f024-multiplayer-hud-spec.md) — F-024 multiplayer HUD enhancements spec
	[f025-ship-comms-spec.md](f025-ship-comms-spec.md) — F-025 ship-to-ship communications spec
	[f026-audio-events-spec.md](f026-audio-events-spec.md) — F-026 audio events spec
	[f027-collision-damage-spec.md](f027-collision-damage-spec.md) — F-027 ship collision detection and damage spec
	F-002 REPL: track <object> and track stop ✅
	F-003 Texture/Bitmap Rendering
	F-004 Procedural Star Field Background
	F-005 Physical Lighting from Stars
	F-006 XYZ Keyboard Navigation + Mouse Facing
	F-007 User-Configurable Key Bindings
	F-008 Artifact Object Type
	F-009 Object-Object Collision / Proximity Detection
	F-010 Multi-Machine Architecture (Option B split)
	F-011 IAAM — Identity, Access, Authentication, and Management
	F-012 Federated Compute — Collaborative Simulation Offload
	F-013 N-Body Barycenter Integration
	F-014 Nearby Systems Expansion Backlog
	F-015 Epoch-Accurate Initial Mean Anomaly ("Start From Today")
	F-016 Wire Rendering Data Pipeline (Schema → Engine → Renderer)
	F-017 Realistic Lighting (Shadows, Atmosphere, Bloom, PBR)
	F-018 Object Annotations HUD (outlines, axes, orbital paths, labels)
	F-019 Run Scripts from UI
	F-020 Multi-Client gRPC Session Layer
	F-021 Client Physical Marker
	F-022 Client Locomotion and Physics
	F-023 Keyboard Configuration
	F-024 Multiplayer HUD Enhancements
	F-025 Ship-to-Ship Communications
	F-026 Audio Events
	F-027 Ship Collision Detection and Damage
	F-028 Input Config as Reusable UI Component
	F-029 Enhanced Sol Corona / Active Atmosphere
	F-030 Solar Weather Events (Flares, CMEs, Particle Storms)
	F-031 Asteroid Visual Classification by Mass
	F-032 ⚠️ Integrate Keybindings into All Simulator Commands (HIGH PRIORITY)
7. Recommended Ordering
8. Tech Debt
	TD-001 Collapse handleInput / updateCameraState Param Lists
	TD-002 Decouple Sim Tick from Render/Input Loop
9. Related Docs

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

### 4.8 UX Polish — Rendering, Camera, and Config Improvements

**Value**: Addressed accumulated rendering quality issues, sticky CLI flags, startup experience, and camera tracking regression.
**Status**: ✅ Complete — 2026-04-08
**Start Date**: 2026-04-07
**Depends on**: Phase 6 complete

#### Work Items

- [x] Enable MSAA 4× by default; add `--no-msaa` to opt out
- [x] Add `--reset` flag: writes factory defaults to `app.json` and exits
- [x] Fix CLI render flags (`--render-scale`, `--render-size`) incorrectly persisting to `app.json` on exit
- [x] Fix time rate HUD showing animation speed instead of simulation seconds-per-second
- [x] Revert HiDPI coordinate queries from `GetRenderWidth/Height` to `GetScreenWidth/Height` (regression fix)
- [x] Set initial tracking camera distance to star surface + 0.75 AU on startup

---

## 7. Recommended Ordering

This is the current best-guess execution sequence integrating dependency order, the three planning observations, and DEF-001. The owner manages actual sprint assignment.

| Step | Item | Rationale |
|------|------|-----------|
| 1 | **F-013** N-body barycenter | Physics accuracy first; single-process validation before distributing |
| 2 | **F-001** Camera collision prevention | 30-min fix; affects every session; pull forward from Group 1 |
| 3 | **DEF-001** Floating-origin exploration + fix | Touches same render sites as F-003/F-004/F-005; fix before visual work to avoid double-rewrite |
| 4 | **TD-001** Collapse handleInput param lists | Clean up before Group 5 network work adds more code around it |
| 4a | **F-023 Phase 1** `DrainQueue()` mitigation | Partial fix for input latency under load; ships with keyboard config refactor |
| 4b | **TD-002** Decouple sim tick from render/input loop | Full fix for input latency; also the architectural pattern F-010 headless split needs |
| 5 | **F-010** Multi-machine split (headless server + client) | Network foundation |
| 6 | **F-011** IAAM (identity, roles, auth) | Safety layer for multi-client; immediately after F-010 |
| 7 | **F-008** Artifact object type | Content foundation for F-009 |
| 8 | **F-009** Object-object collision/proximity | Needs F-008 for full value |
| 9 | **F-007** User-configurable key bindings → **superseded by F-023** | F-023 is the full spec for this; see F-007 entry for annotation |
| 10 | **F-006** XYZ keyboard nav + mouse facing → **deferred until F-023 Phase 1** | F-023 Phase 3 covers mouse-delta; do F-023 first |
| 11 | **F-002** REPL track / track stop | Remaining Group 1 quick win |
| 12 | **F-003** Textures on planets/moons | Group 2 visual; floating-origin fix (DEF-001) already done |
| 13 | **F-004** Procedural star field | Group 2 visual |
| 14 | **F-005** Physical lighting from stars | Needs F-003 |
| 15 | **F-012** Federated compute | Long-term exploratory; F-010, F-011, F-013 must be stable. Own phase. |
| 16 | **F-018** Object annotations HUD | High value, low risk; no engine changes; builds on existing label + render infra |
| 17 | **F-019** Run scripts from UI | Medium priority; builds directly on REPL and existing dialog UX |
| — | **4.7** Belt overlap/speed uniqueness | ⏸ Deferred — cosmetic only, insert whenever bandwidth allows |

---

## 8. Tech Debt

### TD-001 — Collapse `handleInput` / `updateCameraState` Param Lists

**Value**: `handleInput` currently takes 14 individual params and returns a 6-tuple. `updateCameraState` takes 8. Both functions have access to `*App`, `*runtimeSession`, and `*engine.SimulationState` at their call site in `interactive.go`, but receive their fields piecemeal instead.
**Status**: 📋 Not started
**Constraint**: Do not implement until all active phases are stable and a clean test baseline exists.

#### Design (agreed 2026-04-04)

Three targeted changes, each independently safe:

1. **Pass `*runtimeSession` to both functions** — eliminates `cameraState`, `inputState`, `navigationOrder`, and `sim` as separate params; they're already fields on the session.
2. **Pass `*RuntimeContext` (already exists) to both functions** — eliminates `gridVisible`, `hudVisible`, `helpVisible`, `hudDialogVisible`, `labelMode`, `mouseModeEnabled`, `debugEnabled`, `cameraSpeed`, `mouseSensitivity` as separate params; they live on `a.runtime` already.
3. **Collapse the return tuple** — `handleInput` currently returns `(bool, bool, engine.AsteroidDataset, bool, bool, bool)`. With `*RuntimeContext` passed in, only `shouldQuit bool` needs to be returned; all other values are written through the pointer.

#### Result shape
```go
func handleInput(app *App, session *runtimeSession, state *engine.SimulationState) bool
func updateCameraState(session *runtimeSession, runtime *RuntimeContext, dt float32) float32
```

#### Constraints
- `render` package functions (`DrawHUD`, `DrawObjectLabels`) stay as explicit params — the render package cannot import `app`, so no cross-package context struct.
- Do not merge `runtimeSession` and `RuntimeContext` — they have different lifetimes (session is discarded on system reload; RuntimeContext persists).
- Do not add a "god context" that combines engine state, camera, input, and settings — violates SRP and Information Expert (GRASP).

---

### TD-002 — Decouple Sim Tick from Render/Input Loop

**Value**: The simulation engine tick currently runs on the same goroutine as `PollInputEvents` and `BeginDrawing`. Under load, a slow tick starves the input poll — short key taps are missed because they happen entirely between two `PollInputEvents` calls. Decoupling the sim tick lets the render+input loop run at full frame rate regardless of physics load.

**Status**: 📋 Not started
**Priority**: Medium-high — directly causes the input latency symptom (key tap dropped under load); also a prerequisite architectural pattern for F-010 headless split
**Depends on**: TD-001 (cleaner call sites before restructuring the loop). F-023 Phase 1 `DrainQueue()` is a partial mitigation shipped earlier; TD-002 is the full fix.

#### Symptom
When the simulation is under load (many bodies, N-body enabled, dense asteroid belts), keyboard input is unresponsive unless a key is held — short taps are silently dropped.

#### Root cause
`rl.IsKeyPressed(key)` returns true for exactly one frame. If the sim tick blocks the main goroutine for > 1 frame, the key-down→up transition falls entirely between two `PollInputEvents` calls and is never seen. `IsKeyDown` is immune because it reads instantaneous state; `IsKeyPressed` is not.

#### Partial mitigation (F-023 Phase 1 — ship first)
Replace `rl.IsKeyPressed` call sites with `KeyMap.DrainQueue()` → `IsPressed()` (using `rl.GetKeyPressed()` queue draining). Eliminates missed taps as long as the frame eventually runs.

#### Full fix (TD-002)
Move the sim engine tick to its own goroutine. Main goroutine runs only: `PollInputEvents` → `handleInput` → `updateCameraState` → `BeginDrawing` → render snapshot → `EndDrawing`. Sim goroutine runs the physics loop and writes results to the existing snapshot double-buffer. Communication via the existing `AppCmd` channel (commands in) and `WorldSnapshot` atomic swap (results out) — no new concurrency primitives needed.

#### Constraints
- Raylib `PollInputEvents`, `BeginDrawing`, `EndDrawing`, and all `Draw*` calls **must** remain on the main OS thread (GLFW/macOS requirement). Do not move rendering to a goroutine.
- The sim goroutine must not import `internal/client` — preserves architectural boundary from `agent-readme.md` §3.
- Double-buffer safety rules from `lessons-learned-double-buffering.md` apply: sim goroutine writes to back buffer; render goroutine reads from front; atomic swap between ticks.

#### Work items
- [ ] Audit `interactive.go`: identify which calls are render-thread-only vs. sim-only vs. shared
- [ ] Extract sim tick into `runSimLoop(ctx context.Context, app *App, cmds <-chan AppCmd)`
- [ ] Wire snapshot atomic swap: sim writes `back`, main loop swaps and reads `front` each frame
- [ ] Verify race detector passes under `-race` with concurrent sim + render
- [ ] Measure: input latency before/after (target: short taps reliably registered even at 10 fps sim rate)

#### Acceptance criteria
- Single key tap registers correctly even when sim tick takes > 100 ms ✓
- Race detector passes ✓
- Rendering frame rate is not coupled to sim tick rate ✓

## 5. Defects

### DEF-001 — Floating-Point Precision Collapse at Extreme Camera Distances

**Symptom**: When zooming far out from the solar system and looking back, the visual scene begins to contract as if boxed in and shrinking. Worsens with distance.
**Status**: 📋 Not started — short technical exploration needed before full fix is scoped
**Priority**: Medium-high — affects every recording and demo session; fix touches render call sites that F-003/F-004/F-005 also touch. Doing this before Group 2 visual work avoids rewriting those sites twice.
**Depends on**: Nothing blocking; self-contained to the render path

#### Root Cause

Not a Raylib bug or misconfiguration. All object positions are stored as `float32` world-space vectors with SOL at the origin. Raylib's MVP matrix multiplies those positions by a matrix containing the camera's world-space translation. At large camera distances (100,000+ simulation units), the camera-translation components are large `float32` values. Subtracting small object coordinates from a large camera offset causes **catastrophic float32 cancellation** — mantissa bits are lost, objects snap to ghost grid positions, and the scene appears to contract. `CameraFarPlane = 200000.0` and `CameraNearPlane = 0.001` give a ratio of 200,000,000:1, which also destroys depth-buffer precision at range.

#### Fix: Camera-Relative (Floating Origin) Rendering

Before passing positions to the GPU, subtract the camera world position from every object position. The GPU always operates near `(0, 0, 0)` regardless of camera distance from SOL. Physics and protocol layers continue to use full world-space coordinates; only the render call sites change.

#### Exploration Items

- [ ] Reproduce and measure: at what camera distance (simulation units) does the artifact first appear? Establish a threshold.
- [ ] Audit all `DrawSphere`, `DrawModel`, `DrawSphere3D` call sites in the render path — these are the sites that need camera-relative offsets applied
- [ ] Determine whether Raylib's `SetMatrixModelview` or a manual translate before `BeginMode3D` is the cleanest integration point
- [ ] Assess whether `float64` positions are needed in the physics layer (likely yes for long N-body runs, aligns with F-013)
- [ ] Symbolic representation / LOD for extreme zoom-out (separate concern — scope after root cause is fixed)

---

### DEF-002 — Stars Disappear at Outer-Planet Camera Distances

**Symptom**: Sol vanishes from the display when the camera is at Neptune distance (~3 000 sim units) or further. Any star-jump to an outer planet leaves no visible reference point for the star — the solar system appears unanchored.
**Status**: 📋 Not started
**Priority**: High — orientation and navigation break without a visible star at any camera distance

#### Root Cause (diagnosed)

`drawObjectsInstanced` falls through to the `pointRenderingEnabled` path for distant objects. Stars have no dedicated `PointThresholdStar` constant; they fall through to `PointThresholdDefault = 200.0`. At Neptune (~3 000 sim units) Sol is already 15× past that threshold, so it renders as `DrawSphere(pos, pointSize*0.1, color)` → sphere of radius `0.2`. At 3 000 units that is sub-pixel and invisible.

There is no special-case to exempt high-importance or `MaterialEmissive` objects from the point-size collapse.

#### Fix

Two-part: a minimum angular-size floor for stars, and a guaranteed-visible fallback for `MaterialEmissive` bodies regardless of distance.

**Part 1 — Star-specific point threshold and size**
Add `PointThresholdStar` and `PointSizeStar` constants in `engine/constants.go`. Stars should never fall below a minimum rendered size:
```go
PointThresholdStar = math.MaxFloat64  // stars never become points via threshold
PointSizeStar      = float32(10.0)    // minimum apparent radius when forced to point
```

Alternatively: keep the threshold but clamp the point size so it scales with distance to keep apparent angular size constant (pixel-size floor).

**Part 2 — Distance-invariant minimum draw size for emissive bodies**
In `drawObjectsInstanced` and `drawObject`, when `obj.Meta.Material == engine.MaterialEmissive`:
- Bypass the `isPoint` path entirely, OR
- Clamp the draw size: `max(physicalRadius * (drawScale / distance), minPixelRadius)` so the body always subtends at least N screen pixels.

**Part 3 — Add `CategoryStar` to the point-threshold dispatch**
The `if pointRenderingEnabled` block checks `CategoryAsteroid`, `CategoryPlanet`, `CategoryMoon` but has no branch for `CategoryStar`. Add it:
```go
} else if obj.Meta.Category == engine.CategoryStar {
    pointThreshold = engine.PointThresholdStar
    pointSize = engine.PointSizeStar
}
```

#### Acceptance Criteria

- [ ] Sol is visible as a bright disc or large point from Neptune (~3 000 sim units)
- [ ] Sol is visible as a bright point from the edge of the solar system (~10 000 sim units)
- [ ] Stars in multi-star systems (Alpha Centauri) remain visible at similar inter-system distances
- [ ] No regression to Sol rendering at close range (< 200 sim units)

#### Depends on

Nothing blocking; self-contained to `engine/constants.go` and the point-rendering dispatch in `renders.go`.

---

### DEF-003 — Sol Atmosphere Missing

**Symptom**: Sol has no atmosphere glow rendered around it. Other bodies with `AtmosphereThicknessKm` set (e.g. Earth, Venus) display a Fresnel rim glow; Sol does not, despite being the most prominent body in the scene.
**Status**: ✅ Fixed — 2026-04-24
**Priority**: Medium — visual correctness; Sol should have a visible corona/glow comparable to its prominence
**Depends on**: Nothing blocking; `drawAtmosphereGlow` already exists in `renders.go`

#### Root Cause

Two independent bugs both had to be present for Sol's corona to be invisible:

1. **No `PointThresholdStar` constant** — `CategoryStar` fell through to `PointThresholdDefault = 200.0`. At camera distances > 200 su (nearly all normal views), Sol was point-rendered via `DrawSphere` with an early `return`, so `drawAtmosphereGlow` was never reached.

2. **Lambert term inverted for self-luminous bodies** — When Sol *was* within 200 su and sphere-rendered, `drawAtmosphereGlow` was called but the atmosphere shader computed `toLight = normalize(lightPos - fragPos)`. For Sol's glow sphere, `lightPos ≈ Sol's position ≈ fragPos` (the glow sphere is centred *on* Sol), so `toLight ≈ -norm`. That gives `dot(norm, toLight) ≈ -1`, clamped to 0, `litFactor ≈ 0.08` — the corona rendered at ~3% brightness and was effectively invisible.

#### Fix Applied

- `engine/constants.go`: Added `PointThresholdStar = 1e15` (stars never become points) and `PointSizeStar = float32(10.0)`.
- `renders.go`: Added `CategoryStar` branch in the point-threshold dispatch in both `drawObjectsInstanced` and `drawObject`.
- `lighting.go` (`atmoFS` + `atmosphereState`): Added `uniform int selfLuminous` to the GLSL fragment shader. When `selfLuminous == 1`, `litFactor = 1.0` (full intensity, no Lambert). Added `locSelfLuminous int32` to `atmosphereState` and cached the location in `load()`.
- `renders.go` (`drawAtmosphereGlow`): Uploads `selfLuminous` int uniform using the bit-reinterpretation pattern (`*(*float32)(unsafe.Pointer(&selfLum))`) matching the existing `hasNightTexture` pattern.

#### Investigation Items

- [x] Check whether `solar_system.json` sets `atmosphere.thickness_km` for Sol — **already set** (`2000000 km`, `[255,160,50,180]`)
- [x] Check whether `drawAtmosphereGlow` is skipped for `SelfLuminous` or `MaterialEmissive` bodies — **root cause confirmed above**
- [x] Determine the correct atmosphere color hint and thickness for a solar corona approximation — **existing JSON data is correct**
- [x] Verify the atmosphere glow renders at Sol's scale — **glow scale math is correct; `atmoCap = 0.60` bounds Sol's corona to 1.6× radius**

#### Acceptance Criteria

- [x] Sol displays a visible rim/corona glow at normal viewing distances
- [x] Glow scales appropriately with Sol's radius
- [x] No regression to Earth/Venus/other atmospheric body rendering

---

### DEF-004 — Objects Clip or Disappear at a Specific Camera Distance

**Symptom**: At a specific camera distance, objects (planets, moons, or other bodies) abruptly clip through the camera plane or vanish entirely, even when they should be clearly in view. The threshold distance at which this occurs has not been precisely measured.
**Status**: ✅ Investigated — 2026-04-24 — largely resolved; remaining visual pop tracked under DEF-002
**Priority**: Medium-high — affects navigation and usability; may share root cause with DEF-001 (floating-origin precision collapse) or may be a near-plane clipping issue
**Depends on**: Nothing blocking; may be resolved as a side-effect of DEF-001 fix

#### Investigation Findings (2026-04-24)

- **Float32 catastrophic cancellation**: **ALREADY FIXED** — `interactive.go` implements floating-origin camera (`camera.Position = rl.Vector3{}`, all object positions shifted by `-cameraPos`). DEF-001 mitigation fully covers this case.
- **Far-plane culling** (`CameraFarPlane = 200000`): **Not an issue** — Neptune (maximum distance) is only ~1505 sim units; far plane is 133× beyond that.
- **Near-plane clipping** (`CameraNearPlane = 0.001`): **Not an issue in practice** — very small bodies (`PhysicalRadius < 0.5 su`) are always point-rendered before the sphere draw path is reached, so the near plane is never relevant for them.
- **Point-rendering visual pop**: Bodies switch from sphere to `DrawSphere` at distance thresholds (200/300/500 su for default/moon/planet). This abrupt switch is the "specific distance" pop the symptom describes. This behaviour is already tracked and planned under DEF-002.

**Conclusion**: DEF-004 is largely resolved by the existing floating-origin fix. The remaining visual artefact (point/sphere pop at thresholds) is the scope of DEF-002 and should be addressed there.

#### Investigation Items

- [x] Reproduce and record the exact camera distance at which clipping or disappearance first occurs — **threshold is the point-render threshold; see DEF-002**
- [x] Determine whether the cause is near-plane, far-plane, or float precision — **float precision already fixed by DEF-001 floating-origin camera**
- [x] Check whether the threshold scales with object size — **yes: different thresholds per category (200/300/500 su)**
- [x] Assess overlap with DEF-001 — **fully covered; DEF-004 closed**

#### Acceptance Criteria

- [x] No body that is geometrically in the camera frustum clips or vanishes at any zoom level reachable via normal navigation — **satisfied by floating-origin fix**
- [ ] Near-plane value is tuned so close-approach viewing of small moons does not clip geometry — **deferred; no reproduction case identified**

---

Prioritized by dependency order and user-visible value. Items lower in the list generally depend on or benefit from items above them.

---

### F-001 — Camera Collision Prevention

**Value**: Camera should never clip inside a body. Currently possible when zooming in at tracking range or jumping to very small objects.
**Status**: 📋 Not started
**Priority**: High — safety constraint, no deps, contained change
**Depends on**: Nothing

#### Work Items

- [ ] In `UpdateTracking`, clamp `TrackDistance` so the camera surface stays outside `target.Meta.PhysicalRadius + epsilon`
- [ ] In `UpdateJump`, detect if landing position would be inside any rendered object and push the camera out to the object surface
- [ ] Add a general camera-vs-object exclusion check in `updateCameraState` for free-fly mode (prevent flying through spheres)

---

### F-002 — REPL: `track <object>` and `track stop`

**Value**: Lets scripts and live REPL sessions lock the camera to an object or release tracking without touching the keyboard.
**Status**: ✅ Complete — 2026-04-09
**Priority**: High — extends existing tracking system; aligns with REPL Expansion Phase C (Camera)
**Depends on**: Existing `CameraTrackCmd`, REPL command dispatch

#### Work Items

- [x] Add `track <name>` command to REPL: issues `CameraTrackCmd{Name: name}`
- [x] Add `track stop`: issues `CameraTrackCmd{Name: ""}` to enter free-fly
- [x] Wire TAB completion for body names on `track` (stop + bodies; `track s` → `track stop`)
- [x] Proto and gRPC transport already wired via `SetCameraTrack` RPC

---

### F-003 — Texture/Bitmap Rendering for Planets and Moons

**Value**: Replace solid-color spheres with photorealistic diffuse textures. Toggleable so low-resource environments can use the color fallback.
**Status**: ✅ Done
**Priority**: Medium-high — high visual impact; material system already has a `diffuse` material type and texture asset paths in `data/assets/textures.json`
**Depends on**: Nothing blocking; texture assets already present for key bodies

#### Work Items

- [x] Load and bind diffuse textures in the Raylib material pipeline — `loadTexture`/`getModel` with UV correction for par_shapes transposed coords, V-flip, U-mirror, and pole rotation
- [x] Textured `DrawModel` path with axial tilt and real-time spin from `RotationPeriod`; fallback to `DrawSphereEx` when texture absent
- [x] CLI flag `--no-textures` (default: textures on); wired through `app/config.go` → `render.New(noTextures)`
- [x] Graceful fallback to solid color when texture file is missing or flag is set
- [ ] Normal-map and specular-map binding as secondary pass (deferred to F-017)
- [ ] Night-lights and cloud-layer compositing for Earth (deferred to F-017)

---

### F-004 — Milky Way Skysphere Background

**Value**: Replace the blank black background with a Milky Way equirectangular panorama that rotates with camera orientation (no translational parallax at solar-system scale).
**Status**: ✅ Complete — 2026-04-26
**Priority**: Medium — high visual quality improvement; independent of simulation state
**Depends on**: Camera forward vector (already available via floating-origin camera)

#### Implementation

- Equirectangular skysphere: small-radius unit sphere (radius 5.0 su) drawn first inside `BeginMode3D` with depth test and depth-writes disabled, so it is always behind all scene geometry regardless of its geometric position.
- **Critical**: `skyRadius` must be small (1–10 su). At 180,000 su the float32 clip-space precision ratio (~1.8×10⁸) causes OpenGL to silently discard all triangles during near-plane clipping. See lessons-learned #31.
- Winding fix: negative X scale (`MatrixScale(-skyRadius, skyRadius, skyRadius)`) makes inner faces front-facing; no need to toggle backface culling state.
- UV fix for inside-sphere viewing with negative-X winding flip: `new_u = 1 - old_v, new_v = 1 - old_u` (U-flip compensates for the mirroring from negative X scale).
- Texture: `data/assets/textures/starfield_8k.jpg` — Solar System Scope `8k_stars_milky_way.jpg` (CC BY 4.0), committed to repo.
- No-texture mode (`--no-textures`) skips the sky gracefully; black background if texture file is absent.
- `Renderer.Close()` clears the diffuse slot before unloading the sky model to avoid double-freeing the texture (also in `textureCache`).
- Sky uses Raylib's default flat shader (always lit at full brightness, independent of scene lighting).

#### Work Items

- [x] Draw as a skysphere pass before the 3D scene, using camera orientation only (floating-origin camera is always at (0,0,0) — sphere naturally rotates with camera, never translates)
- [x] Load Milky Way equirectangular texture; fall back silently to black if absent
- [x] UV correction for inside-sphere viewing with winding flip
- [x] Disable depth test + depth mask around sky draw
- [x] Negative-X winding flip to avoid unreliable DisableBackfaceCulling interaction with DrawModel
- [x] Small sky radius (5.0 su) to avoid float32 clip-space precision discard at large radii
- [ ] Optional: vary star brightness/color procedurally to supplement the texture (deferred)

---

### F-005 — Physical Lighting from Stars

**Value**: Drive scene lighting from simulated star properties (luminosity derived from mass/type/color) rather than a fixed point light.
**Status**: ✅ Complete — 2026-04-21
**Priority**: Medium — depends on star data already in `solar_system.json`; requires shader or multi-light Raylib work
**Depends on**: F-003 (textures must be bound before physically-based lighting is meaningful); star `mass`, `color`, and `radius` fields in `ObjectMeta`

#### Work Items

- [x] Compute luminosity from star mass using a mass-luminosity approximation (L ∝ M^3.5 for main sequence) — simplified: SolarLuminosity field in JSON; defaults to 1.0
- [x] Map luminosity + color to Raylib light intensity and tint — GLSL 330 Phong shader, warm-white fallback
- [x] Support multiple stars in the scene (up to 4 via `maxLights`; multi-star systems supported)
- [x] Inverse-square falloff per-object from each star's position — `lightScale / dist²` in fragment shader
- [x] CLI flag `--no-lighting` (default: physical lighting on); wired through `app/config.go` → `render.New`

---

### F-006 — XYZ Keyboard Navigation + Mouse Facing

> **Note**: The keyboard binding architecture for this feature is now specified in **F-023**. Implement F-023 Phase 1 first; XYZ navigation bindings are added as part of F-023 Phase 3 (mouse delta input).

**Value**: Provide an explicit 6-DOF free-fly mode (X/Y/Z translate via keyboard, yaw/pitch via mouse) as an alternative to the current relative WASD scheme. Useful for precise positioning and scripted camera work.
**Status**: 📋 Not started — deferred until F-023 Phase 1 ships
**Priority**: Medium
**Depends on**: F-023 Phase 1 (keyboard binding system); F-007 scope subsumed by F-023

#### Work Items

- [ ] Add absolute-axis translate inputs (default: arrow keys + PgUp/PgDn, or configurable)
- [ ] Expose a toggle between current relative mode and absolute XYZ mode
- [ ] Decouple mouse look from movement mode so mouse always controls facing regardless of translate mode

---

### F-007 — User-Configurable Key Bindings

> **Note**: This feature is fully specified and superseded by **F-023**. F-023 delivers everything F-007 described (action enum, JSON config, override merge, conflict detection, REPL help overlay) plus hardware profiles and hot-reload. Implement F-023 instead of F-007.

**Value**: Allow players to remap any action to a different key without recompiling.
**Status**: 📋 Superseded by F-023
**Priority**: N/A — see F-023
**Depends on**: N/A

---

### F-008 — Artifact Object Type

**Value**: Introduce a new object category for non-natural, non-spherical objects: asteroid shapes (polyhedra), satellites, space probes, comets, spacecraft. Enables richer scene content without forcing everything into a sphere.
**Status**: 📋 Not started
**Priority**: Medium-low — architectural; requires schema, loader, and renderer changes
**Depends on**: F-003 (texture pipeline) for surface detail; possibly F-005 (lighting) for accurate material response

#### Work Items

- [ ] Add `CategoryArtifact` to `engine/object.go` object category enum
- [ ] Extend the JSON schema to support `"type": "artifact"` with a `mesh` field pointing to an OBJ/GLB asset
- [ ] Implement mesh loading in the Raylib renderer (Raylib has `LoadModel`/`DrawModel`)
- [ ] Comet type: add dust-tail and ion-tail particle emitters driven by distance-to-star
- [ ] Bounding-sphere approximation for camera collision (F-001) and frustum culling

---

### F-009 — Object-Object Collision / Proximity Detection

**Value**: Fire events when two simulation objects come within a configurable threshold of each other. Foundation for impact alerts, gravitational capture detection, and gameplay events.
**Status**: 📋 Not started
**Priority**: Low — foundational event plumbing; highest value once Artifact objects (F-008) are in flight
**Depends on**: Existing event queue (`internal/server/eventqueue`); F-008 for non-trivial shape detection

#### Work Items

- [ ] Add a `ProximityEvent` type to the event envelope
- [ ] Implement a broad-phase sweep (AABBs or spatial grid already in `internal/client/go/raylib/spatial/`) to cull distant pairs
- [ ] Narrow-phase: sphere-sphere distance check; extend to bounding-sphere vs mesh for artifacts
- [ ] Per-pair configurable threshold stored in `ObjectMeta` or a separate proximity config
- [ ] Publish `ProximityEvent` through `protocol.Broadcaster` so gRPC clients and the REPL can observe collisions

---

### F-010 — Multi-Machine Architecture (Option B: headless server + network clients)

**Value**: Run the simulator and gRPC server on a remote/headless machine; connect one or more Raylib renderer clients over the network. Each client has an independent camera POV. Simulation commands are admin-only via a server-side REPL.
**Status**: 📋 Not started
**Priority**: High architectural — gates multi-client, multi-machine, and federated compute goals
**Depends on**: F-011 (IAAM — need identity before multi-client commands make sense)

#### Decisions (locked)

| Topic | Decision |
|-------|----------|
| Simulation commands | Admin-only; exposed via a server-side REPL (not reachable by renderer clients) |
| Client camera | Fully local to each renderer; never sent to or from the server |
| Snapshot delivery | `WorldService.StreamSnapshot` already exists; clients subscribe over gRPC |
| Snapshot rate vs. render rate | Server produces at physics Hz; clients render independently using latest received snapshot |

#### Work Items

**Group A — `cmd/space-sim-server` (headless)**
- [ ] New entrypoint: no Raylib imports; starts `World`, starts gRPC server, blocks
- [ ] Server-side admin REPL (stdin loop or dedicated port) for simulation commands (setspeed, pause, load, etc.)
- [ ] Expose existing `SimulationService`, `WorldService`, and other handlers unchanged

**Group B — `cmd/space-sim-client` (Raylib renderer)**
- [ ] Subscribe to `WorldService.StreamSnapshot` over gRPC; store latest snapshot in an `atomic.Pointer`
- [ ] Render loop reads from atomic pointer instead of calling `sim.Snapshot()` locally
- [ ] All simulation command RPCs removed from client — camera, nav, window controls remain local

**Bandwidth mitigations**
- [ ] **POV frustum filtering**: client sends its current view frustum to the server each frame; server includes only objects within (or near) that frustum in the snapshot. Reuses existing `internal/client/go/raylib/spatial/` frustum logic, moved server-side.
- [ ] **Client-side interpolation**: server streams at a reduced rate (e.g. 20 Hz); client interpolates object positions between received snapshots for smooth 60 Hz rendering.
- [ ] **Delta compression**: server sends only changed fields since the last acknowledged snapshot per client. Requires per-client sequence tracking.
- [ ] **LOD by distance**: objects beyond a configurable distance threshold are omitted or sent at reduced precision.

---

### F-011 — IAAM: Identity, Access, Authentication, and Management

**Value**: Establish who each connected client is, what they are allowed to do, and how that is enforced at the transport boundary. Required before multi-client commands have safe semantics.
**Status**: 📋 Not started
**Priority**: High — blocks F-010 multi-client safety
**Depends on**: F-010 (deployment model shapes token delivery); separate web frontend + auth backend (external component)

#### Decisions (locked)

| Topic | Decision |
|-------|----------|
| Token issuance | Separate web frontend + backend service; handles registration, login, token issuance. Opens gRPC connection to `space-sim-server` to validate token at connect time. |
| Identity at connect | Bearer token presented in gRPC metadata; server validates with auth backend on connection; identity attached to `context.Context` for lifetime of connection |
| Roles | `admin`, `moderator`, `team_lead`, `user` (see permission table below) |
| Admin channel | Admin commands never exposed via network to non-admin roles; enforced in gRPC interceptor |
| Auth mechanism | Bearer token over TLS; evaluated against ConnectRPC interceptors |

#### Role × Permission Table (draft)

| Command / Feature | admin | moderator | team_lead | user |
|---|---|---|---|---|
| Stream snapshot | ✅ | ✅ | ✅ | ✅ |
| Camera / nav / window (local) | ✅ | ✅ | ✅ | ✅ |
| Pause / resume simulation | ✅ | ✅ | ✅ | ❌ |
| Load system | ✅ | ✅ | ❌ | ❌ |
| Set speed | ✅ | ✅ | ✅ | ❌ |
| Set dataset (asteroid density) | ✅ | ✅ | ✅ | ❌ |
| Shutdown server | ✅ | ❌ | ❌ | ❌ |
| User / role management | ✅ | ❌ | ❌ | ❌ |

*Table is a starting draft — refine during design phase.*

#### Work Items

- [ ] Define role enum and permission table in `internal/auth/` package
- [ ] Implement authentication interceptor in `internal/transport/grpc/interceptors.go`; validate token with auth backend; attach identity + role to `context.Context`
- [ ] Enforce role checks in each handler per the permission table above
- [ ] Token revocation: server calls auth backend to check revocation on each connection (or uses short-lived tokens with refresh)
- [ ] Connection audit log: identity, role, commands issued, connect/disconnect timestamps
- [ ] Design external auth backend interface (separate repo or service — out of scope for this repo except for the validation call)
- [ ] TLS configuration for all gRPC connections (client and server)

---

### F-012 — Federated Compute: Collaborative Simulation Offload

**Value**: Distribute physics computation across multiple machines so simulation scale (object count, fidelity) is not bounded by a single server's CPU. Primary goal is simulation accuracy and scale; fault tolerance is secondary. User count may be capped to protect simulation performance.
**Status**: 📋 Not started — exploratory
**Priority**: Low — long-term; F-010 and F-011 must be stable first
**Depends on**: F-010 (stable server/client split), F-011 (node identity and trust), F-013 (N-body barycenter — accuracy requirement shapes partition strategy), existing `internal/server/pool/distributed/` stub

#### Decisions (locked)

| Topic | Decision |
|-------|----------|
| Primary goal | Simulation accuracy and scale first; redundancy/fault tolerance second |
| User cap | Acceptable to limit concurrent clients to protect simulation fidelity |
| Trust model | Compute nodes treated as trusted peers (internal/private network) |

#### Open Questions (to be explored)

- **Partition strategy**: spatial (octree regions), object-type (planets on one node, belts on another), or load-balanced? N-body gravity complicates pure spatial partitions — cross-boundary forces must be exchanged each tick.
- **Reconciliation**: point-mass approximation for remote partitions vs. full cross-partition force exchange? Accuracy requirement may require full exchange for planet-scale bodies.
- **Clock synchronization**: coordinator epoch (simplest) or vector clock? All nodes must agree on simulation time for a coherent `WorldSnapshot`.
- **Failure handling**: partition freeze, migration to coordinator, or simplified fallback physics?
- **Entry point**: `internal/server/pool/distributed/` stub is the natural starting point.

---

### F-013 — N-Body Barycenter Integration

**Value**: Replace the current single-parent Keplerian approximation with true N-body gravitational simulation. Positions bodies at their mutual barycenter rather than a fixed center. Required for accurate multi-star systems (Alpha Centauri A/B), binary planets, and long-term orbital stability.
**Status**: 📋 Not started — plan complete, ready to implement
**Priority**: Medium-high — accuracy is a stated project goal; also a prerequisite for F-012 accuracy claims
**Depends on**: Nothing blocking; self-contained change to `internal/sim/engine/physics.go`. Coordinate with F-012 design (partition strategy depends on how forces are computed).
**Plan**: [docs/wip/f013-nbody-plan.md](f013-nbody-plan.md)

#### Context

Current physics (`updateObject` in `physics.go`) uses Keplerian two-body mechanics: each object orbits a fixed parent at a fixed center. This is efficient but ignores mutual gravitational attraction between all bodies. Barycenter motion and multi-body perturbations are not modeled.

N-body integration replaces this with a force-sum approach: each body computes gravitational attraction from every other body each tick, then integrates velocity and position. Accuracy scales with integration step size and method.

#### Decisions (open — to be resolved during design)

| Topic | Options |
|-------|--------|
| Integrator | Leapfrog (symplectic, good energy conservation) vs. RK4 (higher accuracy, more expensive) vs. Verlet |
| Force computation | Brute-force O(N²) (fine for solar-system scale, ~100 bodies) vs. Barnes-Hut tree (O(N log N), required for belt-scale N-body) |
| Backward compatibility | Keplerian initial conditions can seed N-body starting state; existing JSON schema unchanged |
| Belt objects | Full N-body for belt particles is prohibitive (tens of thousands); likely keep Keplerian for belts and apply N-body only to named bodies (planets, moons, stars, dwarf planets) |
| Barycenter output | Publish barycenter position per multi-body group in `SimulationState` for use by renderer and camera |

#### Work Items

- [ ] Implement force accumulator: for each named body, sum gravitational attraction from all other named bodies each tick
- [ ] Replace `updateObject` position integration with leapfrog or Verlet for named bodies; keep Keplerian for belt particles
- [ ] Add barycenter computation per gravitationally-bound group; store in `SimulationState`
- [ ] Update `ObjectMeta` / `AnimState` to carry velocity as a first-class field (already partially exists)
- [ ] Validate: Earth-Moon barycenter should be inside the Sun; Alpha Centauri A/B should orbit their mutual barycenter
- [ ] Performance benchmark: O(N²) force sum at 100 named bodies at 60 Hz on M1; establish whether Barnes-Hut is needed
- [ ] Update scene camera's star-tracking logic to track barycenter for multi-star systems

---

### F-014 — Nearby Systems Expansion Backlog

**Value**: Expand `data/systems/` beyond SOL and Alpha Centauri with nearby stellar systems that have enough confirmed or reasonably modelable structure to fit the current simulator. This backlog is ordered by practical fit for the current engine, not only raw distance from SOL.
**Status**: 📋 Not started
**Priority**: Medium — content expansion with direct user-facing value; best tackled while the current JSON loading path is stable
**Depends on**: Existing system JSON flow in `data/systems/`; current parent-child orbital model remains acceptable for single-star systems and approximate multi-star hierarchies

#### Recommended System Order

1. **Epsilon Eridani**
	Best immediate addition after Alpha Centauri. Nearby, well-studied, and compatible with the current simulator because it has a dominant primary star, at least one widely accepted giant planet, and debris-belt structure that maps cleanly onto the existing body + feature JSON model.
2. **Epsilon Indi**
	Strong follow-up because it has a confirmed giant planet around the primary plus the known brown-dwarf companion pair. The system is more complex than a simple single-star system, but still workable under the current approximate parent-child hierarchy.
3. **GJ 1061**
	Good fit for the current engine because it is a compact red-dwarf planetary system with multiple close-in planets and no major need for true barycentric motion.
4. **Luyten's Star**
	Useful nearby addition with a small planetary system that can be represented with the existing star-plus-planets JSON approach and does not require unusual renderer or physics support.
5. **Teegarden's Star**
	Another good compact system candidate. It offers a simple star with close planets and low modeling overhead under the current orbital assumptions.
6. **Barnard's Star**
	Extremely close and high-interest, but lower in execution priority because the planet picture has had more historical uncertainty. Add only if the project is comfortable modeling a system with more tentative planet data.
7. **Wolf 359**
	Very nearby and potentially interesting, but currently less compelling than the systems above because the planetary data is less settled and the modeling payoff is lower.
8. **Sirius**
	Stellar data is strong and the binary is scientifically important, but it is a weaker near-term content addition because there is no equally compelling confirmed planet set. Better as a multi-star showcase than a planet-system showcase.
9. **Luhman 16**
	Very close and scientifically interesting, but mainly a brown-dwarf binary. It is likely modelable visually, yet less aligned with the simulator's current star-and-planets content value than the systems above.

#### Modeling Notes

- Prefer systems with one dominant primary star and confirmed planets first; they fit the current orbital engine with minimal approximation.
- Multi-star systems remain acceptable when modeled hierarchically, but should be treated as approximations until F-013 lands.
- Debris belts and compact close-in planetary systems are good near-term targets because they reuse the existing JSON schema and rendering behavior effectively.
- When planet detections are still disputed, either defer the system or explicitly mark candidate planets as provisional in the system design notes before adding JSON.

#### Work Items

- [ ] Create a per-system data brief for Epsilon Eridani, Epsilon Indi, GJ 1061, Luyten's Star, and Teegarden's Star using confirmed-first data
- [ ] Decide whether Barnard's Star and Wolf 359 should be excluded until their planet sets are more stable
- [ ] Decide whether Sirius and Luhman 16 belong in the same near-term content phase or in a separate multi-star showcase phase
- [ ] Define a lightweight standard for provisional vs. confirmed exoplanet entries before adding uncertain systems
- [ ] Add new `data/systems/*.json` files in backlog order, validating each one with the simulator before moving to the next

---

---

### F-015 — Epoch-Accurate Initial Mean Anomaly ("Start From Today")

**Value**: Replace `"initial_mean_anomaly": "random"` on all solar system bodies with computed J2000 mean anomalies advanced to the actual launch date so the simulator opens with positions matching the real sky, making it useful as a reference and teaching tool alongside the existing free-running mode.
**Status**: 📋 Not started
**Priority**: Medium — high fidelity value; no engine change required; pure data work

#### Background

Every solar system body currently uses `"initial_mean_anomaly": "random"`, which is honest — the simulation is not a planetarium. To start from a real date the mean anomaly `M` for each body must be computed as:

```
M = M0 + n × (t - t0)
```

Where `M0` is the mean anomaly at the J2000 epoch (2000-01-01 12:00 TT), `n = 360 / orbital_period_days` is the mean motion in degrees/day, and `t - t0` is elapsed days since J2000. This can be precomputed for any target date and written directly into each body's `initial_mean_anomaly` field as a numeric value (degrees or radians, matching the existing unit convention).

Exoplanet systems are excluded — orbital phases are not observationally constrained to a known epoch for most confirmed exoplanets.

#### Scope

- Solar system only: Sol, all 8 planets, all dwarf planets, all named moons.
- Ring systems and belt features do not have a meaningful orbital phase.
- The REPL or a CLI flag should eventually allow overriding the epoch date at runtime without needing to regenerate the data file.

#### Work Items

- [ ] Confirm unit convention for `initial_mean_anomaly` in the existing loader (`degrees` vs `radians`)
- [ ] Write a script to compute and write J2000-epoch mean anomalies for each named solar system body using published orbital elements (JPL Horizons or equivalent)
- [ ] Add a `epoch` top-level field to `solar_system.json` (e.g. `"epoch": "2000-01-01T12:00:00TT"`) so the loader knows what the anomaly values are relative to
- [ ] Add optional `--epoch YYYY-MM-DD` flag to the REPL / direct binary to substitute a different start date without rebuilding the JSON
- [ ] Validate by checking Earth, Mars, and Jupiter positions against a known ephemeris for the chosen date

---

### F-016 — Wire Rendering Data Pipeline (Schema → Engine → Renderer)

**Value**: The JSON schema now carries `rendering.texture_image`, `rendering.fallback_color`, `rendering.material`, `physical.albedo`, `atmosphere`, and `luminosity` for every body. None of these fields except `color` and `material` flow past the loader — `ObjectMetadata` has no albedo, texture path, or luminosity fields, and the renderer only reads `Meta.Color` and `Meta.Material`. This item creates the plumbing that F-003 (texture rendering) and F-005 (physical star lighting) both depend on.
**Status**: 📋 Not started
**Priority**: High — blocking prerequisite for F-003 and F-005
**Depends on**: Nothing; schema is already complete

#### Work Items

- [ ] Add `TexturePath`, `Albedo float32`, `SelfLuminous bool`, `SolarLuminosity float32`, `SurfaceTemperatureK float32`, and `EmissionColor engine.Color` fields to `ObjectMetadata` in `internal/sim/engine/object.go`
- [ ] Update `loader.go` `createBodyFromConfig` to populate all new `ObjectMetadata` fields from `RenderingConfig`, `PhysicalConfig`, and the `luminosity` block (add `LuminosityConfig` struct to `schema.go` first)
- [ ] Update `applyOverrides` in `loader.go` to merge the new fields correctly
- [ ] Add `"diffuse_thermal"` as a recognized material string in `parseBodyMaterial` in `loader.go` (currently falls through to `MaterialDiffuse`; give it its own enum value in `engine/object.go`)
- [ ] Thread texture path and albedo through to the Raylib renderer's `drawObject` / `drawObjectsInstanced` so F-003 can bind the texture without structural changes
- [ ] Thread luminosity fields to the renderer's light-source pass so F-005 can use star position + luminosity without structural changes
- [ ] Add `atmosphere` fields (`ThicknessKm`, `ColorHint`, `CloudCoverage`) to `ObjectMetadata` and populate from loader (needed for F-003 atmosphere overlay)
- [ ] `make test` must stay green; add a loader test asserting Sol's `TexturePath` and `SolarLuminosity` are populated after loading `solar_system.json`

---

### F-017 — Realistic Lighting (Shadows, Atmosphere, Bloom, PBR)

**Value**: Upgrade the renderer from Phong point-light shading (F-005) to physically plausible lighting with inter-body shadows, atmospheric glow, and post-process effects. Makes the sim visually convincing for eclipses, transits, and close-approach scenarios.
**Status**: 📋 Not started
**Priority**: Low — high complexity; depends on F-003 (texture binding) and F-005 (physical star lights) being complete first
**Depends on**: F-016 (data pipeline), F-003 (textures), F-005 (star lights)

#### Feature Checklist

| Feature | Technique | Notes |
|---|---|---|
| **Shadows from blocking bodies** | Shadow map per star — render scene depth from star POV into a depth framebuffer; sample in main fragment shader | Raylib requires a custom GLSL shader pair; no built-in support |
| **Eclipse / umbra-penumbra** | Per-fragment ray–sphere intersection against all bodies between fragment and star | Can be approximated as soft shadow kernel in the shadow map, or computed analytically in shader |
| **Atmospheric scattering** (limb glow, haze) | Rayleigh/Mie scattering pass or screen-space atmosphere shader driven by `atmosphere.color_hint` and `thickness_km` | `AtmosphereThicknessKm` and `AtmosphereColorHint` already flow through `ObjectMetadata` after F-016 |
| **Bloom / star glow corona** | Post-process blur pass on the render texture; extract bright pixels above threshold, Gaussian blur, additive composite | Render target already exists; needs a second pass stage |
| **PBR (specular, roughness, metallic response)** | PBR shader with bound roughness/metallic maps; `albedo` from `ObjectMetadata.Albedo` | Replaces Phong entirely; roughness map sourced from `rendering.specular_map` field |
| **Night-side emission** (Earth city lights) | Blend a night-lights texture where `dot(N, L) < 0`; controlled by a second `texture_image` slot | Requires a second texture path field (e.g. `rendering.night_texture_image`) in schema and `ObjectMetadata` |

#### Work Items

- [ ] Design and implement a shadow-map render pass: render depth from each star's position into a `rl.RenderTexture2D`, pass the depth texture and light-space matrix as shader uniforms to the main pass
- [ ] Write a GLSL fragment shader (`body.frag`) that samples the shadow map and computes Phong + shadow attenuation; replace `DrawSphereEx` with `DrawMesh` + custom shader
- [ ] Add a post-process pass after `EndTextureMode`: extract luminance above a threshold, Gaussian blur the bright layer, additively composite back onto the scene
- [x] Implement Rayleigh scattering as a screen-space approximation: for each atmosphere-bearing body, draw an additive-blended disc slightly larger than the sphere, color from `AtmosphereColorHint`, scaled by `AtmosphereThicknessKm` (**done — F-017a; `drawAtmosphereGlow` in renders.go**)
- [x] Add `NightTexturePath string` to `ObjectMetadata` and `rendering.night_texture_image` to JSON schema; blend night texture where the star is below the horizon (**done — F-017a; Earth wired to earth_nightmap.jpg**)
- [ ] Expose `--no-shadows`, `--no-bloom`, `--no-atmosphere` CLI flags; persist in `app.json`

---

### F-018 — Object Annotations HUD

**Value**: Overlay additional spatial information directly in the 3D view — outlines around bodies, rotational-axis lines, orbital path ellipses with direction-of-travel markers, and automatic label activation — without cluttering the main HUD or requiring a separate debug mode.
**Status**: 📋 Not started
**Priority**: High — high user-visible value; builds on existing label and render infrastructure; no engine changes required
**Depends on**: F-003 (texture/render pipeline established), existing `LabelMode` and `drawObjectLabels` in renderer

#### Feature Scope

| Annotation | Description |
|---|---|
| **Body outlines** | Thin rim/silhouette line drawn around each visible planet, moon, star, dwarf planet, and belt object. Scales with apparent size; disabled for point-rendered objects. |
| **Rotational axis** | Line segment through each body's centre aligned to the axial-tilt vector; length proportional to body radius (e.g. 3×). Visually shows obliquity and retrograde/prograde poles. |
| **Orbital path** | Ellipse (or circle for near-circular orbits) traced in the orbital plane of each body. Computed from `SemiMajorAxis`, `Eccentricity`, and parent position at draw time. |
| **Direction of travel** | Small arrowhead or tick marks on the orbital path tangent to the velocity vector at the current position, indicating prograde direction. |
| **Auto-labels** | When annotations mode is activated, `LabelMode` is forced to `LabelModeAll`; restored to previous mode when annotations are toggled off. |

#### Activation

- Toggle key (suggested: `A` — not currently bound).
- Annotations mode stored in `RuntimeContext` and persisted to `app.json` alongside other performance options.
- REPL command: `annotations on` / `annotations off` (or `annotations toggle`).

#### Work Items

- [ ] Add `AnnotationsEnabled bool` to `RuntimeContext` and `PerformanceConfig`; persist to `app.json`
- [ ] Add `A` key toggle in `input.go`; sync to `runtime.AnnotationsEnabled`; force `LabelMode` on/off
- [ ] Add REPL `annotations` command and TAB completion
- [ ] Renderer: draw body outlines as a slightly enlarged wireframe sphere pass (additive blend, thin line width) or rim-highlight shader; skip for `isPoint` objects
- [ ] Renderer: draw rotational-axis line segment using `rl.DrawLine3D` from `buildModelTransform` tilt vector; scale to `3 × PhysicalRadius`
- [ ] Renderer: compute and draw orbital ellipse as a polyline in world space; 64–128 segments; use parent body position as focus
- [ ] Renderer: place direction-of-travel arrowhead at current-position tangent on the orbital ellipse
- [ ] Add `annotations` entry to help screen in `drawHelpScreen`
- [ ] Acceptance: annotations visible for Sol, Earth, Moon at default camera position; no frame-rate regression > 5% with annotations on

---

### F-019 — Run Scripts from UI

**Value**: Allow the user to browse and execute REPL script files from within the running application without switching to a terminal, lowering the barrier to replaying scripted camera tours and simulation sequences.
**Status**: 📋 Not started
**Priority**: Medium — useful for demos and automation; builds directly on the existing REPL and `scripts/` directory
**Depends on**: Existing REPL command dispatch (`internal/client/repl/`), existing `scripts/` directory layout, `SelectionMode` dialog infrastructure in the UI

#### Feature Scope

- A script-browser dialog (reuses the existing selection-dialog UX pattern) lists `.txt` script files found in the `scripts/` directory at runtime.
- The user navigates the list with arrow keys and presses Enter to execute the selected script.
- Script execution feeds each line through the existing REPL command parser; output is shown in the console / HUD banner.
- A `Cmd+R` or dedicated key opens the dialog; Esc cancels without executing.
- The REPL `run <path>` command continues to work for programmatic use.

#### Work Items

- [ ] Scan `scripts/` directory at session start and on dialog open; build a sorted list of `*.txt` files
- [ ] Add `SelectionModeScripts` to the `SelectionMode` enum in `ui/input.go`
- [ ] Add script-browser dialog render path in `renders.go` (mirrors the system-selector dialog pattern)
- [ ] Wire key binding to open the dialog (`Cmd+R` suggested — not currently bound to an interactive action)
- [ ] On confirm, read the selected file line-by-line and dispatch each line through the REPL command parser via `AppCmd`
- [ ] Surface execution progress in the HUD welcome banner ("Running: solar-tour.txt…")
- [ ] Add `run <path>` REPL command if not already present; TAB-complete script names from `scripts/`
- [ ] Add script browser entry to help screen in `drawHelpScreen`
- [ ] Acceptance: user can open dialog, select `solar-tour.txt`, and the camera tour executes without terminal interaction

---

### F-020 — Multi-Client gRPC Session Layer

**Value**: Allow up to 100 concurrent REPL clients to connect to a single `space-sim-grpc` process. Each client has a stable session identity (name, role, color, UUID), and the server tracks all sessions in a registry. Conflict resolution policy defined. IAAM integration reserved as a future slot.
**Status**: 📋 Not started
**Priority**: High — foundational for all multiplayer features; F-021 through F-024 depend on it
**Depends on**: Phase 6 gRPC transport (complete)
**Spec**: [f020-multi-client-spec.md](f020-multi-client-spec.md)

#### Phases
- Phase 1: Session registry + identity + proto (`SessionService` RPCs, color palette, `list sessions` REPL)
- Phase 2: Position and POV streaming (`WorldSnapshot` carries client sessions; marker hook)
- Phase 3: Admin controls + conflict policy enforcement
- Phase 4: IAAM slot (reserved; no work until F-011)

---

### F-021 — Client Physical Marker

**Value**: Give every connected client session a visible physical presence in the simulated world. Three-phase escalation: blinking sphere → IQM model → full textured model. Correct scale (human-sized ship, not planet-sized). LOD rules for planetary distances.
**Status**: 📋 Not started
**Priority**: High — makes multiplayer presence visible
**Depends on**: F-020 Phase 2 (position streaming)
**Spec**: [f021-physical-marker-spec.md](f021-physical-marker-spec.md)

#### Phases
- Phase 1: Blinking sphere in player color; label overlay; screen-space minimum size
- Phase 2: Primitive IQM model tinted by player color (depends on F-008 or standalone)
- Phase 3: Stock + custom textured models from server catalog

---

### F-022 — Client Locomotion and Physics

**Value**: Connected clients can navigate the world using four movement types (drift, thrusters, impulse, superluminal). Client ships respond to gravity from all named bodies (requires F-013). NPC clients are server-driven.
**Status**: 📋 Not started
**Priority**: High — required for a playable multi-client experience
**Depends on**: F-020 Phase 1; F-013 for gravity (Phase 2 only)
**Spec**: [f022-client-movement-spec.md](f022-client-movement-spec.md)

#### Phases
- Phase 1: Kinematic movement (no gravity); all four movement types; `MovementService` proto
- Phase 2: Gravity integration (leapfrog N-body receive pass); requires F-013
- Phase 3: NPC automation (server-driven routine locomotion)

---

### F-023 — Keyboard Configuration

**Value**: Replace hardcoded key constants with a hardware-profile-aware, fully user-remappable binding system. Stock profiles for laptop, full keyboard, mouse+keyboard, and numpad. Hot-reload from `configs/keybindings.json`. Conflict detection at load time. Fulfills F-006 and F-007.
**Status**: 📋 Not started
**Priority**: High — required for F-022 movement controls; also cleans up TD-001 surface
**Depends on**: TD-001 (recommended cleanup before this); no hard blockers
**Spec**: [f023-keyboard-config-spec.md](f023-keyboard-config-spec.md)

#### Phases
- Phase 1: `InputAction` enum; `KeyMap`; laptop + mouse-keyboard profiles; `handleInput` refactor
- Phase 2: Full keyboard + numpad profiles
- Phase 3: Mouse integration (mouse-delta camera, scroll-wheel zoom)

---

### F-024 — Multiplayer HUD Enhancements

**Value**: HUD additions for multi-client awareness: own-client status panel, session list (Tab overlay), compass indicators for off-screen clients, proximity alert, admin session panel.
**Status**: 📋 Not started
**Priority**: Medium — enhances multiplayer UX; not blocking for core functionality
**Depends on**: F-020 Phase 2 (client sessions in WorldSnapshot); F-023 Phase 1 (toggle bindings)
**Spec**: [f024-multiplayer-hud-spec.md](f024-multiplayer-hud-spec.md)

#### Phases
- Phase 1: Own-client status panel + session list (Tab overlay)
- Phase 2: Compass indicators + proximity alert
- Phase 3: Admin session panel with kick action

---

### F-025 — Ship-to-Ship Communications

**Value**: Real-time in-game text messaging between connected client sessions: direct messages, broadcast, emotes, system events (join/leave/damage). Persistent comms log in HUD.
**Status**: 📋 Not started
**Priority**: Medium
**Depends on**: F-020 Phase 2 (SessionStream must exist)
**Spec**: [f025-ship-comms-spec.md](f025-ship-comms-spec.md)

#### Phases
- Phase 1: Text messaging (DM + broadcast) + system events + HUD comms log
- Phase 2: Emotes + admin mute/unmute
- Phase 3: Audio (reserved; out of scope until separately designed)

---

### F-026 — Audio Events

**Value**: Raylib-backed audio cue system for 10 game events (thruster, impulse, warp, damage, collision, proximity alert, join, leave, message received, system notification). Fully client-side; configurable; silent no-op when disabled.
**Status**: 📋 Not started
**Priority**: Medium — enhances immersion; fully independent of server logic
**Depends on**: None for Phase 1 (thruster + impulse + warp sounds); F-020 Phase 2 for Phase 2 (join/leave); F-027 Phase 1 for Phase 3 (collision/damage audio)
**Spec**: [f026-audio-events-spec.md](f026-audio-events-spec.md)

#### Phases
- Phase 1: AudioManager infrastructure + thruster / impulse / warp sounds
- Phase 2: Proximity / join / leave / message-received audio
- Phase 3: Collision / damage / warp audio cues

---

### F-027 — Ship Collision Detection and Damage

**Value**: Bounding-sphere collision detection between client ships; DamageRating model; ImpactEvent broadcast to all clients; respawn on lethal damage. Camera is constrained to never enter an object.
**Status**: 📋 Not started
**Priority**: Medium — adds gameplay consequence to movement; depends on movement system
**Depends on**: F-020 Phase 1 (session registry for DamageRating field); F-022 Phase 1 (kinematic movement for velocity-based impact energy)
**Spec**: [f027-collision-damage-spec.md](f027-collision-damage-spec.md)

#### Phases
- Phase 1: Broad-phase + narrow-phase bounding sphere detection; DamageRating update; ImpactEvent broadcast
- Phase 2: Respawn logic; visual hit indicator in HUD
- Phase 3: BVH broad-phase acceleration (deferred until > 50 active clients)

---

### F-028 — Input Config as Reusable UI Component

**Value**: The Controls tab currently renders keyboard (and eventually mouse) config as ad-hoc draw calls inside the settings dialog. Extract it into a self-contained component so the same editor can be embedded in other contexts (standalone overlay, onboarding wizard, etc.) and so Mouse config can plug in as a parallel section without duplicating layout code.
**Status**: 📋 Not started
**Priority**: Medium — architectural cleanup; prerequisite for a clean Mouse config section (see F-029 area and the current "not yet configurable" placeholder in the Controls tab)
**Depends on**: F-023 (keyboard binding system in place); Mouse config scope TBD

#### Notes

- "Component" means an isolated render function + matching state struct, callable from any dialog tab without access to the full `SettingsState`.
- Keyboard and Mouse sections should be independently embeddable (either can be omitted if the platform has no mouse or keyboard).
- The two-column keybinding list, group headers, Load/Save rows, inline path editor, and file picker overlay are all candidates for extraction.
- Mouse sensitivity, invert-Y, and button mapping are the expected Mouse config fields (scope not yet defined).

---

### F-029 — Enhanced Sol Corona / Active Atmosphere

**Value**: Sol's current corona is a static Fresnel glow. A real star has a visually dynamic corona — brightness variation, limb darkening, granulation texture, and animated rim effects. Making Sol feel "alive" significantly raises scene quality and believability.
**Status**: 📋 Not started
**Priority**: Medium — visual quality; builds on the existing `drawAtmosphereGlow` + `selfLuminous` shader path (DEF-003 fix)
**Depends on**: DEF-003 fixed (✅ complete); F-017 for full PBR/bloom integration (optional)

#### Notes

- Animated corona: time-driven noise in the atmosphere fragment shader (simplex or value noise offsetting the `litFactor` rim).
- Limb darkening: the photosphere center is brighter than the edge — add a `centerBrightness` falloff to the existing `litFactor` calculation.
- Granulation: subtle animated surface texture driven by a time-varying noise function on the sphere surface (not a texture — procedural).
- Separate from solar weather events (F-030); this feature is the ambient "active star" baseline look.

---

### F-030 — Solar Weather Events (Flares, CMEs, Charged Particle Storms)

**Value**: Discrete, timed solar events add dynamic interest and open gameplay hooks (communication interference, shield stress, navigation hazards). Three event types are scoped: solar flares (bright, brief arc on the limb), coronal mass ejections (expanding plasma shell), and charged particle storms (volumetric particle field propagating outward from Sol).
**Status**: 📋 Not started
**Priority**: Low-medium — high visual impact; depends on a particle/event system that does not yet exist
**Depends on**: F-029 (active corona baseline); F-026 (audio events for accompanying sounds); event queue for scheduling and broadcast

#### Event Types

| Type | Visual | Duration | Gameplay Hook |
|------|--------|----------|---------------|
| Solar flare | Bright arc / jet erupting from limb | 30–120 s | Brief light burst, potential comms disruption HUD flash |
| Coronal mass ejection (CME) | Expanding translucent plasma sphere from Sol surface | 5–30 min sim time | Hits objects in path; ImpactEvent hook for shield stress |
| Charged particle storm | Volumetric particle field expanding at solar wind speed | Hours of sim time | Persistent HUD degradation overlay while storm is active |

#### Notes

- Events are driven by the simulation clock (not real time) so time-scaling affects duration and propagation.
- CME propagation speed: ~500–3000 km/s real; mapped proportionally to sim units.
- Particle storm: rendered as a sparse billboard particle field; no physics collision, purely visual + HUD trigger.
- All three types should publish through `protocol.Broadcaster` so gRPC clients can observe and react.

---

### F-031 — Asteroid Visual Classification by Mass

**Value**: All asteroids currently render as spheres regardless of size. Real asteroids below dwarf-planet mass are irregularly shaped rubble piles; only the largest approach hydrostatic equilibrium (roughly spherical). Differentiating visually by mass makes the belt feel physically accurate and adds scene richness.
**Status**: 📋 Not started
**Priority**: Low — visual quality; requires irregular mesh generation or procedural deformation
**Depends on**: F-008 (artifact/non-sphere mesh pipeline) for the irregular mesh path; F-003 (texture pipeline) for rocky surface textures

#### Classification Rules (proposed)

| Mass threshold | Shape | Rendering |
|----------------|-------|-----------|
| ≥ dwarf planet mass (~5×10²⁰ kg) | Semi-spherical — oblate or slightly irregular sphere | Sphere with slight pole flattening and surface bump map |
| < dwarf planet mass | Irregular rocky body — polyhedra assembled from 1–N element "chunks" | Procedural low-poly mesh; element composition drives color/texture |

#### Notes

- "Element composition" means visual material properties (silicate = grey-brown, metallic = reflective silver, carbonaceous = dark matte) not atomic simulation.
- Low-poly irregular mesh: generate by starting with an icosphere and displacing vertices with low-frequency noise scaled to object radius; seed from object GUID for determinism.
- Element chunks: 1–4 material regions per object; region boundaries are Voronoi-partitioned on the mesh.
- Mass threshold constant should live in `engine/constants.go` next to `PointThresholdStar` and similar.
- At extreme distances (point-render threshold) all asteroids fall back to a colored point regardless of shape class.

---

### F-032 — Integrate Keybindings into All Simulator Commands ⚠️ HIGH PRIORITY

**Value**: The `InputAction` enum and `KeyMap` type exist and the Controls tab lets users save custom bindings, but the simulator's input handler still reads raw `rl.KeyXxx` constants directly in many places instead of routing through `KeyMap.IsPressed(action)`. Until this wiring is complete, user-configured bindings have no effect on actual simulator behaviour. This is the critical last step that makes the entire keybinding system functional end-to-end.
**Status**: 📋 Not started
**Priority**: 🔴 High — keybinding configuration is user-visible but silently non-functional until this lands; also required before mouse binding work can be meaningful
**Depends on**: F-023 UI complete (Controls tab, KeyMap load/save); no other hard blockers

#### Scope

- Audit `internal/client/go/raylib/app/input.go` and all input call sites for raw `rl.IsKeyPressed` / `rl.IsKeyDown` calls that correspond to a defined `InputAction`.
- Replace each with the equivalent `km.IsPressed(action)` or `km.IsDown(action)` call so the binding comes from the loaded `KeyMap`.
- Mouse bindings: once keyboard wiring is complete, define mouse `InputAction` values (look-axis, zoom scroll, button clicks) and route them through the same `KeyMap` dispatch path.
- Verify: rebinding an action in the Controls tab and saving should immediately change which key triggers that action in the running simulator (no restart required).

#### Acceptance Criteria

- No `rl.IsKeyPressed` / `rl.IsKeyDown` call in the app input handler references a key that also has an `InputAction` mapping.
- All 28 actions in `OrderedActions()` are wired; each can be rebound and the new binding takes effect within the same session.
- Mouse look-axis and scroll-zoom route through `KeyMap` (or a parallel `MouseMap`) rather than hardcoded axis reads.
- `go test ./...` passes; race detector clean.

---

- [docs/standards/agent-readme.md](../standards/agent-readme.md): architecture, package map, boundaries.
- [docs/standards/coding-standards.md](../standards/coding-standards.md): implementation standards and Definition of Done.
- [docs/history/lessons-learned.md](../history/lessons-learned.md): anti-patterns and root-cause notes.
- [docs/history/changelog.md](../history/changelog.md): completed work archive.

- [docs/history/changelog.md](../history/changelog.md): completed work moved out of the live queue
- [docs/standards/guidance.md](../standards/guidance.md): workflow and work-tracking rules
- [internal/space/package.md](../../internal/space/package.md): current package and architecture context
- [docs/standards/agent-readme.md](../standards/agent-readme.md): repository orientation for agents