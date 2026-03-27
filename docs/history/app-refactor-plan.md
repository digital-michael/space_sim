# `main.go` Refactor Plan (Historical)

## Purpose
Preserve the original refactor rationale that led to the current thin-bootstrap application layout. This document is historical context only and should not be treated as the current architectural source of truth.

## Last Updated
2026-03-26

## Table of Contents
1. Historical Status
2. Goal
3. Current Problems in `main.go`
4. Target Architecture
5. Target Responsibilities
  5.1 `cmd/space-sim/main.go`
  5.2 `internal/space/app`
  5.3 `internal/space/raylib/ui/render`
  5.4 `internal/space/raylib/spatial`
  5.5 `internal/space/app/performance.go`
6. Required Refactor Actions
7. Configuration Strategy
8. Compliance Assessment
9. Migration Order
10. Definition of Done
11. Immediate Next Step

## Historical Status

This plan is retained because its architectural intent still explains the present layout, especially the thin bootstrap in [cmd/space-sim/main.go](../../cmd/space-sim/main.go), orchestration in [internal/space/app/](../../internal/space/app), centralized runtime state in [internal/space/app/runtime_context.go](../../internal/space/app/runtime_context.go), and Raylib-specific rendering under [internal/space/raylib/ui/render/](../../internal/space/raylib/ui/render).

For the current repository view, use [docs/standards/agent-readme.md](../standards/agent-readme.md) and [internal/space/package.md](../../internal/space/package.md) instead.

## Goal

Reduce [`cmd/space-sim/main.go`](../../cmd/space-sim/main.go) to a thin bootstrap:

1. load configuration
2. validate configuration
3. construct an `app.App`
4. execute `app.Run()`

This aligns with:

- **SOLID**
  - **S**ingle Responsibility: `main.go` should bootstrap only
  - **O**pen/Closed: rendering backends and runtime services should be swappable
  - **L**iskov: runtime services should be replaceable behind stable contracts
  - **I**nterface Segregation: app/runtime/rendering concerns split cleanly
  - **D**ependency Inversion: `App` depends on abstractions/services, not monolithic globals

- **DRY**
  - eliminate repeated layout constants, repeated category/tab construction, repeated per-frame state plumbing
  - centralize runtime state in a `RuntimeContext`

- **GRASP**
  - **Controller**: `app.App`
  - **Information Expert**: rendering code in renderer package, culling in spatial package, camera/input in ui package
  - **High Cohesion / Low Coupling**: split bootstrap, app orchestration, rendering, spatial, config, runtime state

---

## Current Problems in `main.go`

[`cmd/space-sim/main.go`](../../cmd/space-sim/main.go) currently owns too many responsibilities:

### 1. Bootstrap concerns
- CLI parsing
- log setup
- config path selection
- window init
- camera init
- simulation init

### 2. Runtime orchestration
- main loop
- frame timing
- screen-size decisions
- simulation coordination
- dataset switching
- fullscreen toggle handling

### 3. UI state management
- help visibility
- HUD visibility
- label visibility
- mouse mode
- performance menu state
- selection state
- tracking state
- temporary banners and indicators

### 4. Rendering
- all Raylib draw functions
- label selection and draw policy
- HUD drawing
- selection dialog drawing
- performance dialog drawing
- help screen drawing
- zoom indicator drawing
- object rendering

### 5. Spatial/culling logic
- frustum culling
- spatial grid creation
- visible cell extraction

### 6. Performance test harness
- performance config parsing
- benchmark execution
- reporting
- CSV saving

### 7. Layout and window concerns
- hardcoded screen assumptions
- magic numbers
- fullscreen behavior
- dynamic resize behavior

This violates SRP and produces a poor seam for future rendering backends.

---

## Target Architecture

## Package Layout

```text
internal/space/
├── app/
│   ├── app.go
│   ├── config.go
│   ├── runtime_context.go
│   ├── lifecycle.go
│   ├── loop.go
│   ├── input.go
│   ├── navigation.go
│   ├── performance.go
│   └── validation.go
│
├── engine/
│   └── ...existing engine package...
│
├── ui/
│   └── ...existing generic ui package...
│
├── raylib/
│   ├── spatial/
│   │   ├── grid.go
│   │   ├── frustum.go
│   │   └── visibility.go
│   └── ui/
│       └── render/
│           ├── renderer.go
│           ├── scene.go
│           ├── labels.go
│           ├── hud.go
│           ├── dialogs.go
│           ├── help.go
│           └── indicators.go
│
├── config.go
├── loader.go
├── objects.go
├── belts.go
└── simulation.go
```

---

## Target Responsibilities

## `cmd/space-sim/main.go`
Owns only:
- CLI parsing
- app config loading
- app config validation
- `app.New(...)`
- `app.Run()`
- process exit status

No rendering.
No per-frame logic.
No UI state.
No spatial logic.

---

## `internal/space/app`
Owns application orchestration.

### `App`
Suggested fields:

- configuration
- logger
- simulation
- camera state
- input state
- runtime context
- renderer
- spatial strategy
- performance harness options

Suggested methods:

- `New(cfg Config) (*App, error)`
- `Run(ctx context.Context) error`
- `runInteractive(ctx context.Context) error`
- `runPerformance(ctx context.Context) error`
- `initWindow() error`
- `initSimulation() error`
- `initRuntime() error`
- `shutdown() error`

### `RuntimeContext`
Purpose: replace the growing list of pass-through runtime parameters.

Suggested responsibilities:
- current and configured screen size
- last known screen size
- fullscreen flag
- HUD/help/labels visibility
- mouse mode state
- current dataset level
- camera movement settings
- mouse sensitivity
- grid visibility
- timing/debug toggles
- temporary banner/indicator state

Suggested API:
- `NewRuntimeContext(cfg AppConfig) *RuntimeContext`
- `UpdateScreenSize(width, height int32)`
- `EffectiveScreenSize() (int32, int32)`
- `ApplyWindowState()`
- `Snapshot()` if render code needs a read-only view later

This is the correct place to prepare for:
- fullscreen support
- dynamic resize
- removal of screen-size magic numbers

---

## `internal/space/raylib/ui/render`
Owns all Raylib-specific drawing.

### Move from `main.go`
- `drawObject`
- `drawObjectsInstanced`
- `drawGroundPlane`
- `drawObjectLabels`
- `selectObjectsForLabels`
- `drawHUD`
- `drawTrackingInfo`
- `drawSelectionUI`
- `drawPerformanceUI`
- `drawHelpScreen`
- `drawZoomIndicator`

### Renderer shape
Suggested type:

```go
type Renderer struct {
    // cached resources, colors, font/layout helpers later
}
```

Suggested methods:

- `RenderFrame(...)`
- `RenderScene(...)`
- `RenderHUD(...)`
- `RenderDialogs(...)`
- `RenderHelp(...)`
- `RenderIndicators(...)`

Important:
- keep this package **Raylib-specific**
- keep layout math close to draw functions
- make screen size come from `RuntimeContext`, never literals

This creates a seam for future renderer backends.

---

## `internal/space/raylib/spatial`
Owns spatial partition and frustum/culling strategies.

### Move from `main.go`
- `SpatialGrid`
- `buildGrid`
- `getCellsInFrustum`
- `frustumCullObjects`
- related helpers/constants that are algorithmic rather than UI-specific

### Notes
This package may hold multiple strategies:
- flat frustum test
- spatial grid
- future quadtree/octree if needed

This matches the stated goal of "one of several solutions."

---

## `internal/space/app/performance.go`
Keep performance harness inside `app`.

### Move from `main.go`
- `PerformanceTestConfig`
- `PerformanceResult`
- `runPerformanceTest`
- `runSingleTest`
- `printPerformanceResults`
- `savePerformanceResults`
- memory metric helpers

### Reason
This is an application mode, not bootstrap logic.

---

## Required Refactor Actions

## Phase 1 — Create app skeleton
1. add `internal/space/app`
2. define `App`
3. define `RuntimeContext`
4. move app-level config loading/validation here
5. make `main.go` call `app.New(...).Run(...)`

### Acceptance
- `main.go` contains no render loop
- `main.go` has no Raylib draw code
- `main.go` has no spatial code

---

## Phase 2 — Move runtime state out of `main.go`
1. identify all non-engine runtime booleans/counters/settings
2. move them into `RuntimeContext`
3. replace long parameter lists with `*RuntimeContext`
4. add `NewRuntimeContext`

### Initial `RuntimeContext` candidates
- `hudVisible`
- `helpVisible`
- `labelsVisible`
- `gridVisible`
- `mouseModeEnabled`
- `cameraSpeed`
- `mouseSensitivity`
- `asteroidDataset`
- fullscreen state
- configured/default/last screen size
- any transient banner state

### Acceptance
- most helper functions take `ctx *app.RuntimeContext` instead of many discrete args

---

## Phase 3 — Extract renderer package
1. create `internal/space/raylib/ui/render`
2. move all `draw*` functions there
3. create a `Renderer` type
4. group functions by responsibility:
   - scene
   - labels
   - HUD
   - dialogs
   - help
   - indicators
5. convert layout calculations to use runtime screen size

### Acceptance
- `main.go`/`app` only calls renderer entrypoints
- all Raylib drawing code is outside `main.go`

---

## Phase 4 — Extract spatial package
1. create `internal/space/raylib/spatial`
2. move spatial grid and frustum culling there
3. expose strategy-oriented APIs
4. keep App free of culling implementation details

### Acceptance
- app code selects a strategy, does not implement one

---

## Phase 5 — Move input/orchestration logic into App
1. move frame input handling into `app/input.go`
2. move selection/navigation flow into `app/navigation.go`
3. move frame lifecycle into `app/loop.go`
4. keep camera math in existing `ui` package
5. keep engine simulation in existing `engine` package

### Acceptance
- app owns interaction rules
- ui package remains generic state/behavior only

---

## Phase 6 — Move performance harness into App
1. move perf structs and helpers into `app/performance.go`
2. expose `RunPerformance(...)`
3. keep `Run()` choosing interactive vs performance mode

### Acceptance
- no benchmark/report code remains in `main.go`

---

## Phase 7 — Window/config normalization
This phase supports later fullscreen and resize work without losing the existing TODO tracking.

1. define `AppConfig`
2. include:
   - default screen size
   - configured screen size
   - last screen size
   - fullscreen
   - resizable
3. define screen-size precedence:
   1. last screen size
   2. configured screen size
   3. logical defaults
4. save config on shutdown
5. initialize `RuntimeContext` from config
6. route all screen queries through runtime context

### Acceptance
- no code reads a hardcoded 1280x720 as authoritative runtime state
- future fullscreen/dynamic-resize work becomes localized

---

## Phase 8 — Magic number removal support
Do **not** drop the existing TODO work. Instead, make it easier to complete.

1. move layout code into renderer package first
2. create shared layout helpers there
3. convert hardcoded panel geometry into named layout values
4. make layout depend on runtime screen size

### Important
This phase should reference existing:
- fullscreen support TODO
- magic number audit TODO

It should not replace them.

---

## Configuration Strategy

## AppConfig
Path of least resistance:

1. introduce `AppConfig` early
2. keep it small initially
3. wire it into `RuntimeContext`
4. expand it later for fullscreen/dynamic resize

Suggested fields:

```go
type AppConfig struct {
    Window struct {
        Width      int32 `json:"width"`
        Height     int32 `json:"height"`
        Fullscreen bool  `json:"fullscreen"`
        Resizable  bool  `json:"resizable"`
    } `json:"window"`
}
```

Later additions:
- last width/height
- window position
- vsync/target fps
- preferred renderer backend

---

## Compliance Assessment

## SRP status after plan
- `main.go`: bootstrap only
- `app`: orchestration only
- `engine`: simulation only
- `ui`: generic camera/input only
- `raylib/ui/render`: Raylib rendering only
- `raylib/spatial`: culling/spatial only

## DRY improvements
- single runtime state source
- single screen-size resolution source
- single renderer entrypoint
- single performance harness location
- single layout helper location

## GRASP improvements
- `App` becomes Controller
- renderer becomes Information Expert for Raylib drawing
- spatial package becomes Information Expert for culling
- runtime context becomes Information Expert for mutable app runtime state

---

## Migration Order

Recommended order:

1. create `app` package and `App` shell
2. create `RuntimeContext`
3. move render loop into `app.Run()`
4. move draw functions into `raylib/ui/render`
5. move spatial code into `raylib/spatial`
6. move performance harness into `app`
7. wire `AppConfig`
8. address magic numbers/fullscreen in renderer/runtime context

This minimizes breakage while steadily shrinking `main.go`.

---

## Definition of Done

`main.go` is considered compliant when:

- it only bootstraps configuration and starts the app
- it contains no draw functions
- it contains no spatial logic
- it contains no per-frame runtime logic
- it contains no performance harness logic
- runtime state is centralized in `RuntimeContext`
- screen size/fullscreen state is owned by config + runtime context
- magic-number cleanup can be completed inside renderer code without touching bootstrap

---

## Immediate Next Step

Implement:
1. `internal/space/app/app.go`
2. `internal/space/app/runtime_context.go`
3. move the render loop from `main.go` into `App.Run()`
4. reduce `main.go` to bootstrap only

That is the smallest high-value first slice.