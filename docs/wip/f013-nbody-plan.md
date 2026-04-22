# F-013 — N-Body Barycenter Integration: Implementation Plan

## Purpose

Detailed implementation plan for F-013. Read this before writing any code.
Reference alongside `docs/standards/agent-readme.md` and `docs/standards/coding-standards.md`.

## Last Updated
2026-04-21

## Status
📋 Not started

---

## 1. Goal

Replace the current single-parent Keplerian approximation with leapfrog N-body
gravitational integration for all named bodies (stars, planets, dwarf planets,
moons). Belt particles (asteroids, Kuiper belt) remain Keplerian — full N-body
at tens of thousands of objects is prohibitive.

The renderer and all downstream consumers keep reading `float32` positions from
`Anim.Position`. N-body precision (float64) stays entirely in the physics layer.

---

## 2. Locked Decisions

| Topic | Decision | Rationale |
|-------|----------|-----------|
| Integrator | **Leapfrog DKD** (Drift-Kick-Drift) | Symplectic; conserves orbital energy long-term; one force evaluation per step; standard for solar-system N-body |
| Force computation | **O(N²) brute force** | ~100 named bodies at 60 Hz is ≪ 1 ms on M1; Barnes-Hut not needed at solar-system scale |
| Scope | **Named bodies only** — `obj.Dataset == -1` | Belt/ring particles (Dataset ≥ 0) stay on the Keplerian path |
| Precision | **`[3]float64` N-body fields added to `AnimationState`**; renderer reads float32 copy | Avoids global float32 migration; precision lives in physics only |
| GM source | `G_sim × obj.Meta.Mass`, computed at load time, stored as `GM float64` on `ObjectMetadata` | Mass already present in JSON for all bodies |
| Initialization | Keplerian state seeds N-body starting positions; vis-viva with correct parent GM seeds velocities | JSON schema unchanged; no new loader fields |
| Belt objects | Keep existing `updateObject` Keplerian path unchanged | Correctness and performance; full N-body for belts is out of scope |
| Barycenter output | Single `SystemBarycenter Vector3` on `SimulationState`; per-group barycenters deferred | Sufficient for camera-track-barycenter in multi-star systems |
| Opt-in flag | None in this phase — N-body becomes the model for named bodies | Can add `--keplerian` debug flag later if regression comparison needed |

---

## 3. Simulation Unit System

The physics simulation uses a custom unit system:

- **1 sim unit = 1 AU / 100** (Earth's orbital radius is 100 sim units)
- **Time unit**: seconds (matches `OrbitalPeriod` in seconds in the JSON)
- **Mass unit**: kilograms (raw SI)

The gravitational constant in sim units is derived as:

```
G_SI  = 6.674e-11 m³ / (kg·s²)
1 AU  = 1.496e11 m
1 sim = 1 AU / 100 = 1.496e9 m

G_sim = G_SI / (1.496e9)³
      = 6.674e-11 / 3.351e27
      ≈ 1.991e-38   sim³ / (kg·s²)
```

Use `G_sim = 1.991e-38` as the gravitational constant.

Sanity check: Earth's orbital period from N-body should be ~3.156e7 seconds (1 year).
Sol GM_sim ≈ 1.991e-38 × 1.989e30 ≈ 3.96e-8.
Earth orbit radius ≈ 100 sim units.
T = 2π √(a³/GM_sim) = 2π √(1e6 / 3.96e-8) ≈ 2π × 5.02e6 ≈ 3.15e7 s ✓

---

## 4. Files to Touch

| File | Change |
|------|--------|
| `internal/sim/engine/constants.go` | Add `G_sim`, `NBodyNamedBodyCategory` helper (or just use Dataset == -1) |
| `internal/sim/engine/object.go` | Add `NBodyPos, NBodyVel, NBodyAcc [3]float64` to `AnimationState`; add `GM float64` to `ObjectMetadata` |
| `internal/sim/engine/state.go` | Add `SystemBarycenter Vector3` to `SimulationState`; compute in a new `updateBarycenter()` helper |
| `internal/sim/engine/physics.go` | Main work — see Phase 3–6 below |
| `internal/sim/loader.go` | Populate `GM = G_sim × Mass` for each body in `createBodyFromConfig` |

No renderer changes. No schema changes. No proto changes.

---

## 5. Implementation Phases

### Phase 1 — Constants & type extensions

**`engine/constants.go`**
```go
// G_sim is the gravitational constant in simulation units.
// 1 sim unit = 1 AU/100 = 1.496e9 m.  G_SI / (1.496e9)^3 ≈ 1.991e-38.
const G_sim = 1.991e-38 // sim³ / (kg·s²)
```

**`engine/object.go` — `AnimationState`** (add after existing fields):
```go
// N-body integration state (float64 for physics precision).
// NBodyPos/Vel/Acc are in sim-unit world space.
// Renderer reads the float32 Position copy updated each tick.
NBodyPos [3]float64
NBodyVel [3]float64
NBodyAcc [3]float64
```

**`engine/object.go` — `ObjectMetadata`** (add after `Mass`):
```go
GM float64 // Gravitational parameter G × Mass in sim³/s² (computed at load)
```

**`engine/state.go` — `SimulationState`** (add after `DeltaTime`):
```go
SystemBarycenter Vector3 // Mass-weighted center of all named bodies
```

### Phase 2 — GM population in loader

In `internal/sim/loader.go`, in `createBodyFromConfig` after `Mass` is assigned:
```go
obj.Meta.GM = G_sim * obj.Meta.Mass
```

Import `"github.com/user/space_sim/internal/sim/engine"` is already present.

### Phase 3 — N-body initialization

Add to `physics.go`, called once from `NewSimulation` after mean-anomaly priming:

```go
// initNBody seeds N-body state from Keplerian positions and velocities.
// Must be called after all objects have valid Anim.Position.
func initNBody(state *SimulationState) {
    for _, obj := range state.Objects {
        if obj.Dataset != -1 { continue } // belt particles stay Keplerian
        obj.NBodyPos = [3]float64{
            float64(obj.Anim.Position.X),
            float64(obj.Anim.Position.Y),
            float64(obj.Anim.Position.Z),
        }
        // Velocity: copy Anim.Velocity (already computed by Keplerian path in
        // updateObject via vis-viva), then add parent velocity for moons.
        obj.NBodyVel = [3]float64{
            float64(obj.Anim.Velocity.X),
            float64(obj.Anim.Velocity.Y),
            float64(obj.Anim.Velocity.Z),
        }
        if obj.Meta.ParentName != "" {
            if parent := state.ObjectMap[obj.Meta.ParentName]; parent != nil {
                obj.NBodyVel[0] += float64(parent.Anim.Velocity.X)
                obj.NBodyVel[1] += float64(parent.Anim.Velocity.Y)
                obj.NBodyVel[2] += float64(parent.Anim.Velocity.Z)
            }
        }
    }
}
```

**Important**: The Keplerian `updateObject` uses `GM = 1.0` (normalized units) for
the vis-viva velocity. When seeding N-body we override the velocity anyway on the
first leapfrog kick, so small initialization error is acceptable — it self-corrects
within one orbital period. If exact epoch-accurate seeding is needed later, compute
velocity analytically from Keplerian elements + real GM_parent.

### Phase 4 — Force accumulator

```go
// accumForces computes pairwise gravitational acceleration for all named bodies.
// namedBodies must be a pre-filtered slice (Dataset == -1).
// Thread-safe: each goroutine reads all positions and writes only its own Acc.
func accumForces(namedBodies []*Object) {
    // Zero accelerations
    for _, a := range namedBodies {
        a.Anim.NBodyAcc = [3]float64{}
    }
    // Pairwise O(N²)
    for i, a := range namedBodies {
        for j := i + 1; j < len(namedBodies); j++ {
            b := namedBodies[j]
            dx := b.Anim.NBodyPos[0] - a.Anim.NBodyPos[0]
            dy := b.Anim.NBodyPos[1] - a.Anim.NBodyPos[1]
            dz := b.Anim.NBodyPos[2] - a.Anim.NBodyPos[2]
            r2 := dx*dx + dy*dy + dz*dz
            if r2 < 1e-6 { continue } // softening: skip if coincident
            r3 := r2 * math.Sqrt(r2)
            // a ← a + GM_b / r³ * Δr
            a.Anim.NBodyAcc[0] += b.Meta.GM / r3 * dx
            a.Anim.NBodyAcc[1] += b.Meta.GM / r3 * dy
            a.Anim.NBodyAcc[2] += b.Meta.GM / r3 * dz
            // b ← b - GM_a / r³ * Δr  (Newton's third law)
            b.Anim.NBodyAcc[0] -= a.Meta.GM / r3 * dx
            b.Anim.NBodyAcc[1] -= a.Meta.GM / r3 * dy
            b.Anim.NBodyAcc[2] -= a.Meta.GM / r3 * dz
        }
    }
}
```

Parallelization: With ~100 named bodies, the pairwise loop is ~5 000 operations —
no goroutine overhead worth adding. Add parallel outer loop only if profiling shows
it matters at higher body counts.

### Phase 5 — Leapfrog DKD integration and update loop

Leapfrog DKD for one timestep `dt`:
1. **Drift** half-step: `pos += vel × dt/2`
2. **Kick** full-step: compute forces at new positions; `vel += acc × dt`
3. **Drift** half-step: `pos += vel × dt/2`

```go
func stepNBody(namedBodies []*Object, dt float64) {
    h := dt / 2.0
    // Drift ½
    for _, obj := range namedBodies {
        obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
        obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
        obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
    }
    // Kick
    accumForces(namedBodies)
    for _, obj := range namedBodies {
        obj.Anim.NBodyVel[0] += obj.Anim.NBodyAcc[0] * dt
        obj.Anim.NBodyVel[1] += obj.Anim.NBodyAcc[1] * dt
        obj.Anim.NBodyVel[2] += obj.Anim.NBodyAcc[2] * dt
    }
    // Drift ½
    for _, obj := range namedBodies {
        obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
        obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
        obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
    }
    // Copy float64 positions back to float32 for renderer
    for _, obj := range namedBodies {
        obj.Anim.Position.X = float32(obj.Anim.NBodyPos[0])
        obj.Anim.Position.Y = float32(obj.Anim.NBodyPos[1])
        obj.Anim.Position.Z = float32(obj.Anim.NBodyPos[2])
        // Also update Velocity for Keplerian children (moons use parent velocity)
        obj.Anim.Velocity.X = float32(obj.Anim.NBodyVel[0])
        obj.Anim.Velocity.Y = float32(obj.Anim.NBodyVel[1])
        obj.Anim.Velocity.Z = float32(obj.Anim.NBodyVel[2])
    }
}
```

**In `update()`**, replace the parent update section:
```go
// Before: parallel Keplerian update for parents
// After:
namedBodies := collectNamed(back) // pre-filtered slice, re-use or cache
stepNBody(namedBodies, scaledDt)
```

Children (moons) still need their `OrbitCenter` updated from parent position before
the Keplerian child pass. The existing child loop already does this:
```go
for _, obj := range children {
    if parent := back.ObjectMap[obj.Meta.ParentName]; parent != nil {
        obj.Anim.OrbitCenter = parent.Anim.Position
    }
}
```
This remains unchanged. Moons are children with a `ParentName`; they will also be
in `namedBodies` (Dataset == -1), so their N-body integration adds the
gravitational pull of all other named bodies on top of the Keplerian moon orbit.
The Keplerian child path runs *after* N-body for belt particles only.

**`collectNamed`** helper:
```go
func collectNamed(state *SimulationState) []*Object {
    named := make([]*Object, 0, 32)
    for _, obj := range state.Objects {
        if obj.Dataset == -1 {
            named = append(named, obj)
        }
    }
    return named
}
```

Consider caching this slice on `SimulationState` alongside `parents`/`children`
to avoid allocation every tick. Follow the same `dirty` flag pattern.

### Phase 6 — Barycenter computation

Add to `SimulationState` or compute inline in `update()` after `stepNBody`:

```go
func updateBarycenter(state *SimulationState, namedBodies []*Object) {
    var cx, cy, cz, totalMass float64
    for _, obj := range namedBodies {
        m := obj.Meta.Mass
        cx += m * obj.Anim.NBodyPos[0]
        cy += m * obj.Anim.NBodyPos[1]
        cz += m * obj.Anim.NBodyPos[2]
        totalMass += m
    }
    if totalMass > 0 {
        state.SystemBarycenter = Vector3{
            X: float32(cx / totalMass),
            Y: float32(cy / totalMass),
            Z: float32(cz / totalMass),
        }
    }
}
```

---

## 6. Double-Buffer / Clone Impact

`NBodyPos`, `NBodyVel`, `NBodyAcc` are fields on `AnimationState`, which is
value-copied in `Clone()` and `CloneWithPool()`. No additional clone logic needed
— the back-buffer runs integration; the front-buffer gets a snapshot copy each
tick via `Swap()`.

`SystemBarycenter` is a `Vector3` on `SimulationState`; it is already copied by
the `clone()` method's explicit field list. **Add it to the clone struct literal**
in `state.go`:
```go
SystemBarycenter: s.SystemBarycenter,
```

---

## 7. Tests & Validation

### Automated (add to `engine/` package tests)

1. **Earth orbit period**: run N-body at sim speed for 1 simulated year; verify
   Earth returns to within 5° of starting true anomaly.
2. **Energy conservation**: total kinetic + potential energy for a two-body
   Sun+Earth system should vary < 0.1% over 10 years of simulated time.
3. **Moon orbit**: Earth-Moon distance should stay within 35–42 sim units over
   1 simulated month.
4. **Barycenter consistency**: Sol-Earth barycenter should be inside Sol
   (< 27.25 sim units from origin).

### Visual (manual, `make run`)

1. Pause sim, note Earth position. Let run for ~1 sim year at high speed. Earth
   should return to approximately the same position.
2. Jump to Earth-Moon. Moon should orbit Earth visually as before (not diverge).
3. Jump to Sol. Sol should remain at or very near origin (within barycenter drift
   — Solar System barycenter shifts ~1 solar radius due to Jupiter).

### Regression

- `make test` passes with `-race` before and after.
- No new import cycles.
- `space-sim-direct` builds and runs without segfault.

---

## 8. Known Risks

| Risk | Mitigation |
|------|-----------|
| Moon orbits destabilize due to initialization velocity error | Moon Keplerian elements give correct relative velocity; adding parent velocity in `initNBody` is the key step. Verify visually in first run. |
| Tick-rate too coarse for inner-planet accuracy | At 60 Hz sim tick, dt ≈ 0.0167 s real-time × speed multiplier. At speed = 1e6 (fast-forward), dt_sim = 16 700 s ≈ 4.6 hours. Leapfrog error grows as dt². At 4.6 h steps, Mercury's perihelion will drift. Acceptable for visualization; note in lessons-learned. |
| `collectNamed` allocates every tick | Cache the slice on `SimulationState` with the existing `dirty` flag pattern; preallocate at load time. |
| N-body applied to rings (`CategoryRing`) | Rings have `Dataset == -1` and `InnerRadius > 0`. Exclude them from `namedBodies` by checking `obj.Meta.Category != CategoryRing` or by checking `OrbitalPeriod == 0`. |

---

## 9. Ordering Within F-013

1. Phase 1 (constants + types) → Phase 2 (loader GM) → `make test`
2. Phase 3 (init) → Phase 4 (forces) → Phase 5 (integration, no-op first run)
3. Phase 6 (barycenter) → visual validation → `make test`
4. Record results in `docs/history/lessons-learned.md`

---

## 10. Related Documents

- [docs/wip/todo.md](todo.md) — F-013 work items and acceptance criteria
- [docs/wip/roadmap.md](roadmap.md) — Step 1 in the implementation sequence
- [internal/sim/engine/physics.go](../../internal/sim/engine/physics.go) — current Keplerian physics
- [internal/sim/engine/object.go](../../internal/sim/engine/object.go) — `ObjectMetadata`, `AnimationState`
- [internal/sim/engine/state.go](../../internal/sim/engine/state.go) — `SimulationState`, double-buffer
- [docs/history/lessons-learned-double-buffering.md](../history/lessons-learned-double-buffering.md) — concurrency invariants to preserve
