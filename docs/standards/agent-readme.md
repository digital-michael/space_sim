# Agent README

## Purpose
Provide a high-signal, agent-oriented map of this repository so an LLM Agent can understand the architecture, runtime behavior, data model, validation surface, and operational constraints before making changes.

## Last Updated
2026-03-28

## Table of Contents
1. Mission and Defaults
2. Repository Map
	2.1 Runtime and Application
	2.2 Core Domain and Simulation
	2.3 Configuration and Data
	2.4 Quality and Ops
	2.5 Package Doc Index
3. High-Level Component Architecture
	3.1 Layered View
	3.2 Important Architectural Boundaries
	3.3 Preserved Refactor Intent
4. Core Process Flows
	4.1 Startup Flow
	4.2 JSON Loading Flow
	4.3 Simulation Update Flow
	4.4 Dataset Switching Flow
	4.5 Interactive UI Flow
	4.6 Performance Mode Flow
5. Data Structures and Contracts
	5.1 Core Runtime Model
	5.2 Key Enums and Stable Concepts
	5.3 JSON Contract Shape
	5.4 Data Patterns in Use
6. User Interface Command Groups
	6.1 App and Display Controls
	6.2 Simulation Time and Dataset Controls
	6.3 Selection and Performance Controls
	6.4 Camera and Navigation Controls
	6.5 Performance Menu Concepts
7. Testing and Validation Surface
	7.1 Go Unit Tests Present Today
	7.2 Command-Level Validation
	7.3 Practical Testing Guidance for Agents
8. Documentation Surface
9. Current Gaps and Cautions
10. Agent Working Defaults

## 1. Mission and Defaults

Space Sim is a real-time, Go-based solar system simulator built on Raylib. The repository centers on four concerns:

1. JSON-driven solar system definition and loading.
2. Real-time orbital simulation using a double-buffered engine.
3. Interactive camera/navigation UI plus performance experimentation.
4. Documentation of architectural and implementation lessons.

Default agent assumptions:

- JSON is the source of truth for solar-system content.
- The simulation engine and UI state are intentionally separated from Raylib types when possible.
- Performance work must preserve clarity and architecture.
- Existing docs in `docs/` and lessons learned files should be treated as guidance when code intent is missing.

## 2. Repository Map

### Runtime and Application

- [cmd/space-sim/](../../cmd/space-sim): app entrypoint and CLI flag parsing.
- [internal/space/app/](../../internal/space/app): runtime orchestration, windowing, session setup, input handling, performance mode, fullscreen, debug support.
- [internal/space/raylib/ui/render/](../../internal/space/raylib/ui/render): Raylib-backed rendering and help screen drawing.

### Core Domain and Simulation

- [internal/space/engine/](../../internal/space/engine): pure simulation kernel, math, object model, double buffer, physics update loop, feature configs, constants.
- [internal/space/ui/](../../internal/space/ui): generic camera and selection state with no Raylib dependency.
- [internal/space/](../../internal/space): SOL-specific loading, belt generation, JSON schema structs, object constructors, and simulation wrapper.

### Configuration and Data

- [configs/app.json](../../configs/app.json): window and render mode defaults.
- [data/systems/](../../data/systems): top-level system JSON files.
- [data/bodies/](../../data/bodies): reusable body template libraries.
- [data/assets/](../../data/assets): texture manifest and texture assets.

### Quality and Ops

- [scripts/](../../scripts): validation, maintenance, asset setup, and historical helper scripts.
- [docs/](../): architectural notes, migration history, lessons learned, performance analysis, and standards.
- [Makefile](../../Makefile): primary developer commands.

### Package Doc Index

Use these package docs as the fastest package-level orientation pass before drilling into source files.

- [cmd/space-sim/doc.go](../../cmd/space-sim/doc.go): interactive application entrypoint and CLI bootstrap.
- [internal/space/doc.go](../../internal/space/doc.go): application-layer JSON loading, object construction, belt generation, and simulation wrapper.
- [internal/space/app/doc.go](../../internal/space/app/doc.go): app runtime orchestration, window/session lifecycle, and execution modes.
- [internal/space/engine/doc.go](../../internal/space/engine/doc.go): renderer-agnostic simulation kernel, object model, double buffer, and physics loop.
- [internal/space/ui/doc.go](../../internal/space/ui/doc.go): generic camera, selection, and performance option state.
- [internal/space/raylib/spatial/doc.go](../../internal/space/raylib/spatial/doc.go): Raylib-backed frustum culling and spatial partitioning helpers.
- [internal/space/raylib/ui/render/doc.go](../../internal/space/raylib/ui/render/doc.go): Raylib rendering implementation for scene and overlay drawing.

## 3. High-Level Component Architecture

### Layered View

1. CLI/bootstrap layer.
	 - [cmd/space-sim/main.go](../../cmd/space-sim/main.go) parses flags, loads app config, builds `app.Config`, and starts the application with signal-aware shutdown.
2. Application orchestration layer.
	 - `internal/space/app` owns window lifecycle, runtime session creation, input dispatch, interactive loop, and performance mode execution.
3. Domain/runtime layer.
	 - `internal/space` loads JSON, translates system definitions into engine objects, and manages SOL-specific asteroid and Kuiper belt dataset behavior.
4. Engine layer.
	 - `internal/space/engine` owns immutable object metadata, mutable animation state, double-buffer synchronization, and physics updates.
5. UI state layer.
	 - `internal/space/ui` models camera state, selection state, and performance option state without renderer coupling.
6. Renderer layer.
	 - Raylib integration consumes the front buffer and UI state to draw the scene and overlays.

### Important Architectural Boundaries

- `internal/space/engine` should remain stdlib-only and renderer-agnostic.
- `internal/space/ui` should remain generic UI state, not a place for Raylib calls.
- `internal/space` is where JSON loading and SOL-specific translation logic belong.
- `internal/space/app` is allowed to orchestrate, but should not absorb engine or schema responsibilities.

### Preserved Refactor Intent

The current layout preserves the useful intent of the completed `main.go` refactor work without treating the old plan as an active source of truth:

- Keep [cmd/space-sim/main.go](../../cmd/space-sim/main.go) as a thin bootstrap that parses flags, loads config, constructs the app, and runs it.
- Keep application orchestration and application modes in [internal/space/app/](../../internal/space/app), including interactive and performance flows.
- Keep mutable runtime and window/UI state centralized in [internal/space/app/runtime_context.go](../../internal/space/app/runtime_context.go) rather than spreading pass-through state across long parameter lists.
- Keep Raylib-specific drawing isolated under [internal/space/raylib/ui/render/](../../internal/space/raylib/ui/render) so rendering concerns stay separate from bootstrap and engine logic.
- Preserve the responsibility split as historical rationale for the current architecture, not as a requirement to recreate every file named in the original refactor plan.

## 4. Core Process Flows

### Startup Flow

1. Load app config from [configs/app.json](../../configs/app.json).
2. Parse CLI flags such as `--performance`, `--profile`, `--threads`, `--no-locking`, `--system-config`, and `--debug`.
3. Construct `app.Config` and validate it.
4. Create the app runtime context and renderer.
5. Initialize the window and runtime session.
6. Load the selected JSON system via `internal/space.LoadSystemFromFile`.
7. Start the simulation goroutine with signal-aware cancellation.
8. Enter either interactive mode or performance mode.

### JSON Loading Flow

1. Read a system file from [data/systems/](../../data/systems).
2. Unmarshal into `SystemConfig`.
3. Load templates from [data/bodies/](../../data/bodies) when configured.
4. Create bodies from body configs, applying template overrides.
5. Translate feature configs into runtime belt/ring structures.
6. Populate `engine.SimulationState`.
7. Build navigation order from categories that are actually present.

### Simulation Update Flow

1. `engine.Simulation.Start(ctx)` runs at the configured simulation Hz.
2. Time scale and dataset changes are applied.
3. Physics updates compute new object positions from orbital parameters.
4. Back buffer state is updated.
5. Buffers are swapped so rendering can read a stable front buffer.

### Dataset Switching Flow

1. Input requests a new asteroid dataset.
2. The SOL wrapper checks whether asteroid/Kuiper objects for that level already exist.
3. Missing datasets are lazily allocated into back and front buffers using deterministic seeds.
4. In-place swap is temporarily disabled during object-slice length mutation.
5. Visibility flags are updated so objects up to the selected dataset are visible.

### Interactive UI Flow

1. App reads current front-buffer state.
2. Input handler updates HUD/help/mouse/grid/label flags, time controls, dataset level, and camera/selection state.
3. Camera state is updated according to mode: free, jumping, or tracking.
4. Renderer draws visible objects and overlays using performance options and thresholds.

### Performance Mode Flow

1. CLI enables `--performance`.
2. App applies optional profile, threads, and locking overrides.
3. Warmup and measurement loops run against configured datasets/options.
4. Results are logged and summarized for performance analysis.

## 5. Data Structures and Contracts

### Core Runtime Model

- `engine.ObjectMetadata`: immutable physical, orbital, hierarchy, and rendering-adjacent properties.
- `engine.AnimationState`: mutable per-frame position/orbit data.
- `engine.Object`: metadata + animation state + `Visible` flag + dataset membership.
- `engine.SimulationState`: collection of objects plus time scale, dataset tracking, and feature config references.
- `engine.DoubleBuffer`: front/back state container for concurrent simulation and rendering.

### Key Enums and Stable Concepts

- `ObjectCategory`: `Planet`, `DwarfPlanet`, `Moon`, `Asteroid`, `Ring`, `Star`, `Belt`.
- `AsteroidDataset`: `Small`, `Medium`, `Large`, `Huge`.
- `SelectionMode`: `None`, `Jump`, `Track`, `TrackEquatorial`, `Performance`.
- `CameraMode`: free navigation, jump transition, and tracking.

### JSON Contract Shape

Primary top-level JSON structures:

- `SystemConfig`
	- system metadata, scale factor, bodies, features, template source, and default runtime state.
- `BodyConfig`
	- `type`, `name`, optional `parent`, `template`, plus nested `orbit`, `physical`, and `rendering` sections.
- `FeatureConfig`
	- procedural feature definitions such as asteroid belts and Kuiper belts.
- `TemplateLibrary`
	- named reusable body and feature templates loaded from `data/bodies/*.json`.

### Data Patterns in Use

- Orbital fields are modeled explicitly, not inferred from rendering.
- Deterministic procedural generation uses explicit seeds.
- Parent-child relationships are name-based.
- Importance values drive performance culling and rendering prioritization.
- Dataset membership and visibility are runtime concerns, not static JSON duplication.

## 6. User Interface Command Groups

These groups are derived from [internal/space/app/input.go](../../internal/space/app/input.go) and are the fastest way for an agent to orient itself to user-facing behavior.

### App and Display Controls

- `Ctrl+G`: toggle grid.
- `Ctrl+H`: toggle HUD.
- `Ctrl+L`: toggle labels.
- `Ctrl+/`: toggle help screen.
- `Ctrl+M`: toggle mouse-look mode and cursor capture.
- `Cmd+S`: open the runtime system selector in interactive mode.
- `Ctrl+F`: toggle fullscreen.
- `Ctrl+Q`: quit.

### Simulation Time and Dataset Controls

- `,` and `.`: decrease or increase simulation seconds-per-second.
- `Shift+,` and `Shift+.`: decrease or increase simulation tick speed.
- `+` and `-`: increase or decrease asteroid/Kuiper dataset level.

### Selection and Performance Controls

- `J`: open jump-to-object selection when in free camera mode.
- `T`: open the tracking selection dialog from free or tracking mode; confirmed selections start equatorial tracking.
- `Ctrl+P`: open performance options UI.
- Dialog invocation rule: unmodified letter keys open navigation-target dialogs, `Ctrl+...` is reserved for system/configuration actions and dialogs, and `Cmd+S` is the dedicated runtime system selector shortcut.
- `Ctrl`, `Alt`/`Option`, and `Cmd` modified variants of navigation keys are ignored so modified shortcuts do not fall through to navigation actions.
- While help, selection, or performance dialogs are open, main-window keyboard and mouse controls are suspended until the dialog closes.
- `Escape`: closes help first, then active selection/performance dialogs, then exits tracking, then exits mouse-look mode.
- In object selection UIs: `Left/Right` changes category, `Up/Down` navigates the current list, `PageUp/PageDown/Home/End` accelerate navigation, text input filters results, `Enter` confirms, and `Escape` cancels.
- In the runtime system selector UI: `Up/Down` navigates systems, `PageUp/PageDown/Home/End` accelerate navigation, `Enter` confirms, and `Escape` cancels.

### Camera and Navigation Controls

- `C`: center on origin in free mode, or reset tracking zoom in tracking mode.
- `F`: drill down to the closest visible child when tracking.
- `B`: move to the tracked object’s parent when tracking.
- `Tab` and `Shift+Tab`: cycle tracked siblings.
- `R`: reset tracking offset.
- `W`, `A`, `S`, `D`: move in free mode or pan/offset while tracking.
- Arrow keys: vertical and lateral movement in camera modes; selection navigation in modal UIs.
- Mouse move: look around or orbit tracked target when mouse mode is enabled.
- Mouse wheel: zoom or change tracking distance.

### Performance Menu Concepts

- Toggle frustum culling, LOD, instanced rendering, spatial partitioning, and point rendering.
- Configure importance threshold.
- Toggle in-place swap behavior.

## 7. Testing and Validation Surface

### Go Unit Tests Present Today

- [internal/space/belts_test.go](../../internal/space/belts_test.go)
	- validates orbital-period handling for procedurally generated belt objects.
- [internal/space/engine/objectcategory_test.go](../../internal/space/engine/objectcategory_test.go)
	- locks enum values and enum count.
- [internal/space/ui/input_test.go](../../internal/space/ui/input_test.go)
	- validates selection-state defaults, transitions, dialog input suspension state, and runtime system selector state behavior.
- [internal/space/ui/cycle_test.go](../../internal/space/ui/cycle_test.go)
	- validates category cycling behavior.
- [internal/space/app/system_selector_test.go](../../internal/space/app/system_selector_test.go)
	- validates runtime system discovery ordering/filtering and safe session-creation failure handling.

### Command-Level Validation

- `make test`: run Go unit tests.
- `make json-check`: validate default JSON system and confirm the app still builds.
- [scripts/test_json_system.sh](../../scripts/test_json_system.sh): JSON validation/build check script.

### Practical Testing Guidance for Agents

- Prefer targeted tests before broad test runs when changing a narrow area.
- If changing data-loading logic, validate both JSON parsing and runtime assumptions.
- If changing key bindings or selection flows, check the input and cycle tests and add coverage where missing.
- If changing concurrency or dataset switching, treat lessons learned docs as required context before refactoring.

## 8. Documentation Surface

High-value docs to consult first:

- [README.md](../../README.md): repository scope and common commands.
- [internal/space/package.md](../../internal/space/package.md): detailed package boundaries and architecture.
- [data/README.md](../../data/README.md): JSON layout and schema guidance.
- [docs/README.md](../README.md): documentation index and folder guide.
- [docs/history/lessons-learned.md](../history/lessons-learned.md): defect history and performance/debugging lessons.
- [docs/history/lessons-learned-double-buffering.md](../history/lessons-learned-double-buffering.md): double-buffer implementation lessons.
- [docs/performance/performance-analysis.md](../performance/performance-analysis.md) and [docs/performance/performance-results.md](../performance/performance-results.md): measured performance context.
- [docs/history/json-only-migration.md](../history/json-only-migration.md): the repository’s move away from hard-coded content.

Documentation convention already established in this repo:

- Substantive project Markdown belongs under `docs/`.
- Root-level Markdown should stay limited to top-level repository files such as [README.md](../../README.md).
- Non-README Markdown filenames should use lowercase kebab-case.

## 9. Current Gaps and Cautions

- The runtime system selector is implemented behind `Cmd+S` for interactive mode, but manual runtime verification is still required before it should be treated as complete work.
- `ring_system` features are now part of the supported loader path; keep rings defined through the feature pipeline rather than reintroducing duplicate body-based ring data.
- Clone-mode swapping now uses the engine object pool when in-place swap is disabled; preserve the double-buffer ownership rules documented in the lessons-learned docs when extending this path.
- Legacy scripts have been isolated under [scripts/legacy/](../../scripts/legacy) and should be treated as historical reference only.
- Performance-related changes can regress correctness or inter-thread safety if front/back buffer assumptions are not preserved.

## 10. Agent Working Defaults

When making changes, default to this workflow:

1. Read the relevant code, tests, docs, and lessons learned before editing.
2. Keep changes aligned with [docs/standards/coding-standards.md](coding-standards.md).
3. Prefer small, verified increments over broad rewrites.
4. Update docs when behavior, contracts, or operating procedures change.
5. Remove temporary work or promote it into a maintained script/doc location.
6. Treat architecture boundaries as intentional unless the user explicitly asks to redesign them.
