# F-024 — Multiplayer HUD Enhancements

## Purpose

Define the HUD (heads-up display) additions needed to support multi-client awareness.
The single-player HUD today shows simulation state, selection info, and debug metrics.
This feature adds the client session overlay, proximity indicators, and role badges
needed for a coherent multi-player experience.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md)
- [`docs/wip/f020-multi-client-spec.md`](f020-multi-client-spec.md) — session data the HUD reads
- [`docs/wip/f021-physical-marker-spec.md`](f021-physical-marker-spec.md) — in-world marker is distinct from HUD overlay
- [`docs/wip/f023-keyboard-config-spec.md`](f023-keyboard-config-spec.md) — HUD toggle bindings

## Last Updated
2026-05-11

## Status
📋 Not started

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Player always knows who else is connected and their approximate location |
| G2 | Off-screen clients have a direction indicator so they can be located |
| G3 | Admin clients have a dedicated overlay listing all sessions with raw position data |
| G4 | HUD additions do not regress existing single-player HUD performance |
| G5 | All new HUD panels are individually toggleable via keybindings |

---

## 2. Non-Goals (this feature)

- In-game chat system (future feature)
- Mini-map or 3D radar with full positional accuracy (deferred; depends on coordinate projection work)
- Achievement or scoring UI
- Custom HUD theme/skin system

---

## 3. New HUD Components

### 3.1 Client Session List (Tab overlay)

**Trigger**: `hud.client_list` binding (default `TAB`).

**Layout**: Fixed-position panel, top-right corner. Scrollable if > 10 entries.

| Column | Content |
|--------|---------|
| Color swatch | 16 × 16 px block in `ClientSession.Color` |
| Label | `ClientSession.Label` (truncated at 24 chars) |
| Role badge | Short string: `[P]` player, `[N]` NPC, `[A]` admin, `[?]` other |
| Distance | Formatted distance to this client in sim units (e.g., `0.42 AU`, `12.3 AU`) |
| Status | `●` green = active last 5 s; `●` yellow = idle 5–30 s |

Own session: highlighted row (lighter background).

**Data source**: `WorldSnapshot.ClientSessions` (added in F-020 Phase 2).

### 3.2 Nearby Client Compass Indicators

When a client is off-screen (outside the camera frustum), a directional indicator appears
on the screen edge pointing toward them. Maximum 5 off-screen indicators shown simultaneously
(nearest 5 by distance). Own marker is excluded.

**Indicator appearance**:
- Small triangle (10 px) in `ClientSession.Color`
- `ClientSession.Label` (truncated at 12 chars) printed adjacent
- Distance text below the label

**Position calculation**:
Project the client's world position through the camera's view-projection matrix. If the
projected point is outside NDC bounds, clamp to the nearest screen edge and draw the indicator there.

### 3.3 Proximity Alert

When any other client's position is within `proximity_alert_distance_sim_units` (default: 0.01
sim units, configurable), a flash overlay appears momentarily (0.5 s) in their color with
the text `<Label> NEARBY`.

This prevents surprise collisions during normal navigation.

### 3.4 Admin Session Panel (Admin-role only)

Visible only to clients with role = ADMIN. Activated by `hud.admin_panel` binding
(default: `F2`).

Full-screen overlay listing all sessions with:
- SessionID (truncated UUID)
- Label, role, color
- World position (3 floats, 4 decimal places, sim units)
- POV vector
- ConnectedAt timestamp
- LastSeen timestamp
- `[KICK]` button (keyboard-driven: navigate to row, press `K` to kick)

Admin panel does not render the compass or nearby-client list while open (too much overlap).

### 3.5 Own-Client Status HUD

Small permanent overlay bottom-left (alongside existing speed/time display):

```
─────────────────────────
 YOU: ExplorerOne [P] ●
 POS: 100.00  0.00  0.00
 SPD: 0.00 su/s
 MODE: THRUST
─────────────────────────
```

| Line | Content |
|------|---------|
| YOU | Own label, role badge, connection status dot |
| POS | Own world position in sim units |
| SPD | Scalar speed (magnitude of velocity vector) |
| MODE | Current movement mode: DRIFT, THRUST, IMPULSE, WARP |

This panel replaces the existing free-fly position debug info (if any) and consolidates
the player's situational awareness in one place.

---

## 4. Existing HUD Compatibility

The new components are additive. Existing HUD panels (simulation speed, object selection
info, performance metrics, help screen) are unchanged.

Render order (back to front):
1. Scene (bodies, markers)
2. Existing HUD panels
3. Nearby client compass indicators (screen-edge, world-projected)
4. Client session list (when open)
5. Admin panel (when open, full-screen)
6. Proximity alert flash (when triggered)

All new panels use the same DrawRectangle / DrawText Raylib calls as existing panels.
No new rendering dependencies are introduced.

---

## 5. Distance Formatting

| Raw distance (sim units) | Display format |
|--------------------------|----------------|
| < 0.01 | `< 0.01 AU` |
| 0.01 – 999.99 | `X.XX AU` |
| ≥ 1,000 | `X.X kAU` |
| ≥ 63,241 | `X.XX ly` (1 ly ≈ 63,241 AU) |

Conversion factor: 1 sim unit = 0.01 AU → displayed AU = sim_units × 0.01.

---

## 6. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/ui/render/hud_session.go` | **Create** | `drawOwnClientStatus()`, `drawClientSessionList()` functions; reads `WorldSnapshot.ClientSessions` |
| `internal/client/go/raylib/ui/render/hud_session_test.go` | **Create** | Distance formatter edge cases (0, sub-AU, AU, kAU, ly) |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Call `drawOwnClientStatus()` and (when TAB held) `drawClientSessionList()` in HUD render pass |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Wire TAB hold/toggle to session list visibility flag |
| `configs/app.json` | **Modify** | Add `"hud"` block: `client_list_max_rows`, `label_visible_distance_sim_units` |

### Phase 2 files

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/ui/render/hud_compass.go` | **Create** | `drawCompassIndicators()` (screen-edge projection); `drawProximityAlert()` (flash timer) |
| `internal/client/go/raylib/ui/render/hud_compass_test.go` | **Create** | Screen-edge projection unit tests; proximity threshold tests |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Call compass and proximity pass after scene draw |
| `configs/app.json` | **Modify** | Add `"proximity_alert_distance_sim_units"` |

### Phase 3 files

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/ui/render/hud_admin.go` | **Create** | `drawAdminPanel()`: full session table, keyboard navigation, `[KICK]` action |
| `internal/client/go/raylib/ui/render/render.go` | **Modify** | Gate admin panel on role == ADMIN; call in HUD pass |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Wire `hud.admin_panel` binding |

---

## 7. Phases

### Phase 1 — Own-Client Status + Session List

**Architectural layer**: Raylib UI render layer (`internal/client/go/raylib/ui/render/`), Raylib app layer.
**Prerequisites**: F-020 Phase 2 (ClientSessions in WorldSnapshot); F-023 Phase 1 (TAB binding via KeyMap).

**Value**: Player knows their own state and who else is connected.

Work items:
- [ ] Add `OwnClientStatusPanel` to render overlay; reads own session from `WorldSnapshot`
- [ ] Add `ClientSessionListPanel` (Tab overlay); reads all sessions from `WorldSnapshot`
- [ ] Distance formatting utility (testable, pure function)
- [ ] `hud.client_list` and `hud.toggle` bindings wired (F-023 Phase 1 dependency)
- [ ] Performance: 100-session list renders in < 1 ms
- [ ] Unit tests: distance formatter edge cases

Acceptance criteria:
- Own session always shows correct label, role, position, speed, mode ✓
- Session list shows all connected clients with correct distances ✓
- TAB toggle works; panel is dismissed cleanly ✓

### Phase 2 — Compass Indicators + Proximity Alert

**Architectural layer**: Raylib UI render layer.
**Prerequisites**: Phase 1 complete.

**Value**: Player can find off-screen clients and get warned of close encounters.

Work items:
- [ ] Screen-edge projection utility (pure function; testable)
- [ ] Compass indicator render pass (≤ 5 off-screen clients)
- [ ] Proximity alert flash (0.5 s timer; debounced per-client)
- [ ] Config: `proximity_alert_distance_sim_units` in `configs/app.json`

### Phase 3 — Admin Panel

**Architectural layer**: Raylib UI render layer, Raylib app layer.
**Prerequisites**: Phase 1 complete; F-020 Phase 3 (KickClient RPC).

**Value**: Admins have full session visibility and moderation tools.

Work items:
- [ ] Full-session table render; keyboard navigation
- [ ] `[KICK]` action wired to `SessionService.KickClient` RPC (F-020 Phase 3)
- [ ] Admin panel gated on `ClientRole == ADMIN`
- [ ] `hud.admin_panel` binding wired

---

## 7. Open Questions

| # | Question | Decision needed by |
|---|----------|--------------------|
| Q1 | Should the session list be sorted by distance (nearest first) or by connection time? | Phase 1 |
| Q2 | Distance: should it be 3D Euclidean or projected onto the ecliptic plane? | Phase 1 |
| Q3 | Should the proximity alert play a sound (requires audio system)? | Phase 2 |
| Q4 | Should the admin panel support search/filter by label? | Phase 3 |
