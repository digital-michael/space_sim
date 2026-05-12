# F-021 — Client Physical Marker

## Purpose

Define the visual representation of each connected client session in the simulated world.
A physical marker makes multi-client presence visible: you can see where other players are,
identify them by color and label, and eventually see their ship model.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md)
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — session identity and color assignment
- [`docs/wip/f022-client-movement-spec.md`](f022-client-movement-spec.md) — marker position driven by movement

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Every connected client session has a distinct, recognizable visual presence in the world |
| G2 | Phase 1 is achievable with zero external assets (no new model files required) |
| G3 | Marker rendering is additive — it does not alter the simulation engine or physics state |
| G4 | Three-phase escalation from primitive sphere → IQM model → full textured model |
| G5 | Markers scale correctly relative to the solar system (human-sized ship, not planet-sized) |
| G6 | LOD rules prevent marker overdraw at planetary distances |

---

## 2. Non-Goals (this feature)

- Collision geometry for client markers (deferred to F-009 follow-on)
- Custom model upload by clients (Phase 3 only, from a server-side catalog)
- Audio indicators tied to marker proximity (future)
- NPC-specific marker appearance (handled in F-022 when NPC automation ships)

---

## 3. Scale and Sizing

Client ships are human-sized spacecraft. Reference sizes in sim units (1 sim unit = 1 AU / 100):

| Object | Real size | Sim unit radius |
|--------|-----------|-----------------|
| Earth | 12,742 km | ~0.085 |
| ISS (reference "small ship") | ~100 m | ~6.7 × 10⁻⁷ |
| "Fighter-sized" ship | ~30 m | ~2.0 × 10⁻⁷ |
| Phase 1 sphere (visible at simulation scale) | display radius — see §4.1 | fixed screen-space minimum |

At solar-system scale a human-sized ship is sub-pixel. Two display strategies are required:

1. **Near-camera regime** (camera within ~1 sim unit of the marker): render at true scale.
2. **Far-camera regime** (camera more than ~1 sim unit away): render at a fixed screen-space
   minimum radius (e.g., 4 px), so the marker remains visible regardless of zoom level.
   This is the same strategy used for distant star rendering.

The transition threshold is configurable in `configs/app.json` under `"marker_far_threshold_sim_units"`.

---

## 4. Phase 1 — Blinking Sphere

### 4.1 Appearance

- Shape: `DrawSphere` (Raylib)
- Base color: `ClientSession.Color` (RGB, server-assigned)
- Blink behavior: alpha oscillates on a per-client independent phase offset
  so two clients with the same blink period do not synchronize visually
- Period: 1.5 s default; configurable
- Near-camera radius: `0.001` sim units
- Far-camera radius: enforced screen-space minimum of 4 px (see §3)
- Label: client `Label` drawn above the sphere using `DrawText3D` or a 2D projected overlay;
  fades out beyond `"label_visible_distance_sim_units"` (default: 0.5 sim units)

### 4.2 Own-marker

The local client's own marker is rendered at reduced opacity (30%) to avoid obscuring the
player's own view. It is always visible regardless of distance.

### 4.3 Blink algorithm

```
phase_offset = hash(SessionID) % 2π   // deterministic per session
alpha = 0.5 + 0.5 * sin(time * (2π / period) + phase_offset)
color.A = uint8(alpha * 255)
```

### 4.4 Renderer integration point

The Raylib render loop (in `internal/client/go/raylib/ui/render/`) reads client sessions
from the `WorldSnapshot` after the main scene draw and before the HUD overlay. Marker
rendering is a distinct render pass so it does not interfere with body rendering.

No changes to `internal/sim/engine` are required for Phase 1.

---

## 5. Phase 2 — Primitive IQM Model

### 5.1 Target appearance

- Replace the sphere with a low-polygon IQM model (Raylib native format)
- Model is tinted by `ClientSession.Color` (shader uniform `colDiffuse`)
- Animation: idle rotation around the model's Y-axis while stationary; ceases when thrusting
- Default model: a simple diamond/octahedron shape (~200 triangles) shipped as a committed
  asset at `data/assets/models/client_ship_default.iqm`
- The model must have a defined bounding sphere for frustum culling and camera collision (F-001)

### 5.2 Asset requirements

| Asset | Path | Format | Committed |
|-------|------|--------|-----------|
| Default ship model | `data/assets/models/client_ship_default.iqm` | IQM | Yes |
| Default ship texture (optional tint base) | `data/assets/models/client_ship_default.png` | PNG | Yes |

### 5.3 Renderer integration point

Marker rendering uses `LoadModel` / `DrawModel` (same path as F-008 Artifact objects).
If F-008 ships before Phase 2, the artifact render pipeline is reused directly.
If Phase 2 ships before F-008, a minimal model-draw path is introduced in the marker
render pass and later unified with F-008.

---

## 6. Phase 3 — Stock and Custom Textured Models

### 6.1 Stock catalog

A server-side model catalog is defined in `data/assets/models/catalog.json`:

```json
{
  "models": [
    { "id": "default",     "name": "Scout",   "path": "data/assets/models/client_ship_default.iqm" },
    { "id": "fighter",     "name": "Fighter", "path": "data/assets/models/fighter.iqm" },
    { "id": "freighter",   "name": "Freighter", "path": "data/assets/models/freighter.iqm" }
  ]
}
```

### 6.2 Model selection

Clients specify their desired model via `MarkerRef` in `RegisterClientRequest`. The server
validates the ID against the catalog; unknown IDs fall back to `"default"`.

### 6.3 Custom textures

Phase 3 allows clients to supply a texture ID from a pre-approved server-side texture
library (same `data/assets/textures/` manifest as existing texture loading). Client-uploaded
textures are explicitly out of scope — the server never accepts binary uploads from clients.

### 6.4 Color application in Phase 3

When a custom texture is present, `ClientSession.Color` is applied as a tint multiplier
over the base texture, preserving texture detail while maintaining client color identity.

---

## 7. LOD Rules

| Camera distance to marker | Render behavior |
|--------------------------|-----------------|
| < near threshold (0.01 sim units) | Full model at true scale; label always visible |
| near threshold – 1 sim unit | Model at screen-space minimum; label visible |
| 1 – 10 sim units | Model collapsed to blinking sphere (Phase 1 appearance) regardless of phase |
| > 10 sim units | Point-sprite only (1–2 px dot in player color); no label |
| > 100 sim units | Not rendered (culled) |

LOD thresholds are configurable in `configs/app.json` under `"marker_lod"`.

---

## 8. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/ui/render/markers.go` | **Create** | `drawClientMarkers(snapshot WorldSnapshot)` function; blink algorithm; screen-space minimum enforcement; label overlay |
| `internal/client/go/raylib/ui/render/markers_test.go` | **Create** | Blink alpha calculation; LOD threshold logic; own-marker opacity |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Call `drawClientMarkers(snapshot)` after scene draw, before HUD overlay |
| `configs/app.json` | **Modify** | Add `"marker_far_threshold_sim_units"`, `"marker_blink_period_s"`, `"label_visible_distance_sim_units"`, `"marker_lod"` block |

### Phase 2 files

| File | Action | Notes |
|------|--------|-------|
| `data/assets/models/client_ship_default.iqm` | **Add** | Committed low-poly IQM model (~200 triangles) |
| `data/assets/models/client_ship_default.png` | **Add** | Base texture for tint application |
| `internal/client/go/raylib/ui/render/markers.go` | **Modify** | Replace `DrawSphere` with `LoadModel` + `DrawModel`; apply `ClientSession.Color` as tint |

### Phase 3 files

| File | Action | Notes |
|------|--------|-------|
| `data/assets/models/catalog.json` | **Create** | Model catalog listing all stock ship models |
| `data/assets/models/fighter.iqm` | **Add** | Stock fighter model asset |
| `data/assets/models/freighter.iqm` | **Add** | Stock freighter model asset |
| `internal/client/go/raylib/ui/render/markers.go` | **Modify** | Catalog lookup by `MarkerRef`; unknown ID falls back to `"default"` |

---

## 9. Phase Details

### Phase 1 — Blinking Sphere

**Architectural layer**: Raylib UI render layer (`internal/client/go/raylib/ui/render/`).
**Prerequisites**: F-020 Phase 2 complete (ClientSessions in WorldSnapshot with position data).

Work items:
- [ ] Create `markers.go` with `drawClientMarkers()` reading `WorldSnapshot.ClientSessions`
- [ ] Implement blink algorithm: `alpha = 0.5 + 0.5*sin(time*(2π/period) + phase_offset)`
- [ ] Phase offset: `hash(SessionID) % (2π)` so clients don't synchronize
- [ ] Screen-space minimum (4 px): use `GetWorldToScreen` to check projected size; if < min, clamp
- [ ] Own-client marker: 30% opacity; always rendered regardless of distance
- [ ] LOD: > 10 sim units → point sprite only; > 100 sim units → cull
- [ ] Label rendering: `DrawText3D` above sphere; fades beyond `label_visible_distance_sim_units`
- [ ] Call `drawClientMarkers()` from `render.go` after scene draw, before HUD
- [ ] Unit tests: blink alpha at t=0/period/period*2; LOD thresholds; own-marker flag

Acceptance criteria:
- 100 simultaneously rendered markers do not drop frame rate below 60 fps ✓
- Each marker blinks independently (no visible synchronization) ✓
- Own-client marker renders at 30% opacity ✓
- Markers > 100 sim units are culled (not rendered) ✓

### Phase 2 — Primitive IQM Model

**Architectural layer**: Raylib UI render layer; asset loading.
**Prerequisites**: Phase 1 complete; IQM model asset committed.

Work items:
- [ ] Add `client_ship_default.iqm` and `client_ship_default.png` to `data/assets/models/`
- [ ] Load model once at startup; store in render state (not per-frame)
- [ ] Replace `DrawSphere` with `DrawModel` in `drawClientMarkers()`
- [ ] Apply `ClientSession.Color` as tint via `Model.Materials[0].Maps[MATERIAL_MAP_DIFFUSE].Color`
- [ ] Respect CGo interior-pointer rule (user memory): copy color to fresh local var before passing to shader

### Phase 3 — Stock and Custom Textured Models

**Architectural layer**: Raylib UI render layer; asset catalog.
**Prerequisites**: Phase 2 complete.

Work items:
- [ ] Create `data/assets/models/catalog.json` with at least 3 stock entries
- [ ] Load catalog at startup; index by `id`
- [ ] Resolve `ClientSession.MarkerRef` → catalog entry; fall back to `"default"` if unknown
- [ ] Load model for each unique `MarkerRef` in use; cache (don't reload per frame)

---

## 10. Phases Summary

| Phase | Trigger | Requires |
|-------|---------|----------|
| Phase 1 | F-020 Phase 2 (position streaming) | No new assets |
| Phase 2 | F-008 (Artifact object type) or standalone | IQM model asset |
| Phase 3 | Phase 2 complete; model catalog defined | Catalog JSON + model assets |

---

## 9. Open Questions

| # | Question | Decision needed by |
|---|----------|--------------------|
| Q1 | Should the own-marker sphere be visible in first-person view (obscures forward vector)? | Phase 1 |
| Q2 | Should labels always render in screen-space (billboard) or true 3D? | Phase 1 |
| Q3 | Should Phase 2 model be a separate committed asset or generated procedurally? | Phase 2 start |
| Q4 | Phase 3 texture tint: multiplicative blend vs. additive? | Phase 3 start |
