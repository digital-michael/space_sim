# F-038 — HUD Profiles

## Purpose
Replace the current ad-hoc collection of HUD panels with a structured, data-driven profile system. Five named profiles govern which panels are active, what information is shown, and what actions are permitted. Profiles are constrained by IAAM roles (F-011). This spec integrates and supersedes the non-role-gated components of **F-024 — Multiplayer HUD Enhancements**; F-024 is closed by this feature.

## Status
📋 Not started

## Last Updated
2026-05-24

## Depends On
- F-020 — Multi-Client Session Layer (session data feeds Client Session List, Proximity Indicators)
- F-021 — Player Physical Marker (own-ship position feeds Own-Client Status HUD)
- F-022 — Client Locomotion (movement mode, speed feeds Ship Status HUD)

## Soft Depends On (constrains panel availability, not build)
- F-011 — IAAM (Admin and Spectral profiles gated by role; gracefully degraded without F-011)
- F-037 — AI/NPC Console (Communications/Chat HUD hosts Personal Copilot)

## Unlocks
- F-037 Phase 1 — Personal Copilot chat panel slot is defined here

## F-024 Integration Notes
F-024 (Multiplayer HUD Enhancements) components are absorbed into this spec. Components mapped:
- F-024 §3.1 Client Session List → Player profile, System Overview panel (§4.3.4)
- F-024 §3.2 Compass Indicators → Player profile, Navigation HUD (§4.3.1)
- F-024 §3.3 Proximity Alert → Player profile, always-on overlay (§4.6)
- F-024 §3.4 Admin Session Panel → Admin profile (§4.4)
- F-024 §3.5 Own-Client Status HUD → Player profile, Ship Status HUD (§4.3.2)

**F-024 is closed by this feature.** Any F-024 acceptance criteria not covered in Phase 1 of F-038 are carried as Phase 2 items below.

---

## 1. Profile Overview

| # | Profile | Description | Role Requirement |
|---|---------|-------------|-----------------|
| 0 | **Debug** | Raw runtime diagnostics. Read-only. | None (dev/diagnostic) |
| 1 | **Educational** | Info HUDs, TAB/SHIFT-TAB, F/B navigation. No ship controls. | None |
| 2 | **Player** | Full game-play HUD: navigation, ship status, comms, system overview. | `user` or higher |
| 3 | **Admin** | Player profile + privileged simulation controls + enhanced debug. | `admin` |
| 4 | **Spectral** | Admin-enabled read-only view of another player's POV and data. | `admin` |

**Without F-011 IAAM**: Profiles 3 and 4 fall back to Profile 2 with a console warning. This is the pre-IAAM graceful degradation path; no crash, no blocked feature.

---

## 2. Data-Driven Profile Configuration

Profiles are defined in `configs/hud_profiles.json` (server-side) and optionally overridden per-client in a client config file. This makes the profile definitions changeable without code modification.

```json
{
  "schema_version": "1.0",
  "profiles": {
    "debug": {
      "id": 0,
      "panels": ["runtime_metrics", "frame_timing", "object_counts", "camera_state"],
      "always_on_overlays": [],
      "required_role": null
    },
    "educational": {
      "id": 1,
      "panels": ["info_hud", "selection_info"],
      "always_on_overlays": ["body_labels"],
      "required_role": null
    },
    "player": {
      "id": 2,
      "panels": ["navigation", "ship_status", "comms_chat", "system_overview"],
      "always_on_overlays": ["proximity_alert"],
      "required_role": "user",
      "panels_default_visible": ["navigation", "ship_status"],
      "panels_user_toggleable": true
    },
    "admin": {
      "id": 3,
      "panels": ["navigation", "ship_status", "comms_chat", "system_overview",
                 "admin_session_panel", "sim_controls", "runtime_metrics"],
      "always_on_overlays": ["proximity_alert"],
      "required_role": "admin",
      "panels_user_toggleable": true
    },
    "spectral": {
      "id": 4,
      "panels": ["spectral_pov", "spectral_data"],
      "always_on_overlays": [],
      "required_role": "admin",
      "panels_user_toggleable": false
    }
  }
}
```

`panels_default_visible`: which panels are shown on profile entry (others available but toggled off).  
`panels_user_toggleable`: whether the player can individually show/hide panels within the profile.

---

## 3. Profile 0 — Debug

Read-only runtime diagnostics visible in all modes as an overlay. Accessible via toggle keybinding from any other profile (does not change the active profile; overlays on top).

**Panels:**
- `runtime_metrics`: FPS, frame time, sim Hz, physics tick count, goroutine count.
- `frame_timing`: render time, sim tick time, snapshot clone time.
- `object_counts`: body count by category; asteroid dataset size.
- `camera_state`: camera position, POV vector, distance to tracked body, tracking mode.

No interaction (read-only). Available without authentication.

---

## 4. Profile 1 — Educational

For observatory, demo, or learning contexts. Shows informational HUDs; no ship navigation or game-play panels.

**Panels:**
- `info_hud`: body information panel for the currently selected body.
- `selection_info`: selected body name, type, orbital parameters, key physical properties.

**Always-on overlays:**
- `body_labels`: on-screen names for all visible named bodies above importance threshold.

Keybindings active: TAB / SHIFT+TAB (cycle selection), F / B (forward/back in orbit order), standard camera controls. Ship thrust and warp bindings are disabled in this profile.

---

## 5. Profile 2 — Player

The primary game-play profile. Four independently-toggleable panels aggregatable into a single view.

### 5.1 Navigation HUD (`navigation`)

Compass indicators, tracking status, warp readiness, and off-screen client pointers.

| Element | Description |
|---------|-------------|
| Off-screen client arrows | Screen-edge triangles pointing to off-screen players (nearest 5, in their colors) |
| Tracking indicator | Current track target name + distance |
| Warp charge meter | Available when F-022 warp mode is active |
| System bearing | Cardinal bearing relative to system center (SOL) |

### 5.2 Ship Status HUD (`ship_status`)

Live ship telemetry.

```
─────────────────────────
 YOU: ExplorerOne [P] ●
 POS:  100.00   0.00   0.00  sim-units
 SPD:  0.00 su/s
 MODE: THRUST
 HULL: 100%   POWER: 87%
─────────────────────────
```

Columns match F-024 §3.5 plus hull and power (from F-033 engine stage data).

### 5.3 Communications/Chat HUD (`comms_chat`)

- Inbound and outbound text messages (F-025 Ship-to-Ship Communications).
- Personal Copilot chat thread (F-037 Phase 1): conversation with the AI assistant.
- Channel selector: `[All]` `[Faction]` `[Private]`.
- Send field at bottom of panel; keybinding to focus/unfocus input.

When F-037 is not yet active, the Copilot thread area shows a placeholder message.

### 5.4 System Overview (`system_overview`)

The F-024 Client Session List (§3.1) lives here, plus:
- List of connected clients with color, label, role badge, distance, and activity status.
- Faction ownership summary (from F-035 universe state) if a scenario is active.
- Nearby artifact and station proximity list (within configurable range).

### 5.5 Proximity Alert (always-on overlay)

Flashes for 0.5 s in the approaching client's color when any other client enters `proximity_alert_distance` sim units. Label + "NEARBY" text. Suppressed during Admin panel full-screen mode.

### 5.6 Player Panel Toggles

Each panel has a dedicated keybinding (`hud.navigation`, `hud.ship_status`, `hud.comms`, `hud.system_overview`) for show/hide. All four panels can be active simultaneously ("aggregate view") or shown individually. State persists per-session.

---

## 6. Profile 3 — Admin

All Player panels plus privileged additions.

**Additional panels:**
- `admin_session_panel`: Full-screen session list (F-024 §3.4). All sessions, positions, raw coords, `[KICK]` and `[TELEPORT]` actions. Activated by `hud.admin_panel` binding.
- `sim_controls`: Pause/resume, time-scale slider, dataset selector, scenario event triggers. Read/write.
- `runtime_metrics`: Same as Debug profile but overlaid in a compact corner widget.

Admin profile enables the same independently-toggleable behavior as Player for the shared panels.

---

## 7. Profile 4 — Spectral

Admin-only. Observe another connected client's POV and HUD data without interacting.

**Panels:**
- `spectral_pov`: Renders the watched client's viewpoint (their camera position/orientation) as a secondary viewport. Implementation: render-to-texture at reduced resolution; drawn as a panel overlay. **Phase 2 — deferred if complex** (see §9).
- `spectral_data`: Read-only copy of the watched client's Ship Status HUD and Navigation HUD data.

**Phase 1 (low risk):** `spectral_data` only — text readout of the watched client's state, polled from `WorldSnapshot.ClientSessions`. No render-to-texture.  
**Phase 2 (higher risk):** Full `spectral_pov` with secondary viewport render. Deferred until render-texture infrastructure is proven (evaluate risk at Phase 1 completion).

Selecting a client to watch: Admin types `spectral <client-label>` in the REPL or navigates the admin session panel and presses a binding.

---

## 8. Profile Persistence and Selection

- Active profile stored in `configs/app.json` under `"hud_profile": "player"`.
- CLI flag `--hud-profile <name>` overrides the config at startup.
- Cycle keybinding (`hud.cycle_profile`) cycles through profiles the current role permits.
- Per-panel visibility state (within Player/Admin profiles) is stored per-session in memory; not persisted across restarts (Phase 1). Persisted as part of user profile in Phase 2.

**Pre-IAAM behavior (no F-011):**
- Profiles 0, 1, 2 available to all clients.
- Profiles 3 and 4 fall back to Profile 2 with a log warning.
- When F-011 lands, `required_role` is enforced by the server at session creation; the client receives a role claim that the HUD system reads.

---

## 9. Spectral Split-Screen Risk Assessment

Full secondary-viewport rendering (`spectral_pov`) requires:
1. `rl.LoadRenderTexture` for the secondary viewport.
2. A separate Raylib `BeginMode3D` / `EndMode3D` pass at the watched client's camera.
3. Blit of the render texture onto the screen as a panel overlay.

**Risk level**: Medium. Raylib render-to-texture is well-documented. The main risk is performance — a full second render pass at a non-trivial resolution. Mitigation: render at 50% resolution; cap at 30 Hz (independent of main loop FPS).

**Incremental path** (making Phase 2 easier):
- Phase 1: Ensure `ClientSession` exposes camera position and orientation in `WorldSnapshot` (may already be there from F-020).
- Between phases: Add a render texture utility function used for any HUD panel that needs offscreen rendering (e.g., mini-map); reuse for Spectral.
- Phase 2: Wire the Spectral viewport using the pre-built render texture infrastructure.

No structural changes are blocked on Spectral; the incremental steps are additions.

---

## 10. Files to Touch

| File | Change |
|------|--------|
| `configs/hud_profiles.json` | New file: profile definitions |
| `internal/client/go/raylib/ui/render/hud_profiles.go` | New: profile registry, active profile state |
| `internal/client/go/raylib/ui/render/panels/` | New sub-package: one file per panel |
| `internal/client/go/raylib/app/interactive.go` | Wire profile toggle keybindings; pass active profile to renderer |
| `internal/client/go/raylib/ui/render/renders.go` | Delegate panel rendering through profile registry |
| `configs/app.json` | Add `hud_profile` field |
| `internal/client/commands/actions.go` | Add `hud.*` InputAction constants |

---

## 11. Acceptance Criteria

### Phase 1
- [ ] `configs/hud_profiles.json` created with all 5 profiles.
- [ ] Debug (0) and Educational (1) profiles fully functional; all panels render correctly.
- [ ] Player (2) profile functional with all four panels (Navigation, Ship Status, Comms placeholder, System Overview).
- [ ] Per-panel toggle keybindings working; state persists within session.
- [ ] Proximity alert overlay fires correctly.
- [ ] Admin (3) profile functional with Admin Session Panel (F-024 §3.4 equivalent).
- [ ] Spectral (4) profile: `spectral_data` text panel only; no render-to-texture.
- [ ] Pre-IAAM graceful degradation: profiles 3/4 fall back to 2 with log warning.
- [ ] F-024 acceptance criteria satisfied (components absorbed here).
- [ ] All existing HUD panels (sim speed, selection info, help screen) preserved and accessible from appropriate profiles.
- [ ] All tests pass.

### Phase 2 (post F-011 + render-texture infrastructure)
- [ ] IAAM role claim enforced for profiles 3/4.
- [ ] Per-panel visibility persisted in user profile across restarts.
- [ ] Spectral `spectral_pov` render-to-texture viewport active.
- [ ] Personal Copilot chat thread live in Comms HUD (F-037 Phase 1 delivered).

---

## 12. Related Documents
- [docs/wip/f024-multiplayer-hud-spec.md](f024-multiplayer-hud-spec.md) — **Superseded by this spec**
- [docs/wip/f020-multi-client-spec.md](f020-multi-client-spec.md) — Session data source
- [docs/wip/f022-client-movement-spec.md](f022-client-movement-spec.md) — Ship telemetry source
- [docs/wip/f037-ai-npc-console-spec.md](f037-ai-npc-console-spec.md) — Personal Copilot in Comms HUD
- [docs/wip/f033-ship-definition-spec.md](f033-ship-definition-spec.md) — Hull/power data in Ship Status HUD
