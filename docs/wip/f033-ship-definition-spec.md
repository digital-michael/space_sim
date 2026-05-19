# F-033 — Ship Definition

## Purpose

Define the externally-loaded, file-driven ship abstraction that gives every connected
player (and NPC) a named, rated spacecraft. A ship definition describes the physical
capabilities and identity of a vessel type; a ship *instance* wraps a definition with
live runtime state and is owned by exactly one client session.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md)
- [`docs/standards/coding-standards.md`](../standards/coding-standards.md)
- [`docs/wip/f021-physical-marker-spec.md`](f021-physical-marker-spec.md) — 3D model render pipeline
- [`docs/wip/f022-client-movement-spec.md`](f022-client-movement-spec.md) — locomotion physics driven by ship ratings
- [`docs/wip/f023-keyboard-config-spec.md`](f023-keyboard-config-spec.md) — keybindings that drive the ship
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — session identity and registry

## Last Updated
2026-05-18

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | All ship definitions live in `data/ships/` as JSON files — no ship data is hardcoded |
| G2 | The player's `ShipInstance` constrains all movement: acceleration is capped by engine stage ratings; turning is capped by the turning rating |
| G3 | Power budget is tracked at runtime: every active system draws from the available power pool; exceeding it degrades non-critical systems |
| G4 | Damage state is durable: survives across sessions (persisted in `ClientSession`) and affects capability |
| G5 | Identification fields (name, transponder, UUID) are visible to other players on the HUD and selectable in the object list |
| G6 | A 3D model path is embedded in the ship definition; the render pipeline (F-021) uses it as-is |
| G7 | Phase 1 ships from a bundled catalog; no user-uploaded definitions |

---

## 2. Non-Goals (this feature)

- Weapon systems and combat (separate future feature)
- Ship crafting, upgrades, or economy
- Ship-to-ship docking
- Internal ship layout / compartments
- NPC faction-specific ship behavior (NPC automation deferred to F-022 Phase 3)

---

## 3. Ship Definition File Format

### 3.1 Location

`data/ships/<id>.json` — one file per ship type. All files in this directory are loaded
at startup into an in-memory catalog. Adding a file and restarting adds the ship.

A catalog index file is not required; the server scans and loads all `*.json` files in
`data/ships/` at startup.

### 3.2 Schema

```json
{
  "$schema": "https://space-sim/schemas/ship/v1",
  "version": 1,
  "id":          "scout_mk1",
  "name":        "Scout Mk I",
  "description": "A nimble single-seat scout ship. Fast turning, modest thrust.",
  "class":       "light",

  "model": {
    "path":    "data/assets/models/scout_mk1.iqm",
    "texture": "data/assets/models/scout_mk1.png",
    "scale":   1.0
  },

  "identification": {
    "transponder_prefix": "SC",
    "registry":           "Sol Frontier Authority"
  },

  "engine_stages": [
    {
      "stage":            1,
      "label":            "Maneuvering",
      "accel_min_ms2":    0.0,
      "accel_max_ms2":    1.0e5,
      "power_draw_w":     5.0e8
    },
    {
      "stage":            2,
      "label":            "Main Drive",
      "accel_min_ms2":    1.0e5,
      "accel_max_ms2":    5.0e7,
      "power_draw_w":     4.0e9
    }
  ],

  "turning": {
    "rate_deg_per_s":  90.0,
    "power_draw_w":    1.0e8
  },

  "power": {
    "available_w":          5.0e9,
    "system_draw_baseline_w": 2.0e8,
    "overload_policy":      "degrade_non_critical"
  },

  "mass_kg": 12000.0,

  "max_speed_sim_units_per_s": 5.0,
  "superluminal_allowed":      true
}
```

### 3.3 Field Reference

#### Top-level

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `version` | int | yes | Schema version; currently `1` |
| `id` | string | yes | Unique slug; must match filename (e.g. `scout_mk1.json` → `"scout_mk1"`) |
| `name` | string | yes | Human-readable display name (max 64 chars) |
| `description` | string | no | Flavor text shown in ship selection UI |
| `class` | string | yes | `"light"`, `"medium"`, `"heavy"`, `"capital"` |
| `model` | object | yes | 3D model reference; see §3.4 |
| `identification` | object | yes | Transponder and registry data; see §3.5 |
| `engine_stages` | array | yes | One or more engine stages; see §3.6 |
| `turning` | object | yes | Rotational capability; see §3.7 |
| `power` | object | yes | Power plant and budget; see §3.8 |
| `mass_kg` | float | yes | Inert mass in kg (used by F-022 physics) |
| `max_speed_sim_units_per_s` | float | yes | Hard speed cap enforced server-side |
| `superluminal_allowed` | bool | yes | Whether warp (`move warp`) is permitted |

#### 3.4 `model` object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `path` | string | yes | Path to IQM model file relative to workspace root |
| `texture` | string | no | Base color texture; empty → tint-only render |
| `scale` | float | yes | Model-space to sim-unit scale factor |

#### 3.5 `identification` object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `transponder_prefix` | string | yes | 2-4 char prefix prepended to instance transponder ID |
| `registry` | string | no | Issuing authority shown in HUD info panel |

#### 3.6 `engine_stages` array

Each stage defines a thrust envelope. The player selects a stage via `move.engine_stage_up/down` 
bindings (new actions in F-023). Active stage determines max acceleration and power draw.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `stage` | int | yes | Stage number (1 = lowest; must be contiguous from 1) |
| `label` | string | yes | Display label (e.g. `"Maneuvering"`, `"Main Drive"`) |
| `accel_min_ms2` | float | yes | Minimum acceleration this stage can produce (m/s²) |
| `accel_max_ms2` | float | yes | Maximum acceleration this stage can produce (m/s²) |
| `power_draw_w` | float | yes | Steady-state power draw while thrusting at max (watts) |

Stages are ordered by capability. Stage 1 is always the lowest-power maneuvering mode.
The engine begins in Stage 1 at session start.

#### 3.7 `turning` object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `rate_deg_per_s` | float | yes | Maximum angular velocity for yaw, pitch, and roll |
| `power_draw_w` | float | yes | Power drawn while turning at max rate |

Actual turn rate = `rate_deg_per_s × power_fraction` where `power_fraction` is the ratio of
available power to total power budget. A fully powered ship turns at max rate; a damaged power
plant reduces turn capability proportionally.

#### 3.8 `power` object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `available_w` | float | yes | Total power plant output (watts) |
| `system_draw_baseline_w` | float | yes | Constant draw from life support, navigation, comms, etc. |
| `overload_policy` | string | yes | `"degrade_non_critical"` (default) or `"hard_cut"` |

Budget formula:
```
free_power = available_w - system_draw_baseline_w - active_engine_draw - active_turning_draw
```
If `free_power < 0`, the overload policy applies:
- `degrade_non_critical`: reduce engine thrust and turn rate proportionally until budget balances.
- `hard_cut`: engine shuts off entirely; only maneuvering (stage 1) is available.

---

## 4. Ship Instance (Runtime State)

A `ShipInstance` wraps a loaded `ShipDefinition` with live mutable state. One instance
exists per connected session. It is part of `ClientSession` in the session registry.

```go
// ShipInstance is the runtime state of one session's vessel.
type ShipInstance struct {
    // --- Identity ---
    InstanceName  string  // Player-chosen name for this copy of the ship
    TransponderID string  // Server-assigned: "<prefix>-<sessionID[:6]>"
    UUID          string  // Copied from ClientSession.SessionID

    // --- Definition reference ---
    DefinitionID string          // Key into ShipCatalog
    Definition   *ShipDefinition // Loaded definition; immutable after assignment

    // --- Engine state ---
    ActiveStage int // 1-based; index into Definition.EngineStages

    // --- Kinematics (sim units / s) ---
    Velocity      [3]float64 // Current velocity vector
    MovementVector [3]float64 // Thrust intent this frame (unit vector × throttle)
    FacingVector  [3]float32 // Unit forward vector (ship nose direction)

    // --- Power ---
    CurrentPowerW float64 // Remaining free power this tick

    // --- Damage ---
    HullIntegrity  float32 // 0.0 (destroyed) – 1.0 (undamaged)
    EngineIntegrity float32 // Scales effective engine output
    PowerIntegrity  float32 // Scales effective power output
    ShieldIntegrity float32 // Reserved for future shield system
}
```

### 4.1 Damage effects

| System | Effect of damage (below 1.0) |
|--------|------------------------------|
| Hull | Visual damage indicators on model (Phase 2); destruction at 0.0 |
| Engine | `max_accel = definition.max_accel × engine_integrity` |
| Power | `available_w = definition.available_w × power_integrity` |
| Shield | Reserved |

Damage is set by the server (F-027 collision system). Clients cannot self-heal except
via a `repair` REPL command (admin-role only; rates configurable).

### 4.2 Transponder assignment

At `RegisterClient`, the server assigns:
```
TransponderID = "<definition.identification.transponder_prefix>-<sessionID[:6].upper()>"
```
Example: `SC-A3F7C2` for a Scout Mk I session.

The transponder ID is broadcast in every `SessionDelta` and displayed in HUD overlays
and the object selection list.

---

## 5. Ship Catalog

### 5.1 Startup loading

At startup, `internal/server/ship/catalog.go` scans `data/ships/*.json`, parses each file,
validates the schema, and stores the result in an in-memory `ShipCatalog` (map keyed by `id`).

Unknown or invalid files log a warning and are skipped — they do not prevent startup.

### 5.2 Default ship

`configs/app.json` adds a `"default_ship_id"` key. If a connecting client does not specify
a ship, the server assigns `default_ship_id`. The bundled default is `"scout_mk1"`.

```json
// configs/app.json
"default_ship_id": "scout_mk1"
```

### 5.3 Ship selection

`RegisterClientRequest` (F-020 proto) gains an optional `ship_id` string field.
If populated and valid, that ship is assigned. If empty or invalid, the default applies.

**Phase 1**: selection at connect time only. In-session ship swaps are deferred.

---

## 6. Bundled Ship Catalog (Phase 1)

| ID | Name | Class | Description |
|----|------|-------|-------------|
| `scout_mk1` | Scout Mk I | light | Fast-turning, dual-stage drive. Suitable for planetary survey. |
| `freighter_t1` | Freighter T1 | medium | High-capacity, sluggish turning. Heavy main drive for bulk transport. |
| `explorer_x1` | Explorer X1 | medium | Balanced long-range survey vessel. Superluminal authorized. |

All three are committed to `data/ships/` in Phase 1.

---

## 7. Relationship to Other Features

| Feature | Relationship |
|---------|-------------|
| F-021 Physical Marker | Uses `ShipDefinition.model.path` and `.texture` for 3D render |
| F-022 Client Locomotion | Uses engine stage ratings for `thrust_force_n`; turning rating for camera/ship rotation rate |
| F-023 Keyboard Config | Adds `move.engine_stage_up/down` actions; all thrust/turn actions constrained by ratings |
| F-020 Multi-Client Session | `ClientSession` gains `ShipInstance` field; transponder shown in session list |
| F-027 Collision/Damage | Writes to `ShipInstance.HullIntegrity`, `EngineIntegrity`, `PowerIntegrity` |
| F-022 Phase 2 (gravity) | `ShipInstance.mass_kg` used as the N-body test-particle mass |

---

## 8. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `data/ships/scout_mk1.json` | **Create** | Bundled Scout Mk I definition |
| `data/ships/freighter_t1.json` | **Create** | Bundled Freighter T1 definition |
| `data/ships/explorer_x1.json` | **Create** | Bundled Explorer X1 definition |
| `internal/server/ship/definition.go` | **Create** | `ShipDefinition`, `EngineStage`, `TurningSpec`, `PowerSpec`, `ModelRef` types |
| `internal/server/ship/catalog.go` | **Create** | `ShipCatalog`; scan+load `data/ships/`; `Get(id)`, `All()`, `Default()` |
| `internal/server/ship/catalog_test.go` | **Create** | Load valid file; skip invalid; default fallback |
| `internal/server/ship/instance.go` | **Create** | `ShipInstance` struct; `NewInstance(def, sessionID, playerName)` constructor; transponder assignment |
| `internal/server/ship/instance_test.go` | **Create** | Transponder format; damage scaling; power budget calculation |
| `internal/server/session/registry.go` | **Modify** | Add `ShipInstance *ship.ShipInstance` to `ClientSession`; assign at `Register()` |
| `api/proto/spacesim/v1/session.proto` | **Modify** | Add `ship_id` to `RegisterClientRequest`; add `transponder_id` to `RegisterClientResponse` and `SessionDelta` |
| `configs/app.json` | **Modify** | Add `"default_ship_id": "scout_mk1"` |

### Phase 2 files (model render integration)

| File | Action | Notes |
|------|--------|-------|
| `data/assets/models/scout_mk1.iqm` | **Add** | Committed low-poly IQM model |
| `data/assets/models/scout_mk1.png` | **Add** | Base texture |
| `data/assets/models/freighter_t1.iqm` | **Add** | — |
| `data/assets/models/explorer_x1.iqm` | **Add** | — |
| `internal/client/go/raylib/ui/render/markers.go` | **Modify** | Use `ShipInstance.Definition.Model` instead of default sphere path |

---

## 9. Phases

### Phase 1 — Definition + Catalog + Identity

**Architectural layer**: Server ship package (`internal/server/ship/`), session registry (`internal/server/session/`), proto layer.
**Prerequisites**: F-020 Phase 1 (session registry must exist).

**Value**: Every session has a named ship instance with capability ratings and a unique transponder ID. Ship physics (thrust, turning) drive F-022 movement from Day 1 of F-022.

Work items:
- [ ] Define `ShipDefinition` and `ShipInstance` types
- [ ] Implement `ShipCatalog` with file scan and default fallback
- [ ] Assign `ShipInstance` at `RegisterClient`; populate transponder
- [ ] Expose `ship_id` in `RegisterClientRequest`; `transponder_id` in response
- [ ] Commit three bundled ship definition files
- [ ] Unit tests: catalog load, schema validation, transponder format, damage scaling, power budget

Acceptance criteria:
- `list sessions` REPL output shows transponder ID for each session ✓
- Unknown `ship_id` at register falls back to `default_ship_id` ✓
- `ShipInstance` fields accessible to F-022 movement handler ✓

### Phase 2 — 3D Model Integration

**Architectural layer**: Render layer (F-021 model pipeline).
**Prerequisites**: Phase 1 complete; F-021 Phase 2 (IQM model rendering) complete.

**Value**: Each ship type renders its own 3D model instead of a generic sphere.

Work items:
- [ ] Commit IQM + texture assets for all three bundled ships
- [ ] `markers.go` reads `ShipInstance.Definition.Model.Path` for `LoadModel`
- [ ] Scale factor applied: `model.scale` in sim units
- [ ] Color tint from `ClientSession.Color` applied over base texture

### Phase 3 — Engine Stage Selection + Power HUD

**Architectural layer**: Input layer, HUD layer.
**Prerequisites**: Phase 1 complete; F-023 Phase 1 complete.

**Value**: Player can cycle engine stages; power budget is visible in HUD.

Work items:
- [ ] `move.engine_stage_up` / `move.engine_stage_down` actions added to F-023 vocabulary
- [ ] Stage cycling input: increments/decrements `ShipInstance.ActiveStage`
- [ ] HUD panel: active stage label, power budget bar, damage integrity bars
- [ ] Power overload triggers overload policy in server movement handler

---

## 10. Open Questions

| # | Question | Decision needed by |
|---|----------|--------------------|
| Q1 | Should `ShipDefinition.id` enforce kebab-case validation or allow any alphanumeric? | Phase 1 |
| Q2 | Can the same ship definition be assigned to multiple concurrent sessions (shared type, distinct instances)? | **Yes** — each session gets its own `ShipInstance`; definitions are immutable shared values | Resolved |
| Q3 | Should `InstanceName` be player-chosen at register time or derived from session label? | Phase 1 |
| Q4 | Should turning constraints apply to the camera (rotating the view) or only to the ship's facing vector? | Phase 3 — determine at F-023 / F-022 integration |
| Q5 | What happens to `ShipInstance` state when a player disconnects and reconnects? | Phase 1 — decision: reset to undamaged on reconnect (no persistence in Phase 1); durable in Phase 2+ via F-005 |
