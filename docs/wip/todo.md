# Space Sim Work Queue

## Purpose
Track active and future work for Space Sim in one operational backlog. Keep this file focused on work that is not yet done.

## Last Updated
2026-04-13

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
6. Feature Backlog
	F-001 Camera Collision Prevention
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
7. Recommended Ordering
8. Tech Debt
	TD-001 Collapse handleInput / updateCameraState Param Lists
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
| 5 | **F-010** Multi-machine split (headless server + client) | Network foundation |
| 6 | **F-011** IAAM (identity, roles, auth) | Safety layer for multi-client; immediately after F-010 |
| 7 | **F-008** Artifact object type | Content foundation for F-009 |
| 8 | **F-009** Object-object collision/proximity | Needs F-008 for full value |
| 9 | **F-007** User-configurable key bindings | Group 3; do after TD-001 reduces handleInput complexity |
| 10 | **F-006** XYZ keyboard nav + mouse facing | Benefits from F-007 |
| 11 | **F-002** REPL track / track stop | Remaining Group 1 quick win |
| 12 | **F-003** Textures on planets/moons | Group 2 visual; floating-origin fix (DEF-001) already done |
| 13 | **F-004** Procedural star field | Group 2 visual |
| 14 | **F-005** Physical lighting from stars | Needs F-003 |
| 15 | **F-012** Federated compute | Long-term exploratory; F-010, F-011, F-013 must be stable. Own phase. |
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

## 6. Feature Backlog

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

### F-004 — Procedural Star Field Background

**Value**: Replace the blank black background with a static star field that shifts parallax-correctly with camera orientation (rotation only; no translational parallax at solar-system scale).
**Status**: 📋 Not started
**Priority**: Medium — high visual quality improvement; independent of simulation state
**Depends on**: Camera forward vector (already available)

#### Work Items

- [ ] Generate a deterministic set of background star positions on a unit sphere at startup (seeded RNG, configurable count)
- [ ] Draw as a sky-box or point-sprite pass before the 3D scene, using camera orientation only (strip camera translation from the view matrix)
- [ ] Vary brightness and color temperature by a Gaussian distribution approximating the Milky Way density band
- [ ] Optional: load a real-star catalog (Hipparcos subset) for accurate positions

---

### F-005 — Physical Lighting from Stars

**Value**: Drive scene lighting from simulated star properties (luminosity derived from mass/type/color) rather than a fixed point light.
**Status**: 📋 Not started
**Priority**: Medium — depends on star data already in `solar_system.json`; requires shader or multi-light Raylib work
**Depends on**: F-003 (textures must be bound before physically-based lighting is meaningful); star `mass`, `color`, and `radius` fields in `ObjectMeta`

#### Work Items

- [ ] Compute luminosity from star mass using a mass-luminosity approximation (L ∝ M^3.5 for main sequence)
- [ ] Map luminosity + color to Raylib light intensity and tint
- [ ] Support multiple stars in the scene (alpha Centauri system has two)
- [ ] Inverse-square falloff per-object from each star's position
- [ ] CLI flag `--no-lighting` (default: physical lighting on); persist in `app.json`

---

### F-006 — XYZ Keyboard Navigation + Mouse Facing

**Value**: Provide an explicit 6-DOF free-fly mode (X/Y/Z translate via keyboard, yaw/pitch via mouse) as an alternative to the current relative WASD scheme. Useful for precise positioning and scripted camera work.
**Status**: 📋 Not started
**Priority**: Medium
**Depends on**: F-007 (key remapping) is a prerequisite only if the new bindings would conflict; can ship with hardcoded defaults first

#### Work Items

- [ ] Add absolute-axis translate inputs (default: arrow keys + PgUp/PgDn, or configurable)
- [ ] Expose a toggle between current relative mode and absolute XYZ mode
- [ ] Decouple mouse look from movement mode so mouse always controls facing regardless of translate mode

---

### F-007 — User-Configurable Key Bindings

**Value**: Allow players to remap any action to a different key without recompiling. Required for accessibility and non-QWERTY layouts.
**Status**: 📋 Not started
**Priority**: Medium — unblocks F-006 without conflict; architectural change touches `input.go`
**Depends on**: Nothing; but complete TD-001 first to reduce `handleInput` complexity before restructuring it further

#### Work Items

- [ ] Define an action enum covering all current hard-coded key uses in `input.go`
- [ ] Load a key-binding map from a config file (JSON, same pattern as `app.json`)
- [ ] Replace `rl.IsKeyPressed(rl.KeyX)` call sites with action-lookup helper
- [ ] Provide a default binding file; allow user overrides via a `keybindings.json` in the config directory
- [ ] REPL / HUD: display current binding for each action in the help overlay

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
**Status**: 📋 Not started
**Priority**: Medium-high — accuracy is a stated project goal; also a prerequisite for F-012 accuracy claims
**Depends on**: Nothing blocking; self-contained change to `internal/sim/engine/physics.go`. Coordinate with F-012 design (partition strategy depends on how forces are computed).

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
- [ ] Implement Rayleigh scattering as a screen-space approximation: for each atmosphere-bearing body, draw an additive-blended disc slightly larger than the sphere, color from `AtmosphereColorHint`, scaled by `AtmosphereThicknessKm`
- [ ] Add `NightTexturePath string` to `ObjectMetadata` and `rendering.night_texture_image` to JSON schema; blend night texture where the star is below the horizon
- [ ] Expose `--no-shadows`, `--no-bloom`, `--no-atmosphere` CLI flags; persist in `app.json`

---

## 9. Related Docs

- [docs/standards/agent-readme.md](../standards/agent-readme.md): architecture, package map, boundaries.
- [docs/standards/coding-standards.md](../standards/coding-standards.md): implementation standards and Definition of Done.
- [docs/history/lessons-learned.md](../history/lessons-learned.md): anti-patterns and root-cause notes.
- [docs/history/changelog.md](../history/changelog.md): completed work archive.

- [docs/history/changelog.md](../history/changelog.md): completed work moved out of the live queue
- [docs/standards/guidance.md](../standards/guidance.md): workflow and work-tracking rules
- [internal/space/package.md](../../internal/space/package.md): current package and architecture context
- [docs/standards/agent-readme.md](../standards/agent-readme.md): repository orientation for agents