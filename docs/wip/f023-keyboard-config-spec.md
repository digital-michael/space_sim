# F-023 — Keyboard Configuration

## Purpose

Define a hardware-profile-aware, fully user-remappable keyboard/input configuration system
for Space Sim. This spec supersedes and replaces the informal stubs in F-006 (XYZ Keyboard
Navigation) and F-007 (User-Configurable Key Bindings). F-006 and F-007 are fulfilled by
this feature.

Read this alongside:
- [`docs/standards/agent-readme.md`](../standards/agent-readme.md)
- [`docs/standards/coding-standards.md`](../standards/coding-standards.md)
- [`docs/wip/f022-client-movement-spec.md`](f022-client-movement-spec.md) — movement commands driven by these bindings
- [`docs/wip/f024-multiplayer-hud-spec.md`](f024-multiplayer-hud-spec.md) — HUD toggle bindings
- [`docs/wip/f033-ship-definition-spec.md`](f033-ship-definition-spec.md) — engine stage actions (`move.engine_stage_up/down`) defined here

Related:
- **F-032** in `todo.md` — integration gate: all hardcoded `rl.IsKeyPressed`/`IsKeyDown` sites in `input.go` must be converted to vocabulary actions. §3 is the exhaustive action list; F-032 tracks the conversion work item.

## Last Updated
2026-05-18

## Status
🔄 In Progress — Phase 1 partial

The `internal/client/go/raylib/input/` package exists with `actions.go`, `keymap.go`, and `loader.go` implemented. Four actions are wired (`ui.settings`, `ui.fullscreen`, `sim.track_next`, `sim.track_stop`). All remaining hardcoded keys are tracked in **F-032** and must be converted before Phase 1 is complete.

---

## 1. Goals

| # | Goal |
|---|------|
| G1 | Ship all movement and simulation bindings in named stock profiles (no manual config required at first run) |
| G2 | Users can override any binding via `configs/keybindings.json` without editing source code |
| G3 | Stock profiles cover: laptop keyboard, full desktop keyboard, mouse+keyboard, numpad/10-key |
| G4 | Gamepad (controller) support is stubbed in the schema as a reserved profile; not implemented in Phase 1 |
| G5 | Bindings are hot-reloaded at runtime (REPL command or file-watch signal) without restart |
| G6 | Conflicting bindings are detected at load time and reported as errors, not silently applied |
| G7 | Key binding config is separate from `configs/app.json` and versioned independently |

---

## 2. Non-Goals (this feature)

- Mouse gesture recognizer (complex gesture sequences deferred)
- Macros / binding sequences (single key → multiple actions deferred)
- On-screen key binding remapper GUI (REPL-only in Phase 1)
- Gamepad / joystick hardware input (schema-reserved; not implemented)

---

## 3. Action Vocabulary

All bindable actions are named strings. The action vocabulary is the single source of truth
for what can be bound; adding a binding for an undefined action is a config error.

### 3.1 Camera and view actions

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `camera.pitch_up` | `UP` | Pitch camera upward |
| `camera.pitch_down` | `DOWN` | Pitch camera downward |
| `camera.yaw_left` | `LEFT` | Yaw camera left |
| `camera.yaw_right` | `RIGHT` | Yaw camera right |
| `camera.roll_left` | `Q` | Roll camera counter-clockwise |
| `camera.roll_right` | `E` | Roll camera clockwise |
| `camera.zoom_in` | `EQUAL` (`+`) | Zoom in |
| `camera.zoom_out` | `MINUS` (`-`) | Zoom out |
| `camera.reset` | `R` | Reset camera to default orientation |
| `camera.toggle_free_fly` | `F` | Toggle between free-fly and tracking mode |

### 3.2 Movement / locomotion actions (F-022)

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `move.thrust_forward` | `W` | Thrust along +POV vector |
| `move.thrust_backward` | `S` | Thrust along -POV vector |
| `move.thrust_left` | `A` | Thrust along -strafe vector |
| `move.thrust_right` | `D` | Thrust along +strafe vector |
| `move.thrust_up` | `SPACE` | Thrust along +up vector |
| `move.thrust_down` | `LEFT_CONTROL` | Thrust along -up vector |
| `move.brake` | `X` | Zero velocity (emergency stop) |
| `move.drift_toggle` | `Z` | Toggle drift mode (no thrust, gravity only) |

### 3.3 Simulation control actions

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `sim.timescale_increase` | `CMD+SHIFT+PERIOD` | Step simulation time scale up (real-time → 1 yr/sec) |
| `sim.timescale_decrease` | `CMD+SHIFT+COMMA` | Step simulation time scale down |
| `sim.tick_speed_increase` | `ALT+PERIOD` | Increase physics tick rate (0 → 100%) |
| `sim.tick_speed_decrease` | `ALT+COMMA` | Decrease physics tick rate |
| `sim.pause_toggle` | `P` | Pause / unpause simulation |
| `sim.track_next` | `T` | Cycle track target forward |
| `sim.track_stop` | `ESCAPE` | Stop tracking |
| `sim.dataset_increase` | `ALT+EQUAL` | Increase asteroid belt dataset level (0-3) |
| `sim.dataset_decrease` | `ALT+MINUS` | Decrease asteroid belt dataset level |

**Note**: `sim.timescale` controls simulated seconds per real second (SecondsPerSecond); `sim.tick_speed` controls how frequently the physics engine ticks per real second. They are independent.

### 3.4 UI and HUD actions

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `hud.toggle` | `H` | Show / hide full HUD |
| `hud.client_list` | — | Show / hide client session list overlay |
| `hud.help` | `F1` | Show / hide help overlay |
| `ui.fullscreen` | `F11` | Toggle fullscreen |
| `ui.settings` | `F1` | Open / close settings dialog |
| `ui.system_selector` | `CMD+S` | Open the system selector dialog |
| `ui.mouse_mode_toggle` | `ALT+M` | Toggle mouse-look mode (cursor hidden / captured) |
| `ui.label_cycle` | `ALT+L` | Cycle label display mode: off → all → nearest → off |
| `ui.infra_cycle` | `I` | Cycle infrastructure ambient-light mode (off → spotlight → reserved) |
| `ui.record_toggle` | `ALT+R` | Start or stop screen recording |
| `ui.record_pause` | `ALT+SHIFT+R` | Pause or resume an active recording |
| `ui.quit` | `ALT+Q` | Quit the application |

### 3.5 REPL actions

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `repl.open` | `GRAVE` (`` ` ``) | Open / focus REPL input |
| `repl.close` | `ESCAPE` | Close REPL input |
| `repl.history_prev` | `UP` (in REPL context) | Previous REPL history entry |
| `repl.history_next` | `DOWN` (in REPL context) | Next REPL history entry |

### 3.6 Navigation actions

Navigation actions operate on the camera/tracking system. Most are mode-conditional (only
active in tracking mode or free-fly mode as noted).

| Action | Default (laptop) | Mode | Description |
|--------|-----------------|------|-------------|
| `nav.jump` | `J` | Free-fly | Open jump-to-object dialog |
| `nav.system_selector` | `CMD+S` (alias for `ui.system_selector`) | Any | Open system selector |
| `nav.child_next` | `F` | Tracking | Drill down to closest child body in orbital hierarchy |
| `nav.parent` | `B` | Tracking | Move up to parent body in orbital hierarchy |
| `nav.sibling_next` | `TAB` | Tracking | Cycle to next sibling body (same parent, same category) |
| `nav.sibling_prev` | `SHIFT+TAB` | Tracking | Cycle to previous sibling body |
| `camera.center` | `C` | Any | Free-fly: point camera at origin. Tracking: reset zoom to auto-zoom distance |

**Note**: `nav.child_next` and `camera.toggle_free_fly` both default to `F`. The `camera.toggle_free_fly` action switches *mode* from tracking to free-fly; `nav.child_next` only fires while *in* tracking mode. Context separation avoids the conflict: `F` in tracking mode → `nav.child_next`; `F` in free-fly mode → `camera.toggle_free_fly`.

### 3.7 Engine stage actions (F-033)

These actions are defined in this vocabulary but only take effect once F-033 Phase 3
(Engine Stage Selection) is implemented.

| Action | Default (laptop) | Description |
|--------|-----------------|-------------|
| `move.engine_stage_up` | `SHIFT+W` or `PAGE_UP` | Advance to the next higher engine stage |
| `move.engine_stage_down` | `SHIFT+S` or `PAGE_DOWN` | Drop to the next lower engine stage |

---

## 4. Config File Format

### 4.1 Location

`configs/keybindings.json`

This file is separate from `configs/app.json`. It is optional: if absent, the active
profile's stock bindings are used. If present, it overrides specific bindings and
specifies which stock profile to use as the base.

### 4.2 Schema

```json
{
  "$schema": "https://space-sim/schemas/keybindings/v1",
  "version": 1,
  "base_profile": "laptop",
  "overrides": [
    {
      "action": "move.thrust_forward",
      "key":    "I",
      "mods":   []
    },
    {
      "action": "move.thrust_backward",
      "key":    "K",
      "mods":   []
    }
  ]
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `version` | integer | yes | Schema version; currently `1` |
| `base_profile` | string | yes | One of: `laptop`, `full_keyboard`, `mouse_keyboard`, `numpad` |
| `overrides` | array | no | Per-action key overrides applied on top of `base_profile` |
| `action` | string | yes (in override) | Must be a known action from the vocabulary (§3) |
| `key` | string | yes (in override) | Raylib key name constant (case-insensitive) |
| `mods` | array of string | no | Modifier keys: `SHIFT`, `CTRL`, `ALT` |

### 4.3 Key name format

Key names follow Raylib's `KEY_*` constant names with the `KEY_` prefix stripped and
lowercased for readability:
- `"W"` → `KEY_W`
- `"LEFT_CONTROL"` → `KEY_LEFT_CONTROL`
- `"SPACE"` → `KEY_SPACE`
- `"F1"` → `KEY_F1`
- `"KP_0"` → `KEY_KP_0` (numpad 0)

A complete key name reference is generated at startup and available via `help keys` REPL command.

---

## 5. Stock Profiles

### 5.1 Profile: `laptop`

Optimized for keyboards without a dedicated numpad. Uses WASD for thrust, arrow keys for
camera orientation, modifier-free bindings throughout.

Full binding table: see §3 action tables (defaults listed are the laptop profile defaults).

### 5.2 Profile: `full_keyboard`

Uses WASD for movement and the numpad for camera control. Expands camera roll and zoom to
numpad keys.

| Action | Key |
|--------|-----|
| `camera.pitch_up` | `KP_8` |
| `camera.pitch_down` | `KP_2` |
| `camera.yaw_left` | `KP_4` |
| `camera.yaw_right` | `KP_6` |
| `camera.roll_left` | `KP_7` |
| `camera.roll_right` | `KP_9` |
| `camera.zoom_in` | `KP_ADD` |
| `camera.zoom_out` | `KP_SUBTRACT` |
| All other actions | Same as `laptop` |

### 5.3 Profile: `mouse_keyboard`

Mouse drives camera orientation (yaw/pitch via mouse delta; roll on `Q`/`E`). WASD for
thrust. Arrow keys freed for other use.

| Action | Key/Input |
|--------|-----------|
| `camera.pitch_up/down` | Mouse Y delta |
| `camera.yaw_left/right` | Mouse X delta |
| `camera.roll_left` | `Q` |
| `camera.roll_right` | `E` |
| `camera.zoom_in/out` | Mouse wheel |
| All movement actions | Same as `laptop` |

Mouse sensitivity is configurable in `configs/app.json` under `"mouse_sensitivity"` (existing field).

### 5.4 Profile: `numpad`

Uses the numpad exclusively for 6-DOF movement. Useful for one-handed navigation while the
other hand uses the mouse.

| Action | Key |
|--------|-----|
| `move.thrust_forward` | `KP_8` |
| `move.thrust_backward` | `KP_2` |
| `move.thrust_left` | `KP_4` |
| `move.thrust_right` | `KP_6` |
| `move.thrust_up` | `KP_9` |
| `move.thrust_down` | `KP_3` |
| `camera.yaw_left` | `KP_7` |
| `camera.yaw_right` | `KP_1` |
| `camera.zoom_in` | `KP_ADD` |
| `camera.zoom_out` | `KP_SUBTRACT` |
| `move.brake` | `KP_5` |

### 5.5 Profile: `gamepad` (reserved, not implemented in Phase 1)

Schema slot reserved. No bindings defined. Loading this profile in Phase 1 logs a warning
and falls back to `laptop`.

---

## 6. Conflict Detection

At load time, all bindings (stock profile + overrides) are checked for duplicates.
A conflict is: two different actions bound to the same `(key, mods)` combination within
the same context.

Context separation: REPL context and world context share no bindings. `ESCAPE` in REPL
context closes the REPL; `ESCAPE` in world context stops tracking. This is not a conflict.

Conflict handling:
- **Hard conflict** (two world actions, same key): load fails with a descriptive error
  listing both action names and the conflicting key. No fallback; user must fix the config.
- **Override shadows stock**: not an error; the override wins silently.

---

## 7. Hot-Reload

Bindings can be reloaded at runtime without restarting:

- **REPL command**: `config reload keybindings` — re-reads `configs/keybindings.json`,
  applies diff, prints changed bindings.
- **File watch**: optional; enabled in `configs/app.json` with `"watch_keybindings": true`.
  Uses `inotify`/`kqueue` via Go stdlib `os.ReadFile` polled every 2 seconds (no external
  watcher dependency).

Hot-reload does not interrupt in-progress held-key state. Currently-held keys complete
their action under the old binding; the new binding takes effect on the next press.

---

## 8. Implementation Notes

### 8.1 Current state

`internal/client/go/raylib/app` handles input today through `handleInput()` with hardcoded
Raylib key constants. The refactor path is:

1. Define `InputAction` enum covering all vocabulary entries.
2. `Binding` maps `(rlKey int32, mods ModSet)` → `InputAction`.
3. `KeyMap` is a slice of bindings indexed by `InputAction`; built at startup from the active profile + overrides.
4. `handleInput()` replaces all `rl.IsKeyDown(rl.KEY_W)` calls with `km.IsDown(InputAction_ThrustForward)`.
5. `KeyMap` is hot-swappable via an atomic pointer; no lock needed on the read path.

### 8.2 TD-001 dependency

The existing 14-parameter `handleInput` / `updateCameraState` function signature (TD-001
in `todo.md`) should be cleaned up before or during Phase 1 of this feature. Adding a
`KeyMap` parameter to a 14-parameter function is the wrong direction.

---

## 9. Files to Touch

### Phase 1 files

| File | Action | Notes |
|------|--------|-------|
| `internal/client/go/raylib/input/actions.go` | **Create** | `InputAction` enum (all §3 vocabulary); `String()` stringer |
| `internal/client/go/raylib/input/keymap.go` | **Create** | `Binding`, `KeyMap`, `Profile` types; `IsDown()`, `IsPressed()`, `JustReleased()`; **`DrainQueue()` called once per frame via `rl.GetKeyPressed()` to capture short-tap events that `IsKeyPressed` misses under load** |
| `internal/client/go/raylib/input/loader.go` | **Create** | Load stock profile from `data/profiles/`; merge `configs/keybindings.json` overrides |
| `internal/client/go/raylib/input/conflict.go` | **Create** | Conflict detection; returns `ConflictError` with human-readable description |
| `internal/client/go/raylib/input/actions_test.go` | **Create** | Enum stability regression; `String()` round-trip |
| `internal/client/go/raylib/input/keymap_test.go` | **Create** | Conflict detection, override merge, key name parsing, IsDown delegation |
| `data/profiles/laptop.json` | **Create** | Stock laptop profile bindings |
| `data/profiles/mouse_keyboard.json` | **Create** | Stock mouse+keyboard profile |
| `configs/keybindings.json` | **Create** | Empty override template (committed; user edits locally) |
| `.gitignore` | **Modify** | If `configs/keybindings.json` should be user-local, add to gitignore; otherwise omit |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Replace all `rl.IsKeyDown(rl.KEY_*)` / `rl.IsKeyPressed(rl.KEY_*)` with `km.IsDown(InputAction_*)` / `km.IsPressed(InputAction_*)` |
| `internal/client/go/raylib/app/app.go` | **Modify** | Add `keyMap *input.KeyMap` field; load in `New()`; hot-swap via `atomic.Pointer[input.KeyMap]` |
| `internal/client/repl/repl.go` | **Modify** | Add `config reload keybindings` and `help keys` dispatch cases |

### Phase 2 files

| File | Action | Notes |
|------|--------|-------|
| `data/profiles/full_keyboard.json` | **Create** | Full keyboard stock profile |
| `data/profiles/numpad.json` | **Create** | Numpad navigation stock profile |

### Phase 3 files

| File | Action | Notes |
|------|--------|-------|
| `data/profiles/mouse_keyboard.json` | **Modify** | Add mouse-delta camera bindings |
| `internal/client/go/raylib/app/interactive.go` | **Modify** | Add mouse-delta camera handling branch; activate when mouse+keyboard profile loaded |

---

## 10. Phases

### Phase 1 — Laptop + Mouse-Keyboard Profiles

**Architectural layer**: Raylib client input layer (new package `internal/client/go/raylib/input/`), Raylib app layer (`internal/client/go/raylib/app/`), REPL client.
**Prerequisites**: TD-001 resolved (`handleInput` param consolidation) before or during this phase.

**Current state**: `input` package created; `actions.go`, `keymap.go`, `loader.go` exist. Four actions are wired in `input.go` (`ui.settings`, `ui.fullscreen`, `sim.track_next`, `sim.track_stop`). F-032 tracks the remaining conversion.

**Value**: All current keyboard interactions work via named bindings; users can override
from `configs/keybindings.json`.

Work items:
- [x] Define `InputAction` enum (initial set — §3.1, §3.2, §3.3 partial, §3.4 partial, §3.5)
- [x] Define `Binding`, `KeyMap`, `Profile` types in `internal/client/go/raylib/input/`
- [x] Implement profile loader: reads `data/profiles/` stock profiles
- [x] Implement config merger: applies `configs/keybindings.json` overrides
- [x] Conflict detection with clear error messages
- [x] `KeyMap.DrainQueue()` implemented: drains `rl.GetKeyPressed()` into per-frame pressed-set
- [ ] **F-032 — Gap closure**: Convert all remaining hardcoded `rl.IsKeyPressed`/`rl.IsKeyDown` sites in `input.go` to vocabulary actions; extend `actions.go` with the §3.3/§3.4/§3.6 additions above
- [ ] Expand `actions.go` with new §3.3 actions: `sim.timescale_increase/decrease`, `sim.tick_speed_increase/decrease`, `sim.dataset_increase/decrease`
- [ ] Add §3.4 additions: `ui.system_selector`, `ui.mouse_mode_toggle`, `ui.label_cycle`, `ui.infra_cycle`, `ui.record_toggle`, `ui.record_pause`, `ui.quit`
- [ ] Add §3.6 navigation actions: `nav.jump`, `nav.child_next`, `nav.parent`, `nav.sibling_next`, `nav.sibling_prev`, `camera.center`
- [ ] Update stock profile JSON files to include new action defaults
- [ ] REPL: `config reload keybindings`; `help keys` to print active bindings
- [ ] Unit tests: enum stability for new constants; conflict detection roundtrip

Acceptance criteria:
- All existing keyboard behaviors work identically after refactor ✓
- No `rl.IsKeyPressed`/`rl.IsKeyDown` calls in `handleInput` outside of UI overlay subsections ✓
- Invalid key name in `configs/keybindings.json` prints error on load and exits ✓
- Hard conflict in overrides prints error on load and exits ✓
- `config reload keybindings` applies changes at runtime ✓

### Phase 2 — Full Keyboard + Numpad Profiles

**Architectural layer**: Input package, profile data files.
**Prerequisites**: Phase 1 complete.

**Value**: Adds numpad navigation for desktop users.

Work items:
- [ ] Commit `full_keyboard` and `numpad` stock profiles to `data/profiles/`
- [ ] `base_profile: "full_keyboard"` and `"numpad"` work correctly

### Phase 3 — Mouse Integration

**Value**: Mouse-driven camera orientation; scroll wheel zoom.

Work items:
- [ ] `mouse_keyboard` profile: replace arrow key camera with mouse delta
- [ ] Mouse sensitivity configurable; exposed via existing `configs/app.json` field
- [ ] Scroll wheel zoom works in `mouse_keyboard` profile

---

## 10. Open Questions

| # | Question | Decision needed by |
|---|----------|--------------------|
| Q1 | Should stock profile JSON files live in `data/profiles/` (committed) or embedded as Go `//go:embed` assets? | Phase 1 |
| Q2 | Should `help keys` output sorted by action group or by key? | Phase 1 |
| Q3 | Context separation: should REPL context bindings be a separate config section or inferred from action name prefix? | Phase 1 |
| Q4 | Should `mouse_keyboard` sensitivity apply only when right-mouse is held (FPS-style), or always? | Phase 3 |
