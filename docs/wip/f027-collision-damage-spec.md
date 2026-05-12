# F-027 — Ship Collision Detection and Damage

## Purpose

Detect when client ships come into contact with each other or with named celestial bodies,
compute an impact severity from the relative velocity, apply a damage rating to the involved
sessions, and emit a durable `ImpactEvent` through the existing `protocol.Broadcaster`.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md) — architectural layers
- [`docs/standards/coding-standards.md`](../standards/coding-standards.md)
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — `DamageRating` field on `ClientSession`
- [`docs/wip/f022-client-movement-spec.md`](f022-client-movement-spec.md) — position and velocity sources
- [`docs/wip/f026-audio-events-spec.md`](f026-audio-events-spec.md) — audio triggered by `ImpactEvent`
- [`docs/wip/f025-ship-comms-spec.md`](f025-ship-comms-spec.md) — comms log posts system message on impact
- [`docs/history/lessons-learned-double-buffering.md`](../history/lessons-learned-double-buffering.md) — buffer safety

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Detect client-to-client and client-to-body overlap each simulation tick |
| G2 | Impact severity is proportional to relative speed at contact |
| G3 | `DamageRating` accumulates durably in the session registry; survives reconnects (when session persistence ships) |
| G4 | Destroyed ships respawn at a random spawn point after a cooldown |
| G5 | `ImpactEvent` is broadcast to all subscribers via `protocol.Broadcaster` |
| G6 | Camera POV never enters another object (reinforces F-001 / camera rules) |
| G7 | The collision detector runs in the server event loop, not the render loop |

---

## 2. Non-Goals (this feature)

- N-body force interactions between client ships and celestial bodies (test-particle model; F-022 §4.2)
- Weapon systems or intentional damage sources
- Detailed mesh/hull collision (bounding-sphere only in all phases)
- Client-to-asteroid-belt particle collision (belt particles are not tracked individually)
- Atmospheric entry drag (deferred)

---

## 3. Damage Model

### 3.1 DamageRating

`DamageRating float32` on `ClientSession`, range [0.0, 1.0]:

| Range | State | Effect |
|-------|-------|--------|
| 0.0 | Undamaged | Normal operation |
| 0.01 – 0.49 | Damaged | HUD warning; no functional impairment in Phase 1 |
| 0.50 – 0.74 | Heavily damaged | HUD warning; max thrust reduced 50% (Phase 2) |
| 0.75 – 0.99 | Critical | Audio alarm; thrust reduced 75% (Phase 2) |
| 1.0 | Destroyed | Session frozen; respawn timer starts |

### 3.2 Impact severity formula

```
relative_speed = |velocity_a - velocity_b|   (sim units / s)
severity = clamp(relative_speed / max_nonlethal_speed, 0.0, 1.0)
damage_delta = severity * damage_scale_factor
```

`max_nonlethal_speed` and `damage_scale_factor` are configurable in `configs/app.json`.
Defaults: `max_nonlethal_speed = 0.1` sim units/s, `damage_scale_factor = 0.25`.

At default settings: a low-speed nudge (0.01 su/s) adds 0.025 damage; a high-speed impact
(0.1+ su/s) adds 0.25 damage; four full-speed impacts destroy a ship.

### 3.3 Respawn

When `DamageRating >= 1.0`:
- Session state is set to `DESTROYED`.
- Position is frozen (no further movement processing).
- After `respawn_cooldown_seconds` (default 10 s, configurable), `DamageRating` resets to
  `0.0` and position is reset to a fresh random spawn point (same algorithm as initial spawn
  from F-020 §3.1).
- A `ClientEventType_RESPAWN` system message is posted to the comms log (F-025).

---

## 4. Collision Detection Algorithm

### 4.1 Bounding spheres

All client ships use a single bounding sphere. Radius is defined by the ship's
`ShipProfile.BoundingRadiusSimUnits` (default: `1e-5` sim units ≈ 1,500 m, intentionally
larger than true ship size to make collisions physically noticeable).

Named bodies already carry `Meta.PhysicalRadius` (stored as `float32`). The detector reads
`float64`-promoted copies from the N-body state when F-013 is active, otherwise uses
`float32` values directly.

### 4.2 Broad phase

Each tick, the detector builds a list of all (client-ship, client-ship) and
(client-ship, named-body) pairs that are within `broad_phase_radius_multiplier × (r_a + r_b)`
of each other. Default multiplier: `4.0` (generous; filters obviously distant pairs cheaply).

Complexity: O(N_clients² + N_clients × N_bodies). At 100 clients and 200 named bodies:
- Client-client pairs: 4,950
- Client-body pairs: 20,000
- Total comparisons: ~25,000 float64 distance checks per tick

At 60 Hz this is 1.5M comparisons/s. On an M1 this is sub-millisecond. No spatial
partitioning is needed in Phase 1.

### 4.3 Narrow phase

For each candidate pair that passes the broad phase:
```
overlap = (r_a + r_b) - distance(pos_a, pos_b)
if overlap > 0:
    generate ImpactEvent
```

### 4.4 Cooldown

A per-pair collision cooldown of `impact_cooldown_frames` (default: 60 frames = 1 s at
60 Hz) prevents the same pair generating continuous impact events while overlapping.
Cooldown state is stored in a `map[[2]string]int` (keyed by sorted SessionID pair); the map
is cleared on unregister.

---

## 5. ImpactEvent

```proto
message ImpactEvent {
  string session_id_a  = 1;  // client SessionID, or empty for a named body
  string session_id_b  = 2;  // client SessionID, or empty for a named body
  string body_id_a     = 3;  // body GUID if session_id_a is empty
  string body_id_b     = 4;  // body GUID if session_id_b is empty
  float  severity      = 5;  // 0.0–1.0
  float  damage_delta  = 6;  // damage applied this event
  int64  occurred_at   = 7;  // server nanosecond timestamp
}
```

The event is pushed to `protocol.Broadcaster`. All registered subscribers (the Raylib
renderer, the comms fan-out, the audio trigger, the event log) receive it.

---

## 6. Files to Touch

| File | Action | Notes |
|------|--------|-------|
| `internal/server/collision/detector.go` | **Create** | `CollisionDetector` type; broad + narrow phase; cooldown map |
| `internal/server/collision/detector_test.go` | **Create** | Overlap math, cooldown logic, broad-phase filter |
| `internal/server/session/registry.go` | **Modify** | Add `UpdateDamage(sessionID string, delta float32) error`; add `DESTROYED` state |
| `internal/server/session/registry_test.go` | **Modify** | Test damage accumulation; destroyed state; respawn reset |
| `internal/server/routines/` | **Modify** | Register collision detector as a per-frame routine |
| `api/proto/spacesim/v1/simulation.proto` | **Modify** | Add `ImpactEvent` message |
| `api/gen/spacesim/v1/` | **Regenerate** | `make proto` after proto change |
| `internal/protocol/` | **Modify** | Add `ImpactEvent` to broadcaster payload type |
| `internal/transport/grpc/session_handler.go` | **Modify** | Subscribe to broadcaster; fan out `ImpactEvent` to affected sessions' streams |
| `internal/client/go/raylib/app/app.go` | **Modify** | Handle incoming `ImpactEvent`; update local damage display; trigger audio (F-026) |
| `configs/app.json` | **Modify** | Add `collision` config block |

---

## 7. Config Block

```json
"collision": {
  "max_nonlethal_speed":      0.1,
  "damage_scale_factor":      0.25,
  "respawn_cooldown_seconds": 10,
  "broad_phase_multiplier":   4.0,
  "impact_cooldown_frames":   60,
  "bounding_radius_sim_units": 1e-5
}
```

---

## 8. Camera Constraint: POV Never Inside Objects

**This is a hard rule, not a feature option.** The camera POV must never enter another
client ship or named body. This is enforced by two independent mechanisms:

1. **F-001 (Camera Collision Prevention)** — prevents the camera from entering named bodies.
   Collision detection adds client ships to the same camera-exclusion list.
2. **Render-time guard** — if the camera center is within a bounding sphere at render time,
   `DrawSphere` / `DrawModel` for that object is skipped (object is "behind" the camera).

The F-001 work must be complete or in-progress before Phase 2 of this feature is considered done.

---

## 9. Phases

### Phase 1 — Client-to-Client Detection and Damage

**Architectural layer**: Server runtime layer (`internal/server/collision/`) + session layer
(`internal/server/session/`) + protocol broadcast layer.  
**No Raylib changes in this phase.**

**Prerequisites**: F-020 Phase 1 (session registry); F-022 Phase 1 (position tracking in registry).

**Work items**:
- [ ] Create `internal/server/collision/` package
- [ ] Implement `CollisionDetector.Tick(sessions []*ClientSession)` — client-to-client only
- [ ] Implement bounding sphere overlap test with cooldown map
- [ ] Add `UpdateDamage` to `Registry`; add `DESTROYED` session state; respawn timer
- [ ] Add `ImpactEvent` proto message; `make proto`
- [ ] Register `CollisionDetector` as a server routine
- [ ] Fan out `ImpactEvent` to affected sessions via `SessionStream` (transport layer)
- [ ] Post system comms message on impact (F-025 Phase 1 dependency)
- [ ] Unit tests: overlap math, cooldown, damage accumulation, destroyed→respawn
- [ ] Integration test: two clients in overlapping positions → `ImpactEvent` delivered

**Acceptance criteria**:
- Two sessions with overlapping bounding spheres produce an `ImpactEvent` within one tick ✓
- `DamageRating` increments correctly per event ✓
- Impact cooldown prevents duplicate events within 60 frames ✓
- Session with `DamageRating == 1.0` respawns after cooldown ✓
- Race detector passes ✓

### Phase 2 — Client-to-Body Detection

**Prerequisites**: Phase 1 complete; F-001 (camera collision) complete.

**Work items**:
- [ ] Extend `CollisionDetector.Tick` to include named bodies from `WorldSnapshot`
- [ ] Promote `float32` body radii to `float64` for comparison; use N-body positions when available
- [ ] Client camera excluded from entering body (hook into F-001 camera guard)
- [ ] Thrust reduction applied when `DamageRating >= 0.5` (F-022 hook)
- [ ] Unit tests: client-body overlap detection

**Acceptance criteria**:
- Ship flying into a planet at speed produces `ImpactEvent` with correct `body_id_b` ✓
- Camera never enters the planet after impact ✓

### Phase 3 — Damage Effects on Movement (deferred)

Thrust reduction when damaged. Requires F-022 to expose thrust-cap API.
Deferred until Phase 2 is validated.

---

## 10. Dependencies

| Feature | Relationship |
|---------|-------------|
| F-020: Session | `DamageRating` lives on `ClientSession`; `UpdateDamage` on `Registry` |
| F-022: Movement | Position/velocity source for collision check; thrust cap in Phase 3 |
| F-025: Comms | System message posted on impact |
| F-026: Audio | `ImpactEvent` triggers `AudioEventCollisionImpact` |
| F-001: Camera Collision | Camera exclusion list extended with client ships |
| F-013: N-Body | float64 body positions used in Phase 2 when available |

---

## 11. Open Questions

| # | Question | Status |
|---|----------|--------|
| Q1 | Should `DamageRating` persist to disk (autosave subscriber)? | Open — Phase 1 |
| Q2 | Should friendly fire be enabled by default, or configurable in `configs/app.json`? | Open — Phase 1 |
| Q3 | Respawn: random body orbit, or last-safe position before collision? | Open — Phase 1 |
| Q4 | Should admin-role sessions be immune to damage? | Open — Phase 1 |
