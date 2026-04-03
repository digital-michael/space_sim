# Technical Reference

## Purpose
Describe the simulation and rendering architecture for Space Sim at a level readable by an informed engineer or advanced user. Covers the current Keplerian simulation, all planned physics models, comparative computational costs, the performance options available in the UI, and how future roadmap features interact with each simulation approach.

## Last Updated
2026-03-30

## Table of Contents
1. Performance Options
   1.1 Frustum Culling
   1.2 Level of Detail (LOD)
   1.3 Point Rendering
   1.4 Spatial Partitioning
   1.5 Instanced Rendering
   1.6 Importance Threshold
   1.7 In-Place Swap
2. Simulation
   2.1 Current Simulator — Keplerian Orbital Mechanics
   2.2 Planned Simulator — Barycenter (Center-of-Mass) Mechanics
   2.3 Additional Physics Models
   2.4 Computational Cost and Model Selection
   2.5 Per-Object Physics Model Routing
   2.6 Runtime Hot-Swapping of Simulators
   2.7 Future Feature Considerations

---

## 1. Performance Options

The performance panel exposes a set of independent rendering and memory toggles. Each toggle is orthogonal — it can be enabled or disabled in any combination. Defaults are chosen to give the best out-of-the-box frame rate without compromising visual correctness.

### 1.1 Frustum Culling

**Default: on**

Frustum culling removes objects that lie outside the camera's view pyramid before any drawing happens. Each frame, the renderer computes a simplified view frustum from the camera position and field of view, then only passes objects that fall inside it (plus a small angular margin) to the draw call.

Objects very close to the camera are always included regardless of angle, preventing a near-object pop-in edge case. In practice, when the camera is positioned far from the system origin and much of the scene is visible, the speedup is modest — around 3% in measured tests. The benefit grows substantially when the camera is positioned inside or near a dense object field such as the asteroid belt, where a large fraction of objects genuinely fall behind the camera.

### 1.2 Level of Detail (LOD)

**Default: off (known issue with large datasets)**

LOD reduces the geometric complexity of sphere objects based on their distance from the camera. Closer objects are rendered with higher polygon counts (up to 32 rings × 32 slices); objects far away drop as low as 6 rings × 6 slices. Small objects (physical radius less than 1.0 simulation unit) get an additional halving of their polygon count at all distances.

On the small dataset (200 asteroids, 514 total objects), LOD alone produces a **5× frame-rate improvement**, cutting average draw time from 37.6 ms to 7.6 ms. It is the single highest-impact rendering option in the suite.

There is a known hang-on-render bug when LOD is enabled alone on datasets larger than roughly 1,000 objects. The bug presents as the frame loop stalling during the measurement phase; it is suspected to be an infinite loop or deadlock condition in the per-object distance calculation path at elevated object counts. Until this is resolved, LOD is safe to use on small datasets or when combined with other options (the "all combined" profile, which includes LOD, ran without issue at medium object counts).

### 1.3 Point Rendering

**Default: off**

Point rendering replaces full 3D sphere meshes with tiny flat-shaded spheres (single-pixel-scale dots) for objects that are either far from the camera or physically small. The distance threshold varies by category: asteroids switch to points beyond 100 simulation units, moons beyond 300, planets beyond 500, and everything else beyond 200. Objects with a physical radius below 0.5 units always render as points regardless of distance.

The visual tradeoff is that distant or small objects lose their spherical silhouette, but at those distances the silhouette provides no useful information. Measured improvement on small datasets is approximately **43% frame rate increase**, making point rendering the second-most-effective single option after LOD.

### 1.4 Spatial Partitioning

**Default: on (in the UI; off in baseline performance tests)**

Spatial partitioning divides the scene into a uniform grid with a cell size of 50 simulation units. When frustum culling is also enabled, the culling pass queries only the grid cells that intersect the view frustum rather than iterating the full object list. When frustum culling is disabled, spatial partitioning provides no benefit at all.

In measured tests, spatial partitioning alone produced a **−3% result** (slightly slower than baseline). The overhead of grid construction and query outweighed the savings at the tested camera position, where most objects were already visible. The optimization is most effective when the camera is close to a region of the scene and a large portion of the grid is outside the frustum. It is paired with frustum culling by design; enabling it without frustum culling is a no-op in terms of cull savings.

### 1.5 Instanced Rendering

**Default: on**

Instanced rendering groups objects that share the same rendering properties — sphere resolution, radius bucket, point flag, and wireframe detail level — into batches and draws each batch together. This reduces the number of distinct draw calls the renderer issues per frame by grouping similar objects.

In measured tests, instanced rendering alone produced a **−3% result**, similar to spatial partitioning. The batching overhead (hash-keyed map construction per frame) offset the draw-call savings at small and medium dataset sizes. The approach is designed to scale better at very high object counts, where the draw-call reduction becomes large enough to dominate the batching overhead. It is expected to show material benefit at the large (2,400) and extra-large (24,000) asteroid datasets.

### 1.6 Importance Threshold

**Default: 0 (all objects rendered)**

Each object carries an integer importance value from 0 to 100 set in the JSON data. The importance threshold slider filters out objects whose importance is below the threshold at draw time. Setting the threshold to 50, for example, suppresses low-priority asteroids and background debris while keeping planets, moons, and named bodies visible.

This is a rendering-only filter; objects below the threshold still exist in the simulation and are still physics-updated each tick. The threshold is applied at draw time, not at simulation time, so it has no impact on CPU physics cost.

### 1.7 In-Place Swap

**Default: on**

The simulation uses a double-buffer design: a back buffer where physics writes new object positions each tick, and a front buffer that the renderer reads. Normally, swapping involves either cloning the back-buffer state into the front buffer (allocating new objects each swap) or swapping pointers. In-place swap copies field values from back to front buffer in place, overwriting the existing front-buffer object slots rather than allocating new ones. This eliminates per-frame allocation noise and dramatically reduces garbage-collection pressure.

The option can be disabled for benchmarking or debugging purposes to measure the cost difference. Under normal operation there is no reason to turn it off.

---

## 2. Simulation

### 2.1 Current Simulator — Keplerian Orbital Mechanics

The current simulator computes all object positions from closed-form Keplerian orbital elements. Each object carries a fixed set of orbital parameters in its metadata: semi-major axis, eccentricity, inclination, longitude of ascending node, argument of periapsis, mean anomaly at epoch, and orbital period. These values do not change at runtime.

Each physics tick, the simulator advances a body's mean anomaly by a small angle proportional to elapsed scaled time. It then solves Kepler's equation (`M = E − e sin E`) for eccentric anomaly using Newton-Raphson iteration (converges in 10 steps or fewer for all physically plausible eccentricities), converts to true anomaly, computes the orbital radius, applies the three Keplerian rotation matrices to get a 3D position relative to the orbital center, and writes the result back to the object's animation state.

The update runs in a parallel worker pool. Parent bodies (stars, planets, top-level bodies) are processed first across multiple goroutines. A parent-position map is then built so each child body (moons, rings) can look up its parent's newly computed position and offset its own orbit accordingly. Children are then processed in a second parallel pass. The entire pass concludes with a buffer swap.

This model is highly efficient per object because it requires no numerical integration, no inter-object force computation, and no iterative convergence beyond the single Kepler solve. The cost scales linearly with object count. The approach is accurate enough for display and navigation purposes at the timescales and eccentricities present in the solar system data.

One fundamental limitation of pure Keplerian mechanics is that orbits are modeled as fixed ellipses around a stationary parent. In reality, two bodies of comparable mass orbit their shared center of mass (barycenter) rather than one body orbiting the other. For most solar system pairs — planet around Sun, moon around planet — the barycenter lies well inside the larger body, and Keplerian mechanics is visually indistinguishable from barycentric. For a few cases (Pluto–Charon, the Sun–Jupiter system, binary stars) the barycenter displacement is physically significant.

### 2.2 Planned Simulator — Barycenter (Center-of-Mass) Mechanics

The Barycenter simulator adds a second physics mode for designated body pairs. Rather than treating one body as a fixed anchor and the other as a satellite, both bodies orbit a computed center of mass point that lies between them on the line joining their centers. The COM position is:

$$\text{COM} = \frac{m_1 \cdot \vec{r_1} + m_2 \cdot \vec{r_2}}{m_1 + m_2}$$

Each body then follows a Keplerian ellipse around that COM point, with the ellipse scaled by the reduced-mass fraction for that body. The orbital period and eccentricity are shared properties of the pair; only the semi-major axis differs between the two (proportional to the inverse mass ratio).

The barycenter mode does **not** replace the existing Keplerian engine. It is added as a second code path inside the same physics tier. The object data model (`ObjectMetadata`, `AnimationState`, `SimulationState`, `DoubleBuffer`) requires no changes; the position result written to `AnimationState` is the same field regardless of which mode computed it. Two new fields are added to `ObjectMetadata`:

- `PhysicsModel` — an enum with values `PhysicsKeplerian` (default) and `PhysicsBarycenter`.
- `BarycentricPeer` — the name of the other body in a barycentric pair.

Objects without these fields set default to `PhysicsKeplerian`, preserving full backward compatibility with all existing JSON data and all loaded systems.

### 2.3 Additional Physics Models

Four physics models complement Keplerian and Barycenter in the planned architecture. Each is dispatched through the same per-object `PhysicsModel` routing seam and adds zero overhead to objects not tagged for that model. They are listed here in order of implementation priority given the current roadmap.

**Patched Conic / Sphere of Influence (SOI)**

Each body orbits using Keplerian mechanics, but the parent it orbits can change. When a body crosses the sphere-of-influence boundary of a dominant attractor its Keplerian elements are recomputed relative to the new parent. The math inside each SOI is identical to the existing Keplerian path — no new integrator is required.

The `ParentName` field and parent-lookup map already implement the structural concept. The additional machinery is a per-tick SOI boundary check for flagged objects and a transition handler that emits a discrete parent-change event. SOI transitions are naturally compatible with event-log replay because they are discrete, event-sized state mutations.

Best use: spacecraft or user-controlled agents introduced by gRPC clients. This is the most direct extension of the current architecture and the most likely model to follow Barycenter.

**Statistical / Procedural**

Objects in this mode skip the Kepler solve entirely. Their position each tick is synthesized from a deterministic function of simulation time and a per-object seed, distributing them within a defined annular region. Belt generation already uses seeded procedural placement for initial positions; a `PhysicsStatistical` dispatch extends that to skip the per-tick Keplerian update for low-importance belt objects.

This has no physical accuracy cost at the distances and LOD levels where background belt members are rendered — individual orbit accuracy is irrelevant when an object is displayed as a single pixel. The CPU savings are directly proportional to the number of objects reassigned.

Best use: background asteroid belt and Kuiper belt bodies at low LOD. The highest-impact option for reducing physics CPU cost at the large (2,400) and extra-large (24,000) asteroid dataset sizes.

**Symplectic / Verlet Numerical Integration**

Each body carries a velocity vector and integrates forward using a leapfrog (Störmer-Verlet) integrator. Symplectic integrators conserve energy over long simulated timescales far better than Euler, making them suitable for scenarios where orbital drift over centuries matters. Additional force terms — solar radiation pressure, oblateness, atmospheric drag, thrust — attach as additive acceleration corrections without changing the integrator structure.

For circular or low-eccentricity orbits over short periods the result is visually indistinguishable from Keplerian at higher cost per tick. This model should be restricted to flagged high-interest objects with meaningfully evolving orbits: long-period comets, highly eccentric bodies, and spacecraft under continuous thrust.

Best use: selective application to objects whose orbits evolve over the simulated timespan. Not viable as a default for belt-scale populations.

**N-Body Gravitational Integration**

Every tagged body exerts gravitational attraction on every other tagged body. Positions advance by numerical integration rather than closed-form math, naturally producing orbital precession, resonances, and mutual perturbations that Keplerian and Barycenter cannot replicate.

Two integration approaches exist. Naive N-body computes all force pairs directly at O(n²) cost — fast and simple for small groups (n ≤ ~10). Barnes-Hut groups distant bodies into aggregate mass nodes using an octree, reducing cost to O(n log n), but the tree construction and traversal overhead means it is only cheaper than naive N-body above roughly n = 80 bodies in this codebase's cost profile.

Best use: named stars, planets, and compact gravitationally significant groups of up to ~20 bodies. Not viable for belt-scale populations regardless of integration strategy.

---

### 2.4 Computational Cost and Model Selection

The table below gives the recommended model for each scenario, followed by the cost analysis that backs it.

#### Selection Guide

| Scenario | Recommended model | Reason |
|---|---|---|
| Planet or moon around a massive parent | Keplerian | Barycenter displacement is sub-pixel; Keplerian cost is optimal |
| Binary star, Pluto–Charon, Sun–Jupiter | Barycenter | Physically correct; cost identical to Keplerian |
| Spacecraft, user agents, trajectory planning | Patched Conic / SOI | Discrete parent transitions; replay-safe; extends current architecture directly |
| Background belt members (large/xlarge datasets) | Statistical / Procedural | Zero Kepler solve overhead; indistinguishable at render distance |
| Long-period comets, thrusting spacecraft | Symplectic / Verlet | Energy-conserving integration; supports additive force terms |
| Compact group of 4–20 interacting bodies | Naive N-body | Cheaper than Keplerian at small n; more accurate |
| Self-gravitating cluster of 80+ bodies | Barnes-Hut | Only regime where octree overhead is justified |
| Asteroid belts (hundreds to thousands) | Keplerian or Statistical | N-body would be quadratically expensive; orbit accuracy not required |

#### Unit Cost Model

To compare models on equal footing, each is reduced to weighted floating-point operations per tick, where a transcendental function (`sin`, `cos`, `sqrt`, `atan`) counts as approximately 10× a regular multiply.

**Keplerian per body — K ≈ 170 weighted ops**
- Advance mean anomaly: 2 ops
- Newton-Raphson Kepler solve (~6 iterations of `sin` + `cos` + divide): ~120 weighted ops
- True anomaly (`atan` + `sqrt`): ~25 weighted ops
- 3D orbit rotation (trig + matrix multiply): ~20 weighted ops

**N-body force pair — F ≈ 35 weighted ops per directed pair**
- Distance vector: 3 subtracts
- Distance magnitude (`sqrt`): ~10 weighted ops
- Force magnitude and direction: 6 multiplies + 3 divides
- Apply acceleration to both bodies: 12 ops
- Leapfrog integration per body (separate, not per pair): ~12 ops

**Barycenter per body — ≈ 1.01 K**
The COM pre-pass adds ~4 arithmetic ops per pair (~2 per body). Indistinguishable from Keplerian in practice.

#### Cost Formulas

$$\text{Keplerian}(n) = n \times 170$$

$$\text{Barycenter}(n) \approx n \times 171$$

$$\text{Naive N-body}(n) = \frac{n(n-1)}{2} \times 35 + n \times 12$$

$$\text{Barnes-Hut}(n) \approx \text{Naive N-body}(n) \times (1 + 0.5 \log_2 n)$$

#### Cost at n = 2, 4, and 6 Interacting Bodies

| n | Keplerian | Barycenter | Naive N-body | Barnes-Hut | N-body vs Keplerian |
|---|---|---|---|---|---|
| 2 | 340 ops | 342 ops | 59 ops | ~77 ops | **0.17×** |
| 4 | 680 ops | 687 ops | 258 ops | ~361 ops | **0.38×** |
| 6 | 1,020 ops | 1,031 ops | 597 ops | ~896 ops | **0.59×** |

N-body naive is cheaper than Keplerian at these small group sizes. This is not a surprise: Keplerian pays its transcendental-heavy Kepler solve per body regardless of how many other bodies exist, while N-body's O(n²) cost starts from a low base. Below roughly n = 10, naive N-body costs less than Keplerian and is also more physically accurate.

#### Crossover Points

Setting naive N-body cost equal to Keplerian and solving:

$$17.5(n-1) + 12 = 170 \implies n \approx 10$$

- **n ≈ 10**: naive N-body crosses Keplerian cost.
- **n = 20**: naive N-body costs approximately 2× Keplerian.
- **n = 50**: naive N-body costs approximately 8× Keplerian.
- **n ≈ 80**: Barnes-Hut first becomes cheaper than naive N-body. Below this the octree overhead dominates.

Consequence: for the small gravitationally significant groups in any realistic solar system dataset (binary pairs, compact clusters of named bodies), naive N-body is the cheaper and more accurate choice. Barnes-Hut is only warranted if a future feature introduces self-gravitating clusters of 80 or more bodies — nothing in the current or planned dataset comes close to that threshold.

---

### 2.5 Per-Object Physics Model Routing

The update loop dispatches to the correct model on a per-object basis. During the parent pass, each object is checked for its `PhysicsModel` value. Keplerian objects go through the existing path unchanged. Barycentric pairs require a preliminary step: the shared COM must be resolved before either body's individual position can be computed. This is handled by a first-pass scan over barycentric-tagged parents that groups them into pairs, computes each pair's COM from their current masses, and stores the result on each body's animation state as a transient orbit center. The individual position updates for barycentric bodies then proceed identically to Keplerian, but orbit around the stored COM rather than the parent's raw position.

This sequencing constraint means barycentric pairs must finish their COM pre-pass before the parallel per-body update begins. The COM pre-pass itself is fast (only a handful of pairs exist in any realistic dataset) and does not affect the heavily-parallelized main update of non-barycentric objects. Asteroid belts, dwarf planets, moons, and all other bodies tagged `PhysicsKeplerian` run concurrently across the full worker pool exactly as they do today.

The routing is fully symmetric: the COM pre-pass is a no-op when no objects carry `PhysicsBarycenter`, so there is zero overhead on datasets that contain no barycentric pairs.

### 2.6 Runtime Hot-Swapping of Simulators

The physics model assignment lives in static metadata loaded from JSON — it is not a runtime selection. However, the architecture supports changing object assignments between sessions without code changes, because the JSON data file is the source of truth and the loader maps it onto `ObjectMetadata` at startup.

A narrower form of hot-swapping — toggling the physics model for a specific pair at runtime — is feasible via the planned event queue (Phase 3). A `SetPhysicsModel` event type, applied through the same channel-based pattern already used for dataset changes and speed changes, would let the simulation apply the change cleanly between ticks without any mid-frame state mutation. The double-buffer contract enforces that changes only take effect on the next back-buffer write, which is the correct behavior.

Object data remains fully isolated from which physics path computed it. The renderer reads positions from `AnimationState.Position` regardless of their origin; it has no knowledge of whether those positions came from Keplerian math, barycentric math, or any future model.

### 2.7 Future Feature Considerations

The following roadmap phases have direct or indirect interactions with the simulator design.

**Phase 3 — Event Queue System.** The event envelope design should reserve a `SetPhysicsModel` event type when the initial event type set is defined, even if the handler is stubbed at first. Adding it after the schema is finalized requires a version increment and increases retrofit friction. The cost of reserving the type early is negligible.

**Phase 4 — Event Loop and Worker Pool.** The canonical parallel frame loop introduced in Phase 4 must accommodate the COM pre-pass required by barycentric pairs. The frame loop should expose a named first-pass hook for objects with inter-body dependencies, even if no barycentric bodies exist yet. If it is designed assuming full independence of all parents and that assumption is later violated, the refactor touches the frame loop core. Building the hook in from the start keeps the change local to the body-type classification code.

**Phase 5 — Persistence.** Snapshots serialize `ObjectMetadata`, which will include `PhysicsModel` and `BarycentricPeer` for barycentric bodies. Adding these fields to `ObjectMetadata` before the protobuf snapshot schema is finalized avoids a version bump later. Event-log replay must reproduce the physics model assignment faithfully; if model assignments can change at runtime via the event queue, the event log must record those changes. Replaying a log that omits model-change events would produce divergent state.

**Phase 6 — gRPC Integration.** RPC handlers that expose or modify physics model assignments depend on the event type established in Phase 3. No additional structural concern beyond the Phase 3 note above.

**Phase 7 — Additional Pool Types.** No interaction. The object pool operates on `Object` slots and is agnostic to which physics path computed the contents. Barycentric objects are standard `Object` instances and are managed identically by any pool strategy.

**Performance Options and Barycenter.** All rendering-layer performance options — LOD, frustum culling, point rendering, spatial partitioning, instanced rendering, importance threshold, and in-place swap — operate on `AnimationState.Position` and object metadata fields that are populated identically regardless of which simulator produced them. All options are fully compatible with both simulation models. The one option with a simulation-side coupling is in-place swap: it applies to the buffer swap that follows the full physics pass, not to the physics pass itself, so it is unaffected by the addition of a COM pre-pass before the parallel update.
