# Runtime System Selector Plan

## Purpose
Track the concrete implementation checklist for the in-app JSON system selector while implementation and manual verification are still in progress.

## Last Updated
2026-03-28

## Table of Contents
1. Goal and Scope
	1.1 In Scope
	1.2 Out of Scope
2. Ownership and Constraints
3. Proposed Runtime Shape
4. Implementation Checklist
	4.1 Phase 1 - Selector State and Discovery
	4.2 Phase 2 - Modal Rendering and Input
	4.3 Phase 3 - Session Reload Flow
	4.4 Phase 4 - Validation and Docs
5. Validation Plan
6. Review Questions

## 1. Goal and Scope

Add a modal selector that lists JSON systems from [data/systems/](../../data/systems), opens from the running app with `Cmd+S`, closes without work when the current system is selected, and reloads the interactive session when a different system is chosen.

### 1.1 In Scope

- System-file discovery from [data/systems/](../../data/systems)
- Modal selector state and rendering
- `Cmd+S` binding for interactive mode
- Clean runtime session replacement for interactive mode
- Focused tests for selector state and reload decision logic
- Documentation updates after implementation and manual verification

### 1.2 Out of Scope

- Hot patching the existing simulation state in place
- Performance-mode support for live system switching
- Arbitrary file browsing outside [data/systems/](../../data/systems)
- Persisting the selected runtime system back into config files unless explicitly requested later

## 2. Ownership and Constraints

- [internal/space/app/](../../internal/space/app) owns system discovery, input dispatch, session lifecycle, and reload orchestration.
- [internal/space/ui/](../../internal/space/ui) owns modal selector state, just as it owns other dialog state today.
- [internal/space/raylib/ui/render/](../../internal/space/raylib/ui/render) owns selector drawing.
- [internal/space/](../../internal/space) remains the JSON loading layer and should not absorb app-level selector or reload workflow logic.
- Double-buffer lessons apply: do not mutate the live front/back simulation state to switch systems. Replace the running session instead.

## 3. Proposed Runtime Shape

1. Add a dedicated selector mode alongside the existing selection modes in [internal/space/ui/input.go](../../internal/space/ui/input.go).
2. Extend input state with selector-specific fields:
	- discovered system entries
	- selected row
	- active system path
	- optional pending reload target path
	- optional selector error/status text for runtime feedback
3. Open the selector from [internal/space/app/input.go](../../internal/space/app/input.go) only when interactive mode is active and no other modal dialog owns input.
4. Render a dedicated selector panel in [internal/space/raylib/ui/render/renders.go](../../internal/space/raylib/ui/render/renders.go) instead of overloading the object-selection tabs.
5. When the user confirms a different system, request an app-level session reload.
6. In the interactive loop, stop the old simulation, build a fresh runtime session from the new system path, start the new simulation goroutine, and continue the loop with the replacement session.

## 4. Implementation Checklist

### 4.1 Phase 1 - Selector State and Discovery

Objective: define the selector state model and collect valid system entries.

- [x] Add a new selection mode for the runtime system selector in [internal/space/ui/input.go](../../internal/space/ui/input.go)
- [x] Extend `InputState` with selector-specific fields that do not interfere with object-selection behavior
- [x] Add helper methods for:
	- opening the selector with a discovered file list
	- cancelling the selector cleanly
	- confirming the selected system path
	- reporting whether a confirmed selection requires reload
- [x] Add an app helper in [internal/space/app/](../../internal/space/app) to enumerate `.json` files under [data/systems/](../../data/systems)
- [x] Normalize selector entries so comparisons use stable paths relative to the app’s active `SystemConfig`
- [x] Store selector entries as label/path pairs to avoid brittle string matching

Validation after Phase 1:

- [x] Focused unit tests for selector state transitions in [internal/space/ui/input_test.go](../../internal/space/ui/input_test.go)
- [x] Focused unit test for system-file discovery filtering and stable ordering if discovery is factored for testability

### 4.2 Phase 2 - Modal Rendering and Input

Objective: make the selector accessible and usable as a true modal dialog.

- [x] Add `Cmd+S` handling in [internal/space/app/input.go](../../internal/space/app/input.go)
- [x] Keep existing modal ownership rules intact so help, performance, object selection, and system selector cannot overlap
- [x] Add selector navigation controls:
	- `Up` and `Down` move selection
	- `Home`, `End`, `PageUp`, `PageDown` accelerate list movement if needed
	- `Enter` confirms
	- `Escape` cancels
- [x] Draw a dedicated selector dialog in [internal/space/raylib/ui/render/renders.go](../../internal/space/raylib/ui/render/renders.go)
- [x] Show the currently loaded system clearly in the dialog
- [x] Show a compact note when the highlighted entry is already active so `Enter` behavior is obvious
- [x] Ensure main-window controls remain suspended while the selector is open

Validation after Phase 2:

- [x] Re-run focused UI input tests in [internal/space/ui/input_test.go](../../internal/space/ui/input_test.go)
- [x] Add or adjust tests for the new selector mode values if enum ordering is intentionally locked

### 4.3 Phase 3 - Session Reload Flow

Objective: replace the interactive runtime session safely when a new system is confirmed.

- [x] Add an app-level helper that creates a replacement runtime session from a supplied system path without duplicating startup logic
- [x] Update [internal/space/app/session.go](../../internal/space/app/session.go) so session creation can target a supplied system config path cleanly
- [x] Introduce a reload seam in [internal/space/app/interactive.go](../../internal/space/app/interactive.go) that:
	- stops the current simulation
	- cancels its simulation goroutine context
	- swaps in a fresh runtime session
	- starts the new simulation goroutine
	- resumes the loop without closing the app window
- [x] Keep runtime state behavior explicit during reload:
	- carry forward window/render settings owned by `RuntimeContext`
	- reset camera/input/session-specific state from the new session unless a narrower preservation rule is explicitly approved
- [x] When the selected system matches the current one, close the selector with no reload
- [x] Surface reload failures inside the selector or HUD in a way that does not leave the app in a half-reloaded state

Validation after Phase 3:

- [x] Add focused tests for reload decision logic if factored into a small helper
- [x] Run targeted package tests for [internal/space/app](../../internal/space/app) and [internal/space/ui](../../internal/space/ui)
- [ ] Perform manual runtime verification for:
	- open selector
	- cancel selector
	- confirm current system and observe no-op close
	- switch from solar system to alpha centauri and back
	- verify dialogs still suspend main controls during selector use

### 4.4 Phase 4 - Validation and Docs

Objective: finalize the work after implementation is proven.

- [x] Update [docs/standards/agent-readme.md](../standards/agent-readme.md) with the implemented selector binding and modal behavior
- [x] Update [docs/wip/todo.md](todo.md) to reflect in-progress implementation and manual verification status
- [ ] Move completed work into [docs/history/changelog.md](../history/changelog.md) once implementation and manual verification are complete
- [ ] Update help text/HUD hints if the implementation introduces selector-specific user guidance
- [ ] Confirm there are no stale references to the pre-implementation plan once the work is finished

## 5. Validation Plan

- Narrow first:
	- `go test ./internal/space/ui`
	- `go test ./internal/space/app`
- Then combined targeted pass:
	- `go test ./internal/space/app ./internal/space/ui`
- Manual runtime verification after code changes because the selector changes live interaction flow and session lifetime

## 6. Review Questions

1. Should the selector preserve any camera state across reload, or should every reload reset to the new session defaults?
2. Should selector entries display just the filename or a friendlier derived label plus filename?
3. Is `Cmd+S` intended as macOS-only behavior, or do we want a second non-macOS shortcut later?
4. If reload fails, should the UI keep the selector open with an inline error, or close it and surface the error elsewhere?