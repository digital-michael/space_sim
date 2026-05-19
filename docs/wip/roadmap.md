# Space Sim — Project Roadmap

## Purpose

Single source of truth for project goals, known requirements, and the prioritized implementation plan. Draws from `todo.md` (work items) and session planning discussions. This document describes *what* and *why*; `todo.md` carries the *how* (work items, acceptance criteria, decisions).

## Last Updated
2026-05-18

---

## 1. Project Goals

| # | Goal |
|---|------|
| G1 | Accurate, real-time solar system simulation with N-body gravity and barycenter motion |
| G2 | High-fidelity visual experience: textures, physically-based lighting, star field, floating-origin rendering at all scales |
| G3 | Multi-machine architecture: headless simulation server, multiple independent Raylib renderer clients connected over gRPC |
| G4 | Role-based access control (IAAM): identity, authentication, and per-role command permissions |
| G5 | Extensible object model: natural bodies + artifact types (probes, ships, comets) with collision/proximity events |
| G6 | Configurable controls: user-remappable key bindings, 6-DOF navigation |
| G7 | Federated compute: distribute physics across multiple nodes to scale simulation fidelity beyond a single machine |
| G8 | Simulation accuracy and experience quality take precedence over maximizing concurrent user count |

---

## 2. Known Requirements

### Simulation
- N-body gravitational integration with barycenter output per bound group
- `float64` positions in the physics layer for long-run accuracy (feeds DEF-001 fix)
- Keplerian initialization seeds N-body starting state; JSON schema unchanged
- Belt particles remain Keplerian (full N-body at tens of thousands of objects is prohibitive)
- Named bodies (planets, moons, stars, dwarf planets) use full N-body force sums

### Rendering
- Camera-relative (floating origin) rendering to eliminate float32 precision collapse at distance
- Diffuse, normal, specular, night-lights, and cloud texture layers per body where available
- Procedural star field shifted by camera orientation (rotation-only parallax)
- Physical point lights per star, inverse-square falloff, intensity derived from stellar mass/type
- Symbolic / LOD representation for extreme zoom-out (scope after DEF-001 fix)

### Network / Multi-Machine
- `space-sim-server`: headless, containerized, single gRPC port; admin access via server-local REPL or admin-role connection
- `space-sim-client`: Raylib renderer dialing remote server; camera state fully local
- POV frustum filtering: client sends frustum to server, server filters snapshot to relevant objects
- Client-side interpolation between received snapshots for smooth rendering at reduced stream rate
- Delta compression and LOD-by-distance as follow-on bandwidth mitigations

### Identity, Access, Authentication, Management (IAAM)
- Separate auth backend (web frontend + token service); `space-sim-server` validates tokens at connect time
- Bearer token over TLS on all gRPC connections
- Roles: `admin`, `moderator`, `team_lead`, `user` (see permission table in F-011)
- Connection audit log per session

### Controls
- User-configurable key bindings loaded from `keybindings.json`
- 6-DOF free-fly mode (absolute XYZ translate + mouse facing)
- REPL: `track <name>` and `track stop`; camera never clips inside a body

### Federated Compute
- Simulation accuracy first; redundancy second
- User cap acceptable to protect simulation performance
- Compute nodes trusted (internal network)
- Partition strategy and reconciliation TBD pending F-013 N-body design

---

## 3. Status-Driven Implementation Plan

Items are ordered by the agreed priority sequence. Dependencies are noted per item. Items that can execute independently of one another in the same window are grouped under the same step.

### Legend

| Symbol | Meaning |
|--------|---------|
| 📋 | Not started |
| 🔄 | In progress |
| ✅ | Complete |
| ⏸ | Deferred |
| 🔍 | Exploration / spike needed |

### Phase Summary

| Phase | Steps | Focus | Status |
|-------|-------|-------|--------|
| Pre-A | 1, 2, 3 | N-body foundation, camera fix, floating origin | 1 done (DEF-001 ✅); 1+2 pending |
| A | 4 | Input cleanup + keybinding gap closure (TD-001 + F-032) | ✅ Complete |
| B | 4a | Ship definition catalog (F-033 Phase 1) | 📋 Not started |
| C | 4b | Kinematic movement + player as ship (F-022 Phase 1 + §9) | 📋 Not started |
| D | 4c | Player physical marker (F-021 Phase 1) | 📋 Not started |
| E | 5 | Multi-client session registry (F-020 Phase 1) | 📋 Not started |
| F | 6, 7 | Network split + IAAM (F-010, F-011) | 📋 Not started |
| G | 8, 9 | Object model expansion (F-008, F-009) | 📋 Not started |

---

### Step 1 — F-013: N-Body Barycenter Integration
**Phase**: Pre-A (foundation; independent)
**Status**: 📋 Not started — implementation plan complete ([f013-nbody-plan.md](f013-nbody-plan.md))  
**Independent**: Yes — self-contained to `internal/sim/engine/physics.go`  
**Why first**: Physics accuracy is a stated project goal (G1). Validating correctness in a single-process model before distributing (F-010, F-012) avoids compounding errors. Also informs DEF-001's `float64` position requirement and F-012's partition strategy.

Key decisions to make during design:
- Integrator: leapfrog (symplectic) vs. RK4 vs. Verlet
- Force computation: O(N²) brute-force (adequate for ~100 named bodies) vs. Barnes-Hut for scale
- N-body as opt-in mode per system JSON vs. global switch

---

### Step 2 — F-001: Camera Collision Prevention
**Phase**: Pre-A (quick win; pull into any gap)
**Status**: 📋 Not started  
**Independent**: Yes — contained to `UpdateTracking` and `UpdateJump` in `camera.go`  
**Why here**: 30-minute fix; affects every recording and demo session from this point forward. No benefit to deferring.

---

### Step 3 — DEF-001: Floating-Origin Rendering (Precision Collapse Defect)
**Phase**: Pre-A (done)
**Status**: ✅ Complete (implemented; roadmap status was stale)
**Implementation**: `interactive.go` uses a zero-origin camera (`camera.Position = rl.Vector3{}`) and shifts all object positions by `-cameraPos` before passing to Raylib draw calls. The GPU always operates near `(0,0,0)`. Physics and protocol layers continue to use full world-space coordinates.
**Verified by**: Floating-origin camera block at `interactive.go` line ~80.

---

### Step 4 — TD-001 + F-032: Input Cleanup + Keybinding Gap Closure
**Phase**: A — Input Completeness
**Status**: ✅ Complete (commit 43577bc)

**TD-001 target shape**:
```go
func (a *App) handleInput(session *runtimeSession, state *engine.SimulationState) bool
func (a *App) updateCameraState(session *runtimeSession, dt float32) float32
```

**F-032 target**: All 19 missing `InputAction` constants added to `actions.go`; every hardcoded `rl.IsKeyPressed`/`rl.IsKeyDown` site in `input.go` replaced with `km.IsPressed(action)`/`km.IsDown(action)`. See `todo.md` F-032 for the per-action audit table.

**Unlocks**: User Feature 1 (keyboard config) complete; all controls rebindable via `configs/keybindings.json` with immediate effect.

### Step 4a — F-033 Phase 1: Ship Definition Catalog
**Phase**: B — Ship Data Layer
**Status**: 📋 Not started  
**Independent from F-020**: Yes — `ShipInstance` attaches to `runtimeSession` for the direct binary now; migrates to `ClientSession` (F-020) later.  
**Why before F-022**: F-022 kinematic movement reads thrust acceleration and turning rate from `ShipInstance`. Without F-033, F-022 uses the stub `ShipProfile` values which are unrated and produce an arcade-feel ship.

Delivers:
- `internal/server/ship/definition.go` — `ShipDefinition`, `EngineStage`, `TurningSpec`, `PowerSpec`
- `internal/server/ship/catalog.go` — scan `data/ships/*.json` at startup
- `internal/server/ship/instance.go` — `ShipInstance` with live runtime state; transponder assignment
- `data/ships/scout_mk1.json`, `freighter_t1.json`, `explorer_x1.json` — three bundled ships
- `runtimeSession.ShipInstance *ship.ShipInstance` — assigned at session creation from default ship

**Unlocks**: User Feature 2 (file-driven ship definitions) complete; ship ratings available for F-022.

---

### Step 4b — F-022 Phase 1 + §9: Kinematic Movement + Player as Ship
**Phase**: C — Player Movement
**Status**: 📋 Not started  
**Depends on**: TD-001 (cleaner call sites), F-033 Phase 1 (ship ratings), F-023 Phase 1 (thrust/turn bindings wired via F-032)  
**Independent from F-020**: Yes — single-player movement works in `space-sim-direct` directly.

Delivers:
- Camera forward vector mirrored as `ShipInstance.FacingVector` each input frame
- Thrust applied along facing at `stage.accel_max_ms2 × mass_kg`, capped to `max_speed`
- Turn rate capped to `turning.rate_deg_per_s` (from ShipInstance)
- Four movement modes: thrusters, drift, impulse, warp (REPL-driven)
- Kinematic integration: `velocity += accel × dt`, `position += velocity × dt`
- No gravity in Phase 1 (gravity requires F-013)

**Unlocks**: User Feature 3 (Player POV as ship) complete; player navigates via ship engine ratings.

---

### Step 4c — F-021 Phase 1: Player Physical Marker
**Phase**: D — Visual Confirmation
**Status**: 📋 Not started  
**Depends on**: F-020 Phase 1 OR runtimeSession position for single-player  
**Can ship for single-player without F-020**: Yes — render a blinking sphere at `runtimeSession.ShipInstance.Position` in `renders.go`.

Delivers:
- Blinking sphere rendered at player ship position
- Distinct color per session (from transponder or hardcoded for single-player)
- HUD crosshair when player ship is in view

**Unlocks**: Player position is visible; confirms F-022 movement is working.

---

### Step 5 — F-020 Phase 1: Multi-Client Session Registry
**Phase**: E — Multi-Client Foundation
**Status**: 📋 Not started  
**Depends on**: Steps 4/4a/4b (ship and movement proven in single-player first)  
**Independent from F-010**: Yes — the session registry is the in-process data structure; F-010 is the separate-process networking layer.

Migrates `runtimeSession.ShipInstance` → `ClientSession.ShipInstance` when this lands.

---

### Step 6 — F-010: Multi-Machine Architecture (Option B split)
**Phase**: F — Network Split
**Status**: 📋 Not started  
**Depends on**: TD-001, F-020  
**Independent from**: F-013, DEF-001, visual group

Two parallel sub-tracks:
- **Group A** `cmd/space-sim-server`: headless entrypoint, no Raylib, containerized
- **Group B** `cmd/space-sim-client`: Raylib renderer consuming `WorldService.StreamSnapshot` over gRPC

---

### Step 7 — F-011: IAAM — Identity, Access, Authentication, Management
**Phase**: F — Network Split
**Status**: 📋 Not started  
**Depends on**: F-010 (deployment model shapes token delivery)  
**Independent from**: Visual group, F-013

Roles and permission table already drafted (see `todo.md` F-011). Auth backend is an external component (separate repo/service); this step covers only the `space-sim-server` side (interceptor, context attachment, role enforcement, audit log).

---

### Step 8 — F-008: Artifact Object Type
**Phase**: G — Object Model Expansion
**Status**: 📋 Not started  
**Independent from**: F-010, F-011 (can start after Step 4)  
**Blocks**: F-009

Introduces `CategoryArtifact`, mesh loading via Raylib `LoadModel`/`DrawModel`, schema extension for `"type": "artifact"`. Bounding-sphere approximation for camera collision and frustum culling.

---

### Step 9 — F-009: Object-Object Collision / Proximity Detection
**Phase**: G — Object Model Expansion
**Status**: 📋 Not started  
**Depends on**: F-008 (for non-trivial shape detection)  
**Independent from**: Network group

Broad-phase spatial sweep → narrow-phase sphere-sphere → `ProximityEvent` through `protocol.Broadcaster`.

---

---

### Step 9 — F-007 and F-006: Superseded/Deferred
**F-007** (User-Configurable Key Bindings): ❌ **Superseded by F-023**. F-023 is the full spec and implementation; F-007 is retired.
**F-006** (XYZ Keyboard Navigation + Mouse Facing): ⏸ **Deferred to F-023 Phase 3**. Mouse-delta camera and 6-DOF nav are Phase 3 of F-023.

---

### Step 11 — F-002: REPL `track <object>` / `track stop`
**Status**: ✅ Complete — 2026-04-09  
**Independent**: Yes — extends existing `CameraTrackCmd` and REPL command dispatch  
**Note**: Pulled forward and completed before Steps 1–10. `track <name>` / `track stop` top-level REPL verbs with TAB completion (stop + body names).

---

### Step 12 — F-003: Texture / Bitmap Rendering
**Status**: ✅ Complete — 2026-04-14 (pulled forward before DEF-001)  
**Depends on**: ~~DEF-001~~ (dependency relaxed; float32 precision acceptable at inner-system camera distances where textures are visible)  
**Independent from**: Network group

Diffuse texture layer + UV correction (V-flip, U-mirror, pole rotation). `--no-textures` flag. Solid-color fallback. Normal-map, specular, night-lights deferred to F-017.

---

### Step 13 — F-004: Procedural Star Field Background
**Status**: 📋 Not started  
**Depends on**: DEF-001  
**Independent from**: F-003 (can run in parallel with Step 12)

Deterministic unit-sphere point distribution, camera-orientation-only parallax, Gaussian brightness/color-temperature distribution.

---

### Step 14 — F-005: Physical Lighting from Stars
**Status**: ✅ Complete — 2026-04-21 (pulled forward before F-013)  
**Depends on**: F-003 ✅; ~~F-013~~ (dependency relaxed; SolarLuminosity field used directly instead of mass-derived value)  
**Independent from**: Network group

GLSL 330 Phong shader compiled at runtime. Inverse-square falloff, up to 4 star lights, warm-white emission fallback, `--no-lighting` flag, day/night terminator confirmed working.

---

### Step 14b — F-017a: Atmosphere Limb Glow + Night-Side City Lights
**Status**: ✅ Complete — 2026-04-21 (pulled forward; partial delivery of F-017)  
**Depends on**: F-003 ✅, F-005 ✅  

`drawAtmosphereGlow`: additive-blended sphere at physicalRadius×1.10 using `AtmosphereColorHint`; visible for Venus, Earth, Jupiter, Saturn, Uranus, Neptune, Sol corona. Extended `phongFS` shader: `texture1` (city lights) blended on dark side via `smoothstep` on `maxDiff`; `hasNightTexture` int uniform guards the branch. Earth wired to `earth_nightmap.jpg`.

---

### Step 15 — F-012: Federated Compute
**Status**: 📋 Not started — exploratory  
**Depends on**: F-010, F-011, F-013 (all must be stable); `internal/server/pool/distributed/` stub is the entry point  
**This is its own phase**; do not treat as part of Steps 5–6 sprint

Open research questions (partition strategy, reconciliation, clock sync, failure handling) must be resolved before implementation begins. Own planning document when the time comes.

---

### Deferred

| Item | Status | Notes |
|------|--------|-------|
| 4.7 Belt overlap / speed uniqueness | ⏸ Deferred | Cosmetic only; insert whenever bandwidth allows |

---

## 4. Independent vs. Dependent Summary

### Can execute independently (no cross-item deps within the plan)
- F-013 N-body (Step 1)
- F-001 Camera collision (Step 2)
- DEF-001 Floating origin (Step 3, after F-013 float64 work lands)
- TD-001 Param collapse (Step 4)
- F-008 Artifact type (Step 7, after TD-001)
- F-002 REPL track (Step 11, anytime)
- F-003 Textures (Step 12, after DEF-001)
- F-004 Star field (Step 13, after DEF-001, parallel with F-003)

### Dependent chains
```
DEF-001 ✅ (floating origin done; unblocks F-004, confirmed F-003 safe)

F-013 ──► F-022 Phase 2 (gravity requires N-body engine)
F-013 ──► float64 physics positions (long-run accuracy)

TD-001 + F-032 (do together: clean handleInput, then wire all keybinding gaps)
  └──► TD-002  (clean loop structure before goroutine split)
  └──► F-023 full phase completion

F-033 Phase 1 ──► F-022 Phase 1 (ship ratings available for kinematic movement)
  └──► F-022 §9 (camera→FacingVector; thrust along facing; turn rate from ShipInstance)
  └──► F-021 Phase 1 (render blinking sphere at ShipInstance.Position)

F-020 Phase 1 ──► F-022 Phase 1 multi-client (movement in session registry)
F-020 Phase 1 ──► F-021 Phase 1 multi-client (markers for all sessions)
NOTE: F-033 Phase 1 and F-022 §9 + F-021 Phase 1 can ship for single-player
      WITHOUT F-020 by attaching ShipInstance to runtimeSession directly.

F-010 ──► F-011     (split before auth)
F-020 + F-023 ──► F-024 (HUD reads sessions; uses toggle bindings)
F-020 Phase 2 ──► F-025 Phase 1 (messaging needs SessionStream)
F-020 Phase 2 + F-025 Phase 1 ──► F-026 Phase 2 (audio needs join/leave/message events)
F-022 Phase 1 ──► F-027 Phase 1 (collision needs position + velocity tracking)
F-027 Phase 1 ──► F-026 Phase 3 (damage audio needs ImpactEvent)
F-033 Phase 2 ──► F-021 Phase 2 (ship 3D model via IQM render pipeline)
F-033 Phase 3 ──► engine stage HUD (power budget panel)

F-008 ──► F-009                (artifact type before collision events)
F-010 + F-011 + F-013 ──► F-012  (all stable before federated compute)

F-007 ❌ superseded by F-023
F-006 ⏸ deferred to F-023 Phase 3
```

---

## 6. Multi-Client Feature Group

These features together deliver a playable multi-client experience against the existing
`space-sim-grpc` Option A binary. They are independent of F-010/F-011 and can be
sequenced after the visual group.

| Feature | Value | Spec |
|---------|-------|------|
| F-020 Multi-Client gRPC Session Layer | Session registry, identity, 100-client cap | [f020-multi-client-spec.md](f020-multi-client-spec.md) |
| F-021 Client Physical Marker | Visible in-world presence (sphere → IQM → textured) | [f021-physical-marker-spec.md](f021-physical-marker-spec.md) |
| F-022 Client Locomotion and Physics | Drift / thrusters / impulse / superluminal; gravity | [f022-client-movement-spec.md](f022-client-movement-spec.md) |
| F-023 Keyboard Configuration | Hardware profiles; remappable; hot-reload (fulfills F-006, F-007) | [f023-keyboard-config-spec.md](f023-keyboard-config-spec.md) |
| F-033 Ship Definition | Externally-loaded ship catalog; engine stages; power; damage; identity | [f033-ship-definition-spec.md](f033-ship-definition-spec.md) |
| F-024 Multiplayer HUD Enhancements | Session list, compass, proximity alert, admin panel | [f024-multiplayer-hud-spec.md](f024-multiplayer-hud-spec.md) |
| F-025 Ship-to-Ship Communications | Text DM + broadcast via SessionStream; emotes; comms HUD log | [f025-ship-comms-spec.md](f025-ship-comms-spec.md) |
| F-026 Audio Events | Raylib audio cues for 10 game events; client-only | [f026-audio-events-spec.md](f026-audio-events-spec.md) |
| F-027 Ship Collision Detection and Damage | Bounding-sphere impact; DamageRating; ImpactEvent broadcast | [f027-collision-damage-spec.md](f027-collision-damage-spec.md) |

Recommended sequencing within the group:
1. **TD-001 + F-032** — clean handleInput + wire all keybinding gaps (no deps; unblocks everything)
2. **F-033 Phase 1** — ship catalog + instance; attaches to `runtimeSession` for single-player
3. **F-022 Phase 1 + §9** — kinematic movement + camera-as-ship (reads F-033 ratings)
4. **F-021 Phase 1** — blinking sphere marker at player position
5. **F-020 Phase 1** — session registry; migrate ShipInstance from runtimeSession to ClientSession
6. **F-022 Phase 1 multi-client** — movement over session registry
7. **F-023 Phase 2/3** — full keyboard + numpad + mouse-delta profiles
4. F-020 Phase 2 (position streaming — unlocks marker placement, comms, audio)
5. F-021 Phase 1 (blinking sphere, needs F-020 Phase 2)
6. F-025 Phase 1 (text messaging, needs F-020 Phase 2 SessionStream)
7. F-026 Phase 1 (audio infrastructure — independent; can run in parallel with steps 3–6)
8. F-024 Phase 1 (session list + own-client status, needs F-020 Phase 2 + F-023 Phase 1)
9. F-027 Phase 1 (collision detection, needs F-022 Phase 1 + F-020 Phase 1)
10. F-026 Phase 2 (proximity/join/leave audio, needs F-020 Phase 2 + F-025 Phase 1)
11. F-026 Phase 3 (collision/damage audio, needs F-027 Phase 1)
12. F-021 Phase 2 / F-022 Phase 2 / F-024 Phase 2 — parallel

---

## 5. Related Documents

| Document | Purpose | Status |
|----------|---------|--------|
| [docs/wip/todo.md](todo.md) | Full work items, acceptance criteria, decisions per feature | Active |
| [docs/history/repl-expansion.md](../history/repl-expansion.md) | Original REPL expansion design — **archived; all phases complete** | Archived |
| [docs/history/smoke-test-origin.md](../history/smoke-test-origin.md) | Origin story from February 2026 smoke test — **archived** | Archived |
| [docs/history/changelog.md](../history/changelog.md) | Completed work archive | Active |
| [docs/history/lessons-learned.md](../history/lessons-learned.md) | Anti-patterns and root-cause notes | Active |
| [docs/standards/agent-readme.md](../standards/agent-readme.md) | Repository map, package ownership, architectural boundaries | Active |
| [docs/wip/f020-multi-client-spec.md](f020-multi-client-spec.md) | Multi-client session layer spec | Active |
| [docs/wip/f021-physical-marker-spec.md](f021-physical-marker-spec.md) | Client physical marker spec | Active |
| [docs/wip/f022-client-movement-spec.md](f022-client-movement-spec.md) | Client locomotion and physics spec | Active |
| [docs/wip/f023-keyboard-config-spec.md](f023-keyboard-config-spec.md) | Keyboard configuration spec | Active |
| [docs/wip/f024-multiplayer-hud-spec.md](f024-multiplayer-hud-spec.md) | Multiplayer HUD enhancements spec | Active |
| [docs/wip/f025-ship-comms-spec.md](f025-ship-comms-spec.md) | Ship-to-ship communications spec | Active |
| [docs/wip/f026-audio-events-spec.md](f026-audio-events-spec.md) | Audio events spec | Active |
| [docs/wip/f027-collision-damage-spec.md](f027-collision-damage-spec.md) | Ship collision detection and damage spec | Active |
