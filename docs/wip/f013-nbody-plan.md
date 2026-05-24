# F-013 — N-Body Gravitational Integration: Implementation Plan

## Purpose

Detailed implementation plan for F-013. Read this before writing any code.
Reference alongside `docs/standards/agent-readme.md` and `docs/standards/coding-standards.md`.

## Last Updated
2026-05-24

## Status
📋 Not started — spec complete; implementation paused pending capture of other in-flight work

---

## 1. Goal

Replace the current single-parent Keplerian approximation with leapfrog N-body
gravitational integration over **configurable gravitational sets**. A gravitational
set (`GravSet`) is an explicit list of bodies that interact gravitationally with
each other. Different sets cover different physical contexts:

- All stars and planets in a system (global system dynamics)
- A planet and its moons (local cluster)
- All bodies within a sphere of influence (SOI) of a chosen center
- Asteroids that have entered a planet's SOI
- Artifacts (space stations, large ships, dwarf planets in context)
- Player ships and NPC ships as test particles in any of the above

Belt particles (tens of thousands of asteroids, Kuiper belt objects) **remain
Keplerian by default** — full N-body at that scale is prohibitive. Individual
belt members near a planet's SOI can be promoted to a GravSet on demand.

The renderer and all downstream consumers keep reading `float32` positions from
`Anim.Position`. N-body precision (`float64`) stays entirely in the physics layer.

---

## 2. Locked Decisions

| Topic | Decision | Rationale |
|-------|----------|-----------|
| Integrator | **Leapfrog DKD** (Drift-Kick-Drift) | Symplectic; conserves orbital energy long-term; one force evaluation per step; standard for solar-system N-body |
| Force computation | **O(N²) brute force per set** | Each set is small (≤ a few hundred bodies); Barnes-Hut not needed at this scale |
| Default scope | **All named bodies** (`Dataset == -1`, non-ring) as the default `SystemSet` | Backwards-compatible starting point; specialized sets compose on top |
| GravSet API | **`GravSet` struct with `Participants` + `TestParticles` slices** | Clean separation between bodies that exert gravity and those that only receive it |
| Test particles | **Ships and small bodies receive forces but do not exert them on Participants** | Standard test-particle approximation; negligible mass effect; avoids O(N²) explosion from thousands of ships |
| Precision | **`[3]float64` N-body fields added to `AnimationState`**; renderer reads float32 copy | Avoids global float32 migration; precision lives in physics only |
| GM source | `G_sim × obj.Meta.Mass`, computed at load time, stored as `GM float64` on `ObjectMetadata` | Mass already present in JSON for all bodies |
| SOI radius | **Hill sphere** stored as `HillRadius float64` on `ObjectMetadata`, computed at load | Used for set membership queries; switches to Laplace SOI formula for moons |
| Initialization | Keplerian state seeds N-body starting positions; vis-viva with correct parent GM seeds velocities | JSON schema unchanged; no new loader fields |
| Belt objects | Keplerian by default; promoted to a GravSet's `TestParticles` list when inside a body's SOI | Correctness for normal operation; per-SOI promotion for realism near planets |
| Artifacts | New `CategoryArtifact` category; eligible as `Participants` in a GravSet if mass is defined | Covers stations, large ships, dwarf planets in non-standard contexts |
| SOI crossing | `SOITracker` checks each tick whether a body has entered or exited a SOI and updates set membership | Dynamic but bounded: only checked for bodies near SOI boundaries |
| Barycenter output | Single `SystemBarycenter Vector3` on `SimulationState`; per-group barycenters deferred | Sufficient for camera-track-barycenter in multi-star systems |

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

## 4. Gravitational Sets

A `GravSet` is the unit of N-body computation. Each set runs an independent
leapfrog step. Multiple sets can be active simultaneously.

### 4.1 GravSet Structure

```go
// GravSet is a configurable set of bodies to integrate under mutual gravity.
// Participants mutually attract. TestParticles receive gravity from all
// Participants but are too small to exert meaningful force in return.
type GravSet struct {
    Name         string     // human label for debug/logging
    Participants  []*Object  // full mutual interaction (stars, planets, moons, artifacts)
    TestParticles []*Object  // receive forces only (ships, asteroids near SOI)
}
```

### 4.2 Predefined Set Types

| Set Type | Participants | TestParticles | Use Case |
|----------|-------------|---------------|----------|
| `SystemSet` | All named bodies (stars, planets, dwarf planets, moons) in the system | Player ships registered in the system | System-wide dynamics; default on load |
| `LocalSet(planet)` | The planet + its direct moons | Player ships within the planet's Hill sphere | Moon navigation; landing approach |
| `SOISet(center, radius)` | All named bodies within `radius` sim units of `center` | Ships and asteroids inside the SOI | Custom sphere of influence |
| `ArtifactSet(planet)` | The planet + its artifacts (stations, large ships) | Player ships near the artifact | Station approach; orbital insertion |
| `UserSet(bodies...)` | Caller-specified bodies | Caller-specified test particles | Scripted scenarios; custom subsets |

### 4.3 Category-Based Collectors

The collection layer is separated from the integration layer. **Collectors** return
`[]*Object` slices filtered by various criteria. **Set builders** compose those slices
into a `GravSet`. The integrator (`stepGravSet`) knows nothing about categories.

This decoupling means:
- Adding a new body category never touches the integrator.
- Any combination of categories, SOI bounds, or names can form a set.
- Callers can compose sets from multiple collector calls before passing to the integrator.

#### Collectors — return `[]*Object`

```go
// CollectByCategory returns all objects in state matching any of the given categories.
// Excludes objects with Dataset >= 0 (belt particles) unless CategoryAsteroid is
// explicitly requested. Excludes CategoryRing always.
func CollectByCategory(state *SimulationState, cats ...ObjectCategory) []*Object

// CollectInSOI returns all objects (named and belt) whose NBodyPos is within
// radius sim units of center. Named bodies are candidates for Participants;
// belt members are candidates for TestParticles.
func CollectInSOI(state *SimulationState, center Vector3, radius float64) []*Object

// CollectInHillSphere returns all objects inside body.Meta.HillRadius of body.
// Convenience wrapper around CollectInSOI using body.Anim.NBodyPos as center.
func CollectInHillSphere(state *SimulationState, body *Object) []*Object

// CollectChildren returns all objects whose Meta.ParentName == parent.Meta.Name.
// Includes direct children only (moons of a planet, not moons-of-moons).
func CollectChildren(state *SimulationState, parent *Object) []*Object

// CollectByName returns the named objects from state.ObjectMap.
// Missing names are silently skipped.
func CollectByName(state *SimulationState, names ...string) []*Object
```

#### Set Builders — compose collectors into a `GravSet`

The predefined builders cover the common use cases and serve as reference
implementations for custom compositions.

```go
// SystemSet builds the default system-wide GravSet:
// Participants = all named non-ring bodies (stars, planets, dwarf planets,
//               moons, artifacts).
// TestParticles = provided ship particles.
func SystemSet(state *SimulationState, ships []*ShipParticle) GravSet {
    return GravSet{
        Name: "system",
        Participants: CollectByCategory(state,
            CategoryStar, CategoryPlanet, CategoryDwarfPlanet,
            CategoryMoon, CategoryArtifact,
        ),
        TestParticles: shipObjectsFrom(ships),
    }
}

// LocalSet builds a planet-centric GravSet:
// Participants = the planet + its direct moons + artifacts in its Hill sphere.
// TestParticles = ships inside planet's Hill sphere.
func LocalSet(state *SimulationState, planet *Object, ships []*ShipParticle) GravSet {
    return GravSet{
        Name: planet.Meta.Name + "/local",
        Participants: append(
            CollectByName(state, planet.Meta.Name),
            CollectChildren(state, planet)...,
            filterCategory(CollectInHillSphere(state, planet), CategoryArtifact)...,
        ),
        TestParticles: filterInHillSphere(shipObjectsFrom(ships), planet),
    }
}

// SOISet builds a spatial GravSet from an explicit center and radius.
// Named bodies inside the sphere become Participants; belt members and ships
// inside the sphere become TestParticles.
func SOISet(state *SimulationState, center Vector3, radius float64, ships []*ShipParticle) GravSet {
    inside := CollectInSOI(state, center, radius)
    return GravSet{
        Name: fmt.Sprintf("soi(r=%.1f)", radius),
        Participants:  filterNamed(inside),
        TestParticles: append(filterBelt(inside), filterInSOI(shipObjectsFrom(ships), center, radius)...),
    }
}
```

#### Custom composition example

Any combination of collectors can be passed directly to `GravSet{}`:

```go
// Custom: Earth + Moon + ISS artifact, no ships.
gs := GravSet{
    Name: "earth-moon-iss",
    Participants: append(
        CollectByName(state, "Earth", "Moon"),
        CollectByCategory(state, CategoryArtifact)...,
    ),
}
stepGravSet(gs, dt)
```

The integrator (`stepGravSet`) receives a fully assembled `GravSet` and has no
knowledge of how it was built.

#### Internal helpers (unexported)

```go
filterCategory(objs []*Object, cat ObjectCategory) []*Object
filterNamed(objs []*Object) []*Object            // Dataset == -1, non-ring
filterBelt(objs []*Object) []*Object             // Dataset >= 0
filterInHillSphere(objs []*Object, body *Object) []*Object
filterInSOI(objs []*Object, center Vector3, r float64) []*Object
shipObjectsFrom(ships []*ShipParticle) []*Object // bridge ShipParticle → Object
```

### 4.4 Test Particle Force Integration

Test particles are integrated by summing forces from all Participants only.
They do not contribute to the pairwise Participant force sums.

```go
// accumTestParticleForces accumulates gravity on each test particle from
// all Participants. Called after accumForces (which handles Participants).
func accumTestParticleForces(gs GravSet) {
    for _, tp := range gs.TestParticles {
        tp.Anim.NBodyAcc = [3]float64{}
        for _, p := range gs.Participants {
            dx := p.Anim.NBodyPos[0] - tp.Anim.NBodyPos[0]
            dy := p.Anim.NBodyPos[1] - tp.Anim.NBodyPos[1]
            dz := p.Anim.NBodyPos[2] - tp.Anim.NBodyPos[2]
            r2 := dx*dx + dy*dy + dz*dz
            if r2 < 1e-6 { continue }
            r3 := r2 * math.Sqrt(r2)
            tp.Anim.NBodyAcc[0] += p.Meta.GM / r3 * dx
            tp.Anim.NBodyAcc[1] += p.Meta.GM / r3 * dy
            tp.Anim.NBodyAcc[2] += p.Meta.GM / r3 * dz
        }
    }
}
```

---

## 5. Sphere of Influence (SOI)

### 5.1 Hill Sphere (primary SOI definition)

The Hill sphere is the region around a body where its gravity dominates over
the parent body's tidal forces. This is the boundary for set membership in
`LocalSet` and SOI-based promotions.

```
r_Hill = a × (m / 3M)^(1/3)
```

Where:
- `a` = semi-major axis of the body's orbit around its parent (sim units)
- `m` = body mass (kg)
- `M` = parent mass (kg)

For top-level stars (no parent), `r_Hill` is set to the system radius or a
configured cap (e.g., the system's outer boundary).

### 5.2 Laplace SOI (alternative for moon entry detection)

The Laplace SOI is smaller and more precise for entry/exit detection:

```
r_SOI = a × (m / M)^(2/5)
```

Use `r_Hill` for set membership containment, `r_SOI` for transition trigger
(when a body enters or exits gravitational influence).

### 5.3 SOI Radius on ObjectMetadata

```go
// ObjectMetadata additions
HillRadius  float64 // Hill sphere radius in sim units; 0 = use system default
LaplaceSOI  float64 // Laplace SOI radius in sim units; 0 = use HillRadius
```

Both are computed at load time in the loader from mass, semi-major axis, and
parent mass. No JSON schema changes required.

### 5.4 SOI Tracker — Dynamic Set Membership

The `SOITracker` runs once per N physics ticks (configurable; default every 60
ticks = ~1 real second at 60 Hz). It checks whether test particles and belt
members have crossed SOI boundaries and updates set membership accordingly.

Entry/exit events:
- **Enter**: body's distance to parent drops below `r_SOI` → added to `LocalSet` or `SOISet`
- **Exit**: distance exceeds `r_Hill` → removed from the local set, remains in `SystemSet`

This is the "elements moving into, through and out of sphere of influences" use case.

---

## 6. Artifact Category

`CategoryArtifact` is a new `ObjectCategory` for human-constructed or non-natural
objects with meaningful mass and gravitational presence: space stations, large
capital ships, derelict megastructures.

```go
CategoryArtifact ObjectCategory = 7  // Space stations, large ships, megastructures
```

Artifacts differ from player ships in that:
- They have a defined mass (JSON `mass_kg` field) and thus a `GM` and `HillRadius`
- They are eligible as `Participants` in a `GravSet` (not just `TestParticles`)
- They have a persistent world position stored in the simulation state (not in session registry)
- They are loaded from a data file, not registered as client sessions

**Schema**: Artifacts live in `data/bodies/artifacts.json` (new file, separate spec for
full definition). For F-013 they only need `name`, `mass_kg`, `position_x/y/z`.
Full artifact spec (F-035, not yet defined) covers rendering, docking, ownership.

**Gravitational significance**: A 1e9 kg station at 100 sim units from Earth has
`GM ≈ 1.991e-38 × 1e9 ≈ 2e-29`. Earth's GM_sim ≈ 2.4e-11. The ratio is ~1e18 —
station gravity is negligible in practice. They are `Participants` for realism and
future docking/approach accuracy, not because they meaningfully perturb orbits.

---

## 7. ShipParticle — Player Ships in Gravity

Client ships are test particles (zero gravitational contribution to named bodies).
The `ShipParticle` type bridges the session registry with the N-body physics layer.

```go
// ShipParticle is the physics representation of a client ship session.
// It holds N-body state for a single registered session.
type ShipParticle struct {
    SessionID string
    NBodyPos  [3]float64
    NBodyVel  [3]float64
    NBodyAcc  [3]float64
}
```

`ShipParticle.NBodyPos` is the authoritative server-side position. The session
registry `Position [3]float64` is updated from here each tick. Client-driven
position updates (from `UpdatePosition` RPC) override `NBodyPos` when the client
sends one (impulse / warp modes); in thruster mode the server integrates.

Lifecycle:
1. Client calls `RegisterClient` → server creates `ShipParticle`, adds to `SystemSet.TestParticles`
2. Ship moves within a planet's SOI → `SOITracker` promotes it to `LocalSet.TestParticles`
3. Ship exits SOI → removed from `LocalSet`, stays in `SystemSet`
4. Client calls `UnregisterClient` → `ShipParticle` removed from all sets

This is deferred to F-022 Phase 2 for full implementation. F-013 defines the
data structure and the GravSet integration point.

---

## 8. Files to Touch

| File | Change |
|------|--------|
| `internal/sim/engine/constants.go` | Add `G_sim`, ring exclusion helper |
| `internal/sim/engine/object.go` | Add `NBodyPos, NBodyVel, NBodyAcc [3]float64` to `AnimationState`; add `GM, HillRadius, LaplaceSOI float64` to `ObjectMetadata`; add `CategoryArtifact = 7` |
| `internal/sim/engine/state.go` | Add `SystemBarycenter Vector3` to `SimulationState`; add `SystemSet GravSet`; add `namedCache []*Object` for `collectNamed` |
| `internal/sim/engine/gravityset.go` | **New** — two-layer design: (1) **collectors** (`CollectByCategory`, `CollectInSOI`, `CollectInHillSphere`, `CollectChildren`, `CollectByName`) return `[]*Object` slices; (2) **set builders** (`SystemSet`, `LocalSet`, `SOISet`) compose collectors into a `GravSet`; (3) **integrator support** (`accumForces`, `accumTestParticleForces`, `stepGravSet`); (4) `SOITracker` for dynamic membership; (5) `ShipParticle` bridge type |
| `internal/sim/engine/physics.go` | Replace Keplerian parent loop with `stepGravSet(state.SystemSet, dt)` call; keep Keplerian child path for belt only |
| `internal/sim/loader.go` | Populate `GM`, `HillRadius`, `LaplaceSOI` in `createBodyFromConfig`; load `data/bodies/artifacts.json` if present |

No renderer changes. No proto changes in this phase.

---

## 9. Implementation Phases

### Phase 1 — Constants & type extensions

**`engine/constants.go`**
```go
const G_sim = 1.991e-38 // sim³ / (kg·s²)
```

**`engine/object.go` — `AnimationState`**:
```go
NBodyPos [3]float64
NBodyVel [3]float64
NBodyAcc [3]float64
```

**`engine/object.go` — `ObjectMetadata`**:
```go
GM         float64 // G × Mass in sim³/s²
HillRadius float64 // Hill sphere radius in sim units
LaplaceSOI float64 // Laplace SOI radius in sim units
```

**`engine/object.go` — `ObjectCategory`**:
```go
CategoryArtifact ObjectCategory = 7 // Space stations, large ships, megastructures
```

**`engine/state.go` — `SimulationState`**:
```go
SystemBarycenter Vector3
SystemSet        GravSet // default set: all named bodies + registered ships
```

### Phase 2 — GM and SOI population in loader

In `internal/sim/loader.go`, after `Mass` is assigned:

```go
obj.Meta.GM = G_sim * obj.Meta.Mass

// Hill radius: requires semi-major axis 'a' and parent mass 'M'.
// Semi-major axis in sim units derived from OrbitalPeriod + parent GM via Kepler III:
//   T² = 4π²a³ / GM_parent  →  a = (GM_parent × T² / 4π²)^(1/3)
// For top-level bodies (no parent), HillRadius = system outer boundary (e.g. 1e5 sim units).
if parent != nil && obj.Meta.OrbitalPeriod > 0 {
    T := obj.Meta.OrbitalPeriod
    a3 := parent.Meta.GM * T * T / (4 * math.Pi * math.Pi)
    a := math.Cbrt(a3)
    obj.Meta.HillRadius = a * math.Cbrt(obj.Meta.Mass / (3 * parent.Meta.Mass))
    obj.Meta.LaplaceSOI = a * math.Pow(obj.Meta.Mass/parent.Meta.Mass, 0.4)
}
```

### Phase 3 — gravityset.go

Create `internal/sim/engine/gravityset.go` containing:
- `GravSet` struct (§4.1)
- `ShipParticle` struct (§7)
- `SystemSet()`, `LocalSet()`, `SOISet()` builders (§4.3)
- `accumForces(gs GravSet)` — pairwise for Participants
- `accumTestParticleForces(gs GravSet)` — per §4.4
- `stepGravSet(gs GravSet, dt float64)` — leapfrog DKD over the whole set
- `SOITracker` — entry/exit detection (can be a no-op stub in Phase 3, filled in Phase 5)

`stepGravSet` leapfrog:
```go
func stepGravSet(gs GravSet, dt float64) {
    h := dt / 2.0
    all := append(gs.Participants, gs.TestParticles...)

    // Drift ½ — all bodies
    for _, obj := range all {
        obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
        obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
        obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
    }
    // Kick — forces at new positions
    accumForces(gs)
    accumTestParticleForces(gs)
    for _, obj := range all {
        obj.Anim.NBodyVel[0] += obj.Anim.NBodyAcc[0] * dt
        obj.Anim.NBodyVel[1] += obj.Anim.NBodyAcc[1] * dt
        obj.Anim.NBodyVel[2] += obj.Anim.NBodyAcc[2] * dt
    }
    // Drift ½ — all bodies
    for _, obj := range all {
        obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
        obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
        obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
    }
    // Copy float64 → float32 for renderer (Participants only)
    for _, obj := range gs.Participants {
        obj.Anim.Position.X = float32(obj.Anim.NBodyPos[0])
        obj.Anim.Position.Y = float32(obj.Anim.NBodyPos[1])
        obj.Anim.Position.Z = float32(obj.Anim.NBodyPos[2])
        obj.Anim.Velocity.X = float32(obj.Anim.NBodyVel[0])
        obj.Anim.Velocity.Y = float32(obj.Anim.NBodyVel[1])
        obj.Anim.Velocity.Z = float32(obj.Anim.NBodyVel[2])
    }
    // Ship particles: position is written back to session registry by F-022 caller.
}
```

### Phase 4 — N-body initialization

Called once from `NewSimulation` after mean-anomaly priming. Seeds `NBodyPos` and
`NBodyVel` from Keplerian state for all Participants in `SystemSet`.

```go
func initNBody(state *SimulationState) {
    for _, obj := range state.SystemSet.Participants {
        obj.Anim.NBodyPos = [3]float64{
            float64(obj.Anim.Position.X),
            float64(obj.Anim.Position.Y),
            float64(obj.Anim.Position.Z),
        }
        obj.Anim.NBodyVel = [3]float64{
            float64(obj.Anim.Velocity.X),
            float64(obj.Anim.Velocity.Y),
            float64(obj.Anim.Velocity.Z),
        }
        // Add parent's velocity so moons have correct inertial-frame velocity.
        if obj.Meta.ParentName != "" {
            if parent := state.ObjectMap[obj.Meta.ParentName]; parent != nil {
                obj.Anim.NBodyVel[0] += float64(parent.Anim.Velocity.X)
                obj.Anim.NBodyVel[1] += float64(parent.Anim.Velocity.Y)
                obj.Anim.NBodyVel[2] += float64(parent.Anim.Velocity.Z)
            }
        }
    }
}
```

### Phase 5 — SOI Tracker

Implement the `SOITracker` loop. Run every 60 physics ticks.

```go
type SOITracker struct {
    tickInterval int
    tickCount    int
    // localSets maps planet name → active LocalSet (nil if no ships inside)
    localSets    map[string]*GravSet
}

func (t *SOITracker) Tick(state *SimulationState) {
    t.tickCount++
    if t.tickCount < t.tickInterval { return }
    t.tickCount = 0

    for _, ship := range state.SystemSet.TestParticles {
        for _, planet := range state.SystemSet.Participants {
            dist := distance(ship.Anim.NBodyPos, planet.Anim.NBodyPos)
            local := t.localSets[planet.Meta.Name]
            insideSOI := dist < planet.Meta.LaplaceSOI

            if insideSOI && !inTestParticles(local, ship) {
                // Promote: add to LocalSet
                t.getOrCreateLocalSet(state, planet).TestParticles =
                    append(..., ship)
            } else if !insideSOI && local != nil && inTestParticles(local, ship) {
                // Demote: remove from LocalSet
                removeTestParticle(local, ship)
            }
        }
    }
}
```

Each `LocalSet` runs its own `stepGravSet` each tick in addition to `SystemSet`.
This gives ships and asteroids inside a planet's SOI accurate local gravity from
moons without paying for full system-wide test-particle forces.

### Phase 6 — Integration update loop

In `physics.go` `update()`, replace the parent Keplerian section:

```go
// Before: parallel Keplerian update for parents
// After:
stepGravSet(state.SystemSet, scaledDt)

// Run active LocalSets (ships inside SOIs)
for _, ls := range soiTracker.localSets {
    if len(ls.TestParticles) > 0 {
        stepGravSet(*ls, scaledDt)
    }
}
```

Belt Keplerian path (children) runs after N-body, unchanged.

### Phase 7 — Barycenter computation

```go
func updateBarycenter(state *SimulationState) {
    var cx, cy, cz, totalMass float64
    for _, obj := range state.SystemSet.Participants {
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

## 10. Double-Buffer / Clone Impact

`NBodyPos`, `NBodyVel`, `NBodyAcc` are fields on `AnimationState`, which is
value-copied in `Clone()` and `CloneWithPool()`. No additional clone logic needed.

`SystemBarycenter` is a `Vector3` on `SimulationState`. **Add it to the clone
struct literal** in `state.go`:
```go
SystemBarycenter: s.SystemBarycenter,
```

`SystemSet` holds pointer slices. The back-buffer owns and mutates the slices;
the front-buffer clone only reads `Position` / `Velocity` (float32). **Do not
copy `SystemSet` into the front-buffer clone** — the integrator runs exclusively
on the back buffer. This is consistent with the existing pattern for `parents`
and `children`.

---

## 11. Tests & Validation

### Automated (add to `engine/` package tests)

1. **Earth orbit period**: run N-body for 1 simulated year; verify Earth returns
   within 5° of starting true anomaly.
2. **Energy conservation**: total KE + PE for Sun+Earth two-body should vary < 0.1%
   over 10 years simulated.
3. **Moon orbit**: Earth-Moon distance stays within 35–42 sim units over 1 sim month.
4. **Barycenter consistency**: Sol-Earth barycenter inside Sol (< 27.25 sim units).
5. **SOI promotion**: ship placed at Earth+0.9×LaplaceSOI enters `LocalSet` within
   60 ticks; placed at Earth+1.1×LaplaceSOI stays in `SystemSet` only.
6. **Test particle neutrality**: adding 1,000 massless test particles does not
   change Participants' trajectories (verify by comparing two runs).

### Visual (manual, `make run`)

1. Pause sim, note Earth position. Let run for ~1 sim year at high speed. Earth
   returns to approximately the same position.
2. Jump to Earth-Moon. Moon should orbit Earth visually as before.
3. Jump to Sol. Sol remains at/near origin.
4. Register two REPL sessions, position them near Earth. Verify both appear in the
   Earth `LocalSet` and that their markers visually follow Earth's gravity.

### Regression

- `make test` passes with `-race` before and after.
- No new import cycles.
- `space-sim-direct` and `space-sim-grpc` build and run without segfault.

---

## 12. Known Risks

| Risk | Mitigation |
|------|-----------|
| Moon orbits destabilize from init velocity error | Add parent velocity in `initNBody` is the key step. Verify visually first run. |
| Tick rate too coarse for inner planets | At speed = 1e6, dt_sim ≈ 4.6 h. Leapfrog error grows as dt². Mercury perihelion will drift. Acceptable for visualization; note in lessons-learned. |
| `LocalSet.TestParticles` grows unboundedly | Cap at `MaxSOITestParticles` (default 1000) per set; oldest entrants evicted. |
| Belt asteroid SOI promotion is expensive | `SOISet` for asteroids uses a spatial partition (existing `spatial` package) to pre-filter candidates before distance check. Limit to Dataset level 0 (200 objects) by default. |
| Rings have Dataset == -1 and should stay Keplerian | Exclude `CategoryRing` from `SystemSet.Participants` in `SystemSet()` builder. |
| Multiple overlapping SOIs (e.g., asteroid in both Earth and Moon SOI) | Promote to innermost (smallest `LaplaceSOI`) body's LocalSet only. |

---

## 13. Ordering Within F-013

1. Phase 1 (types) → Phase 2 (loader GM + SOI) → `make test`
2. Phase 3 (`gravityset.go` with stub SOITracker) → Phase 4 (init) → visual check
3. Phase 5 (SOI tracker) → Phase 6 (update loop) → visual check with REPL ships
4. Phase 7 (barycenter) → full test suite → `make test -race`
5. Record results in `docs/history/lessons-learned.md`

---

## 14. Deferred to Later Features

| Topic | Feature | Notes |
|-------|---------|-------|
| Full artifact definition (rendering, docking, ownership) | F-035 (not yet defined) | F-013 adds `CategoryArtifact` + mass/position only |
| Server-authoritative ship physics (thruster/impulse/warp modes) | F-022 Phase 2 | Requires F-013 `GravSet` + `ShipParticle` to be in place |
| NPC autopilot using gravity for orbital insertion | F-022 Phase 3 | Builds on ship-as-test-particle from F-013 |
| Belt-wide N-body (Barnes-Hut, GPU) | F-014 (exploratory) | Out of scope until belt-scale performance is a stated goal |
| Cross-system multi-body (federated compute) | F-012 | Depends on F-013 being stable and single-process validated first |

---

## 15. Related Documents

- [docs/wip/todo.md](todo.md) — F-013 work items and acceptance criteria
- [docs/wip/f022-client-movement-spec.md](f022-client-movement-spec.md) — ship physics (uses F-013 GravSet)
- [internal/sim/engine/physics.go](../../internal/sim/engine/physics.go) — current Keplerian physics
- [internal/sim/engine/object.go](../../internal/sim/engine/object.go) — `ObjectMetadata`, `AnimationState`
- [internal/sim/engine/state.go](../../internal/sim/engine/state.go) — `SimulationState`, double-buffer
- [docs/history/lessons-learned-double-buffering.md](../history/lessons-learned-double-buffering.md) — concurrency invariants to preserve
