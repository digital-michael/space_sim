# F-034 — System Data Directory Structure

## Purpose
Replace each monolithic `data/systems/<name>.json` file with a versioned, per-system directory containing typed sub-files. Extend the object type vocabulary to include rogues (high-eccentricity comets and interstellar objects) and artifacts (human-made or alien-made durable objects). This work integrates and supersedes **F-008 — Artifact Object Type**.

## Status
📋 Not started

## Last Updated
2026-05-24

## Priority
**High** — Foundational data management. Must precede F-013 N-body work so that a dedicated N-body test system can be defined and committed. Deferring until after F-013 would require a data migration during or after physics work.

## Depends On
- Nothing blocking. Self-contained schema and loader change.

## Unlocks
- F-013: N-body test system can be defined cleanly in the new structure.
- F-008 functionality is delivered as part of this feature (closed).
- F-035 Game Definition references system directories by name; the layout must be stable first.

## F-008 Integration
F-008 (Artifact Object Type, currently Phase G) is absorbed into this feature. All artifact-related design goals from F-008 are treated as requirements here. Any F-008 acceptance criteria that cannot be fully met in Phase 1 of this feature are explicitly listed as Phase 2 items below.

---

## 1. Motivation

The current single-file layout (`solar_system.json`, `alpha_centauri_system.json`, etc.) has the following problems:

1. All object types are mixed in one array — no per-type validation pass is possible without a full parse.
2. Adding a new object type (rogues, artifacts) means modifying every existing system file or relying on `"type"` field dispatch in a flat array.
3. Version management is a single field on the top-level object with no per-type granularity.
4. Large systems (SOL, with moons, belts, and dwarf planets) produce files that are hard to edit by hand or diff.
5. Build-your-own test systems for F-013 N-body require artificial trimming of the monolithic file.

---

## 2. New Directory Layout

```
data/
  systems/
    solar_system/
      system.json         ← manifest: name, version, scale settings, active flags
      stars.json
      planets.json
      dwarf_planets.json
      moons.json
      belts.json
      rogues.json         ← NEW: comets, interstellar objects, high-eccentricity bodies
      artifacts.json      ← NEW: human-made / alien-made durable objects (absorbs F-008)
    alpha_centauri_system/
      system.json
      stars.json
      planets.json
      ...
    nbody_test/           ← NEW: minimal system for F-013 N-body validation
      system.json
      stars.json
      planets.json

  bodies/                 ← Template libraries (unchanged path; versioned separately)
    planets.json          ← version field added; backward-compatible
    moons.json
    stars.json
    dwarf_planets.json
```

### Rules
- Each sub-file is **optional**. A loader that finds no `rogues.json` treats that as an empty rogue list.
- `system.json` is **required**. Absence is a fatal load error.
- All files in a system directory share the same `system_version` declared in `system.json`.
- Template libraries under `data/bodies/` are **independently versioned** with their own `version` field and are referenced from body definitions via `template_source`.

---

## 3. system.json Schema

```json
{
  "name": "Solar System",
  "system_version": "2.0",
  "schema_version": "2.0",
  "scale_factor": 50,
  "simulation": {
    "default_time_scale": 1,
    "nbody_mode": "keplerian"
  },
  "files": {
    "stars":        "stars.json",
    "planets":      "planets.json",
    "dwarf_planets":"dwarf_planets.json",
    "moons":        "moons.json",
    "belts":        "belts.json",
    "rogues":       "rogues.json",
    "artifacts":    "artifacts.json"
  }
}
```

| Field | Notes |
|-------|-------|
| `system_version` | Content version. Bump when body data changes. |
| `schema_version` | Loader schema version. Bump when field shapes change. |
| `nbody_mode` | `"keplerian"` (default) or `"nbody"`. Enables full N-body integration when F-013 is active. |
| `files` | Map of type → filename. Omit keys for sub-files that do not exist. |

---

## 4. Per-Type File Schemas

### 4.1 stars.json / planets.json / dwarf_planets.json / moons.json

Same body-definition shape as today (orbit, physical, rendering, importance). No breaking changes.

```json
{
  "schema_version": "2.0",
  "bodies": [ ... ]
}
```

### 4.2 belts.json

Same belt/ring/feature-config shape as today.

```json
{
  "schema_version": "2.0",
  "belts": [ ... ]
}
```

### 4.3 rogues.json — New Type

Comets, interstellar objects, and any natural body that does not orbit in a near-circular, ecliptic-plane path.

```json
{
  "schema_version": "2.0",
  "rogues": [
    {
      "type": "comet",
      "name": "Halley",
      "subtype": "periodic",
      "orbit": {
        "semi_major_axis": 17.8,
        "eccentricity": 0.967,
        "inclination": 2.834,
        "longitude_ascending_node": 1.005,
        "argument_periapsis": 1.949,
        "orbital_period": 27375.84,
        "initial_mean_anomaly": "random"
      },
      "physical": {
        "radius": 0.05,
        "mass": 2.2e+14,
        "rotation_period": 52.8,
        "axial_tilt": 0.0,
        "albedo": 0.04
      },
      "rendering": {
        "material": "diffuse",
        "fallback_color": [200, 220, 255, 255],
        "coma_color": [200, 240, 255, 180],
        "tail_color": [180, 220, 255, 120],
        "coma_scale": 5.0,
        "tail_length_au": 1.0
      },
      "importance": 40
    }
  ]
}
```

#### Rogue Subtypes

| Subtype | Description |
|---------|-------------|
| `periodic` | Short or long-period comet on a bound, high-eccentricity orbit |
| `hyperbolic` | Interstellar object on a hyperbolic (e > 1) trajectory through the system |
| `dormant` | Extinct comet nucleus; no active coma; treated as rocky rogue |
| `interstellar_asteroid` | Non-icy interstellar object (Oumuamua-class) |

#### Phase 1 vs Phase 2 Notes
- **Phase 1**: Rogues load and simulate as Keplerian orbits. Rendering uses fallback color only (no coma/tail shader).
- **Phase 2**: Coma and ion-tail shaders active; tail direction dynamically computed from sun-vector. Close-approach events fire from the event queue (links to F-009).

#### N-Body Eligibility
Rogues with high eccentricity (periodic and hyperbolic subtypes) are candidates for full N-body integration under F-013 when `nbody_mode = "nbody"`. The Keplerian orbit in `rogues.json` serves as the initial-state seed, identical to the approach F-013 uses for named planets. The N-body integrator is expected to naturally reproduce perihelion passage and gravitational deflection for these bodies without special-casing. This relationship should be validated as part of F-013 acceptance criteria using a rogue comet in the `nbody_test` system.

### 4.4 artifacts.json — New Type (absorbs F-008)

Human-made or alien-made objects with persistent presence in the system. Examples: probes, stations, megastructures, derelicts, historical markers.

```json
{
  "schema_version": "2.0",
  "artifacts": [
    {
      "type": "artifact",
      "name": "Voyager 1",
      "subtype": "probe",
      "faction": null,
      "orbit": {
        "semi_major_axis": 0,
        "eccentricity": 0,
        "inclination": 0,
        "orbital_period": 0,
        "position_override": [147.7, 12.3, 2.1],
        "velocity_override": [17.0, 0.1, -0.3]
      },
      "physical": {
        "radius": 0.002,
        "mass": 825.5
      },
      "rendering": {
        "material": "diffuse",
        "model_file": null,
        "fallback_color": [200, 200, 220, 255],
        "scale_override": 1.0
      },
      "importance": 30,
      "metadata": {
        "launch_date": "1977-09-05",
        "description": "First human-made object to enter interstellar space.",
        "tags": ["probe", "historical", "nasa"]
      }
    }
  ]
}
```

#### Artifact Subtypes

| Subtype | Description |
|---------|-------------|
| `probe` | Unmanned scientific or exploratory vehicle |
| `station` | Fixed or orbital installation |
| `derelict` | Non-functional vessel or installation; historical |
| `megastructure` | Large artificial structure (ring world, dyson element, etc.) |
| `beacon` | Navigation or communication marker |
| `marker` | Fictional or lore-only placement |

#### Phase 1 (this feature)
- Loader recognizes `artifact` type; creates `ArtifactMeta` and `ArtifactState` in engine.
- Artifacts rendered as fallback-color spheres (placeholder model rendering).
- `position_override` and `velocity_override` allow placing artifacts at known coordinates rather than computing from Keplerian elements.
- `faction` field parsed but not enforced (faction system belongs to F-035).

#### Phase 2 (post F-035 / game definition)
- `faction` field linked to F-035 faction registry.
- 3D model rendering via IQM pipeline (same as F-033 Phase 2 for ships).
- Interaction events (proximity triggers, comms from artifact) via F-009 / F-025.

#### F-008 Gaps Carried Forward
The following F-008 acceptance criteria are deferred to Phase 2:
- [ ] 3D model rendering (requires IQM pipeline from F-033 Phase 2)
- [ ] Faction ownership enforcement
- [ ] Artifact-to-artifact and artifact-to-player proximity events (requires F-009)

---

## 5. Versioning Strategy

### system_version
- Tracks the **content** of a system (which bodies are defined, their orbital parameters, etc.).
- Bump on any change to body data.
- The loader logs the system_version at startup (visible in debug output).

### schema_version
- Tracks the **shape** of the JSON schema (field names, types, nesting).
- The loader validates `schema_version` against supported versions and rejects unsupported schemas with a clear error.
- Supported versions must be listed in a `LoaderConfig` constant so old systems remain loadable during transition.

### Template Library Versioning
- `data/bodies/*.json` files carry their own `version` field.
- Body definitions that reference a template may declare a minimum template version: `"template_min_version": "1.2"`.
- Loader warns (does not fail) if a template version is older than `template_min_version`.

### Migration from v1 (monolithic) to v2 (directory)
- A migration script (`scripts/migrate_system_v1_to_v2.py`) will split existing `.json` files into the new directory layout.
- Legacy single-file loader path remains active under a `--legacy-system-file` flag until all systems are migrated.
- Migration script is provided as part of Phase 1 delivery.

---

## 6. Loader Changes

### Current
```
internal/sim.LoadSystemFromFile(path string) (*SystemConfig, error)
```

### Target
```
internal/sim.LoadSystemFromDir(dir string) (*SystemConfig, error)
internal/sim.LoadSystemFromFile(path string) (*SystemConfig, error)  // legacy; kept for migration period
```

`LoadSystemFromDir`:
1. Read `system.json` → extract manifest and file map.
2. For each declared sub-file, read and unmarshal into the typed sub-struct.
3. Validate schema_version per sub-file.
4. Resolve parent references across files (moons reference planets by name; validated post-load).
5. Return merged `SystemConfig` — same shape as today for backward compatibility with all downstream consumers.

---

## 7. N-Body Test System

A minimal synthetic system `data/systems/nbody_test/` will be created with:
- 1 star (Sol-mass)
- 2 planets in resonant orbits at known mass ratios
- Optionally: 1 binary pair or barycenter test case

This system's orbital parameters will be chosen so that the Keplerian initial state and N-body evolution have analytically checkable properties (e.g., circular orbit → constant radius; two-body problem → known precession rate).

The `nbody_test` system will be the primary validation artifact for F-013.

---

## 8. Acceptance Criteria

### Phase 1
- [ ] `data/systems/solar_system/` directory created with `system.json`, `stars.json`, `planets.json`, `dwarf_planets.json`, `moons.json`, `belts.json`; existing behavior preserved exactly.
- [ ] `rogues.json` present for solar_system with at minimum Halley's Comet as a Keplerian rogue body.
- [ ] `artifacts.json` present for solar_system with Voyager 1 and Pioneer 10 as position-overridden probes.
- [ ] `LoadSystemFromDir` loads, validates schema_version, resolves parents, and returns correct `SystemConfig`.
- [ ] Legacy `LoadSystemFromFile` still works; `--legacy-system-file` flag supported.
- [ ] Migration script converts all existing system JSON files; output validated by loader.
- [ ] `data/systems/nbody_test/` created with 3-body test configuration.
- [ ] All existing unit tests pass; new tests for per-type loading and parent resolution added.
- [ ] Rogue and artifact bodies appear in the simulation (rendered as fallback-color spheres).
- [ ] `schema_version` mismatch produces a clear, actionable error message.

### Phase 2 (post F-035/F-033 Phase 2)
- [ ] Rogue coma and ion-tail shader active.
- [ ] Artifact 3D model rendering via IQM pipeline.
- [ ] `faction` field on artifacts linked to F-035 faction registry.
- [ ] Artifact proximity events fire via F-009.

---

## 9. Related Documents
- [docs/schema/solar-system-json-schema.md](../schema/solar-system-json-schema.md) — existing schema; update on Phase 1 complete
- [docs/wip/f013-nbody-plan.md](f013-nbody-plan.md) — N-body implementation plan (references test system)
- [docs/wip/f035-game-definition-spec.md](f035-game-definition-spec.md) — Game Definition (themes, factions)
- [docs/wip/roadmap.md](roadmap.md) — ordering and phase assignments
