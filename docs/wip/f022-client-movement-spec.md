# F-022 — Client Locomotion and Physics

## Purpose

Define how connected client sessions move through the simulation world and interact with
gravity. This feature transforms sessions from static presence markers into physically
coherent actors that obey the same gravitational field as natural bodies.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md)
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — session registry and position ownership
- [`docs/wip/f013-nbody-plan.md`](f013-nbody-plan.md) — gravity engine; required for realistic gravity
- [`docs/wip/f023-keyboard-config-spec.md`](f023-keyboard-config-spec.md) — control bindings that drive movement
- [`docs/wip/f021-physical-marker-spec.md`](f021-physical-marker-spec.md) — marker follows position; renders ship 3D model
- [`docs/wip/f033-ship-definition-spec.md`](f033-ship-definition-spec.md) — ship capability ratings that constrain movement; replaces the minimal ShipProfile in §6

## Last Updated
2026-05-18

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Four named movement modes with distinct physics behavior |
| G2 | Client ships respond to gravitational forces from all named bodies (N-body, F-013 required) |
| G3 | Client mass is physically correct for the ship size (negligible effect on simulation bodies) |
| G4 | Movement is server-authoritative; clients send intent, server applies physics and returns position |
| G5 | Movement modes are accessible via keyboard (F-023) and REPL commands |
| G6 | NPC clients (role = NPC) use server-driven autopilot rather than client input |

---

## 2. Non-Goals (this feature)

- Ship-to-ship collisions and damage (F-027 scope)
- Weapon systems or combat mechanics
- Wormhole or portal travel (separate future feature)
- Fluid dynamics (atmospheric entry effects deferred)
- AI-driven NPC behavior via MCP/REPL bridge (noted; deferred to F-022 Phase 3 extended)

---

## 3. Movement Types

### 3.1 Drift

**Behavior**: No thrust applied. The client ship moves only under gravitational influence and
carries its current velocity. A ship placed at rest relative to an orbiting body will fall
toward the nearest gravitational attractor.

**Physics**: Client velocity vector integrated using the same leapfrog step as F-013 named
bodies. Client mass is treated as a test particle (does not affect N-body force sums on
other objects — mass is too small to matter).

**Use case**: Coasting, sightseeing, freefall orbits.

**REPL command**: `move drift` — enables drift mode for the current session.

### 3.2 Thrusters

**Behavior**: Directional thrust applied along the client's POV vector (and strafe/up axes).
Thrust adds a velocity delta each simulation frame for as long as the input is held.
Releasing the input ceases thrust; existing velocity (and gravity) continues.

**Physics**:
```
acceleration = thrust_force / client_mass
velocity += acceleration * dt
position += velocity * dt   (leapfrog kick-drift)
```

`thrust_force` is configurable per client profile (see §6). Default: `1e10 N` (provides
perceptible acceleration at a 1,000 kg ship over seconds, not years).

**Use case**: Normal space navigation; maneuvering near bodies.

**REPL command**: Driven by keyboard bindings (F-023); no REPL text command in normal use.
REPL: `move thrust <dx> <dy> <dz>` for scripted/NPC movement.

**Heading**: Thrust direction is the client's current POV vector. The ship's POV is
updated by mouse/keyboard rotation independently of its velocity vector, so clients can
thrust in any direction relative to the camera.

### 3.3 Impulse

**Behavior**: Instantaneous velocity set. The client's velocity vector is set to a specified
value in a single RPC call. Gravity continues to act from the new velocity. Useful for
quickly repositioning to a target orbit or body.

**Physics**: Sets `velocity = impulse_vector` directly. Bypasses incremental thrust.

**Magnitude limit**: `impulse_max_speed_sim_units_per_s` in `configs/app.json`.
Default: `10.0` sim units/s (~1 AU/s, which is fast but sub-superluminal).

**Use case**: Teleport-adjacent maneuver; setting up an orbit; large repositioning.

**REPL command**: `move impulse <vx> <vy> <vz>` (sim units/s).

### 3.4 Superluminal

**Behavior**: Instantaneous world-position translation with no physics intermediate. The client
is moved to a target position or offset without traversing the intervening space. Gravity
resumes from the new position with zero inherited velocity unless `--carry-velocity` is
specified.

**Physics**: Direct registry `SetPosition` call. Velocity reset to zero. N-body gravity
accumulator reset. No simulation frames consumed in transit.

**Use case**: Jumping between star systems (e.g., Sol to Alpha Centauri); admin teleportation;
spectator mode repositioning.

**Constraint**: Role restriction applies — see table below.

**REPL command**: `move warp <body-name>` or `move warp <x> <y> <z>`.

### 3.5 Role restrictions

| Movement type | PLAYER | NPC | ADMIN | OTHER |
|--------------|--------|-----|-------|-------|
| Drift | ✓ | ✓ | ✓ | ✓ |
| Thrusters | ✓ | server only | ✓ | ✓ |
| Impulse | ✓ | server only | ✓ | ✓ |
| Superluminal | ✓ (own session) | server only | ✓ (any session) | ✓ (own session) |

---

## 4. Gravity Interaction

### 4.1 Dependency on F-013

Gravity for client ships requires the leapfrog N-body integrator from F-013. Before F-013
ships, client position updates are **kinematic only** (velocity integration without
gravitational force). The gravity flag is a runtime toggle:

```json
// configs/app.json
"client_gravity_enabled": false   // default until F-013 ships; true after
```

When `false`, gravity is not applied to client sessions. Movement still works correctly in all
four modes.

### 4.2 Client mass and force budget

Client ships are test particles. Their mass (~1,000 kg for a small ship) is ~10²⁷ times
smaller than Earth. Adding client force sums to the N-body loop would:
- Have zero measurable effect on any named body's orbit.
- Increase the N-body loop by one summation per client (negligible).

Implementation decision: **Include client ships in the N-body force-receive pass but
exclude them from the force-exert pass.** Named bodies never feel force from client ships.
This is the standard test-particle approximation.

### 4.3 Orbit stability

A client placed in a stable circular orbit will maintain that orbit indefinitely under
pure leapfrog gravity (symplectic integrator preserves energy). This provides realistic
orbital mechanics for players who want to orbit a body naturally.

---

## 5. NPC Client Automation

NPC clients (role = NPC) are server-instantiated sessions with scripted or AI-driven movement.
Their position updates come from the server event loop rather than client RPC calls.

NPC movement is driven by a `RoutineFunc` registered with `internal/server/routines/`. Each
routine tick computes the next thrust vector (waypoint following, patrol, orbit insertion)
and calls `Registry.UpdatePosition` and `Registry.UpdatePOV` directly.

NPC automation profiles (patrol, orbit, intercept) are deferred to a follow-on spec.

---

## 6. Client Ship Profile

**Note**: The minimal profile below is the Phase 1 stub. It will be **superseded by
`ShipInstance` from F-033** (Ship Definition) once that feature lands. F-033 defines full
engine stages, turning rating, power budget, and damage state. F-022 Phase 1 uses
the stub; F-022 Phase 2+ reads from `ShipInstance`.

Each client session carries a `ShipProfile` with physics-relevant parameters.
Default values ship in `configs/app.json`; clients may customize within server-defined limits.

```json
{
  "ship_profile": {
    "mass_kg":             1000.0,
    "thrust_force_n":      1.0e10,
    "max_speed_sim_units": 5.0,
    "superluminal_allowed": true
  }
}
```

| Parameter | Default | Notes |
|-----------|---------|-------|
| `mass_kg` | 1,000 kg | Test-particle mass; does not affect N-body force sums on bodies |
| `thrust_force_n` | 1 × 10¹⁰ N | Provides ~10⁷ m/s² acceleration (very fast; scale down for realistic feel) |
| `max_speed_sim_units` | 5.0 | ~500 AU/s; capped server-side to prevent position divergence |
| `superluminal_allowed` | true | Can be disabled per-role by server config |

Note on thrust scale: realistic chemical thrusters for a 1,000 kg ship produce ~10,000 N
(0.01 × 10⁶ m/s² → 0.01 m/s² acceleration). The default is intentionally heroic to make
cross-system travel tractable. Realistic profiles can be offered via named presets.

**F-033 migration path**: When F-033 Phase 1 lands, `ShipProfile` fields in `configs/app.json`
are replaced by a ship catalog lookup keyed by `default_ship_id`. The `mass_kg` and
`max_speed_sim_units` fields are read from `ShipDefinition`. The `thrust_force_n` field
becomes the active engine stage's `accel_max_ms2 * mass_kg`. Turning rate is read from
`ShipDefinition.turning.rate_deg_per_s` scaled by power availability.

---

## 9. Player as Ship

This section defines the Player Point-of-View (POV) as a first-person instantiation of a
ship definition. It is the integration contract between F-022, F-033, F-023, and F-021.

### 9.1 Concept

The camera's viewpoint IS the ship's cockpit. When the player moves the camera (yaw, pitch,
roll), they are rotating the ship's facing vector. When the player applies thrust, they
are firing the ship's engines in the direction the ship's nose points. The player cannot
decouple camera orientation from ship facing (first-person only; no chase-cam in Phase 1).

### 9.2 Ship instance ownership

Every PLAYER-role session owns exactly one `ShipInstance` (F-033). The instance is assigned
by the server at `RegisterClient` (F-020). The `ShipInstance` determines:

| Property | Source | Effect |
|----------|--------|--------|
| Thrust acceleration | `ActiveStage.accel_max_ms2 × mass_kg` | Max velocity delta per frame |
| Turn rate | `turning.rate_deg_per_s × power_fraction` | Max camera rotation per frame |
| Speed cap | `max_speed_sim_units_per_s` | Server-enforced hard cap |
| Power budget | `available_w - baseline_draw - active_engine_draw` | Overload degrades engine + turn |
| Damage | `engine_integrity × power_integrity` | Scales all capability ratings |

### 9.3 Input → ship → physics pipeline

```
[F-023 KeyMap]
  move.thrust_* / camera.yaw / camera.pitch / camera.roll
          ↓
[F-022 Movement Handler]
  1. Read active ShipInstance.ActiveStage for thrust_force_n
  2. Clamp dt*accel to stage max
  3. Apply engine_integrity multiplier
  4. Resolve power budget; apply overload policy if negative
  5. Apply velocity delta via kinematic integration
          ↓
[Server Session Registry]
  Position updated; broadcast in next WorldSnapshot
          ↓
[F-021 Physical Marker]
  3D model rendered at new position with FacingVector applied
```

### 9.4 Camera orientation is ship facing

`session.cameraState.Forward` (the camera's forward unit vector) is mirrored as
`ShipInstance.FacingVector`. On every input frame:

```
ShipInstance.FacingVector = session.cameraState.Forward
```

Thrust is applied along `FacingVector`. Strafe is the cross product of `FacingVector` and
the world up vector. Up/down thrust is along the ship's local up axis.

### 9.5 Camera modes and ship

| Camera mode | Ship behavior |
|-------------|---------------|
| Free-fly | Ship moves, camera IS the ship; turning keys rotate ship |
| Tracking | Ship is stationary; camera orbits the tracked body; ship facing does not update |
| Jumping | Ship is stationary; camera interpolates to target; ship facing frozen |

When the player exits tracking mode (ESC / `sim.track_stop`) and returns to free-fly, the
ship resumes its pre-tracking position and velocity unchanged.

### 9.6 Damage and power as movement limiters

Damage from F-027 collisions writes directly to `ShipInstance.HullIntegrity`,
`EngineIntegrity`, and `PowerIntegrity`. The movement handler reads these each frame:

- If `EngineIntegrity < 0.2`: all thrust disabled; drift mode forced.
- If `PowerIntegrity < 0.5`: engine stage limited to Stage 1.
- If `HullIntegrity == 0.0`: session disconnected; respawn flow triggered (Phase 2+).

### 9.7 Phases

**Phase 1 (within F-033 Phase 1 / F-022 Phase 1)**:
- `FacingVector` updated from camera state each frame
- Thrust applied along facing vector at `thrust_force_n` from stub `ShipProfile`
- No power budget calculation (F-033 Phase 1 delivers the capability data)

**Phase 2 (after F-033 Phase 1)**:
- Replace stub `ShipProfile` with `ShipInstance` from F-033 catalog
- Apply engine stage ratings, power budget, and damage multipliers
- Turn rate limited by `turning.rate_deg_per_s × power_fraction`

**Phase 3 (after F-033 Phase 3)**:
- Engine stage cycling via `move.engine_stage_up/down` keybindings
- Power HUD shows budget bar, stage label, integrity indicators

---

## 7. Files to Touch

### Phase 1 files (kinematic, no gravity)

| File | Action | Notes |
|------|--------|-------|
| `api/proto/spacesim/v1/movement.proto` | **Create** | `MovementService` with `SetDrift`, `SetThrust`, `Impulse`, `Warp` RPCs; `ThrustVector`, `ImpulseRequest`, `WarpRequest` messages |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` |
| `internal/transport/grpc/movement_handler.go` | **Create** | Handler impl; role guards on Warp; speed cap enforcement |
| `internal/transport/grpc/movement_handler_test.go` | **Create** | Role permission, speed cap, position delta tests via bufconn |
| `internal/server/session/registry.go` | **Modify** | Add `UpdateVelocity`, `UpdatePosition` (kinematic step) to interface; add `ShipState` (pos, vel) to `ClientSession` |
| `internal/server/session/registry_test.go` | **Modify** | Add kinematic update and max-speed-cap tests |
| `internal/client/commands/commands.go` | **Modify** | Add `MoveDriftCmd`, `MoveImpulseCmd`, `MoveWarpCmd` types |
| `internal/client/repl/repl.go` | **Modify** | Add `move drift`, `move impulse <vx> <vy> <vz>`, `move warp <body\|x y z>` dispatch |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Keyboard thrust input edges call `MovementService.SetThrust` RPC |
| `configs/app.json` | **Modify** | Add `"ship_profile"` block (mass, thrust_force, max_speed, superluminal_allowed) |

### Phase 2 files (gravity, requires F-013)

| File | Action | Notes |
|------|--------|-------|
| `internal/sim/engine/physics.go` | **Modify** | Add client session receive pass to N-body force loop; skip force-exert (test-particle) |
| `internal/sim/engine/physics_test.go` | **Modify** | Add stable-orbit test for client ship at Earth distance |
| `internal/server/session/registry.go` | **Modify** | Expose `Velocity [3]float64` for N-body integration; add `client_gravity_enabled` check |
| `configs/app.json` | **Modify** | Add `"client_gravity_enabled": false` (default until F-013 ships) |

---

## 8. Phases

### Phase 1 — Kinematic Movement (no gravity)

**Architectural layer**: Wire protocol (`api/proto/`), transport layer (`internal/transport/grpc/`), server session layer (`internal/server/session/`), REPL client, Raylib app layer.
**Prerequisites**: F-020 Phase 1 (session registry must exist); F-023 Phase 1 (keyboard bindings for thrust input).

**Value**: Clients can navigate the world using all four movement types. Gravity is disabled.

Work items:
- [ ] Add `MovementService` proto (Drift, SetThrust, Impulse, Warp RPCs)
- [ ] Add `movement_handler.go` in `internal/transport/grpc/`
- [ ] Server applies kinematic position update (velocity integration, no gravity)
- [ ] Warp validates role permission; position clamped to sim bounds
- [ ] REPL: `move drift`, `move impulse <vx> <vy> <vz>`, `move warp <body-name>`
- [ ] `ShipProfile` loaded from `configs/app.json` defaults
- [ ] Integration test: client moves from origin to 100 sim units from Sol

Acceptance criteria:
- All four movement types result in correct position changes in `list sessions` ✓
- Superluminal from non-player role returns permission error ✓
- Max speed cap enforced ✓

### Phase 2 — Gravity Integration (requires F-013)

**Architectural layer**: Engine layer (`internal/sim/engine/`), server session layer.
**Prerequisites**: Phase 1 complete; F-013 N-Body complete.

**Value**: Client ships feel gravitational pull from all named bodies.

Work items:
- [ ] Add client sessions to N-body receive pass in `internal/sim/engine/physics.go`
- [ ] `client_gravity_enabled` config flag respected
- [ ] Drift mode: ship falls toward nearest attractor correctly
- [ ] Validate stable circular orbit at Earth's distance over 1 simulated orbit
- [ ] Performance test: 100 clients in N-body receive pass; frame budget impact < 0.5 ms

Acceptance criteria:
- Ship released from rest at Earth distance falls toward Sol ✓
- Ship in circular orbit maintains orbit within 1% period error over 1 simulated year ✓
- N-body frame time increase with 100 clients < 0.5 ms ✓

### Phase 3 — NPC Automation (deferred)

Server-driven routine locomotion for NPC-role sessions. Patrol, orbit, intercept profiles.
Deferred pending Phase 2 validation.

**Note on AI-driven NPCs**: An external AI agent (e.g., MCP tool client) calling into the
REPL/gRPC interface to drive NPC sessions is a considered future direction. The REPL's
existing command dispatch and `MovementService` RPCs are the natural integration point.
No structural changes are needed before Phase 3; this remains a design note only.

---

## 8. Open Questions

| # | Question | Decision | Status |
|---|----------|----------|---------|
| Q1 | Thrust: continuous hold input or per-frame impulse from REPL? | TBD | Open — Phase 1 |
| Q2 | Should max speed vary by movement type (e.g., impulse capped lower than warp)? | TBD | Open — Phase 1 |
| Q3 | Default spawn position: near Sol, near origin, or configurable? | **Resolved**: random orbit around any named body (star, planet, dwarf planet, moon) or inside any asteroid belt. Picked by server at register time. | Closed |
| Q4 | Should NPC profiles be JSON-driven (data/npcs/) or code-only? | TBD | Open — Phase 3 |
| Q5 | Is there a "thrust-assist" mode that automatically counteracts drift to hold position? | TBD | Open — Phase 2 |
