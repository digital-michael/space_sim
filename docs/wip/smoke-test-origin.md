# Raylib Smoke Test: Solar System Simulator

**Purpose**: Validate RaylibSim architecture with visual Raylib demo  
**Theme**: Solar system with planets, asteroids, satellites  
**Status**: 🚧 In Progress  
**Started**: 2026-02-12  

## Objectives

### Architecture Validation
- ✅ Thread safety: Simulation goroutine + render on main thread
- ✅ State snapshot: Double-buffered lock-free reads
- ✅ Performance: 60 FPS with 20+ objects
- ✅ Object management: Named objects, lookup by name
- ✅ Camera system: Free-fly, jump, track modes

### Raylib Feature Showcase
- ✅ 3D rendering: Spheres, basic materials
- ✅ Lighting: Point light (sun)
- ✅ Materials: Diffuse (planets), emissive (sun), mirror (satellites)
- ✅ Text: 3D labels + 2D HUD
- ✅ Input: Keyboard + mouse controls
- ✅ Camera: First-person + tracking modes

## Implementation Plan

### Milestone 1: Foundation ✅ (Target: Session 1)
**Goal**: Single planet orbiting sun with camera controls

- [ ] **1.1 Project Structure**
  - [ ] Create `cmd/raylib-smoke/main.go`
  - [ ] Create `internal/smoke/` package
  - [ ] Add raylib-go dependency

- [ ] **1.2 Basic Types**
  - [ ] `Vector3` (position, velocity)
  - [ ] `Object` struct (name, position, radius, color, material)
  - [ ] `SimulationState` (all objects, timestamp)
  - [ ] `DoubleBuffer` (front/back state buffers)

- [ ] **1.3 Simulation Goroutine**
  - [ ] Orbital mechanics (simple circular orbits)
  - [ ] Fixed timestep loop (60 Hz)
  - [ ] State buffer swap
  - [ ] Sun (stationary) + 1 planet (orbiting)

- [ ] **1.4 Renderer (Main Thread)**
  - [ ] Raylib initialization (1280×720 window)
  - [ ] Read from front buffer (lock-free)
  - [ ] Draw spheres (sun + planet)
  - [ ] Basic camera (fixed position for now)
  - [ ] FPS counter

- [ ] **1.5 Camera Controls**
  - [ ] Free-fly mode (WASD + mouse look)
  - [ ] Smooth movement
  - [ ] Mouse sensitivity adjustment

**Success Criteria**: 
- Planet orbits sun smoothly
- Camera navigable with WASD + mouse
- Stable 60 FPS
- No race conditions (run with `-race` flag)

---

### Milestone 2: Object System ✅ (Target: Session 1-2)
**Goal**: Multiple objects with naming and navigation

- [ ] **2.1 Object Factory**
  - [ ] `NewSun()` - emissive yellow, radius 2.0
  - [ ] `NewPlanet(name, distance, speed, radius, color)` - configurable
  - [ ] `NewAsteroid(name, distance, speed)` - small, random positions
  - [ ] `NewSatellite(name, parent, distance, speed)` - mirror material

- [ ] **2.2 Object Management**
  - [ ] Add 3-4 planets (Mercury, Venus, Earth, Mars analogs)
  - [ ] Add 10-15 asteroids (belt between Mars/Jupiter)
  - [ ] Add 2 satellites (orbit Earth, Mars)
  - [ ] Object lookup by name (map[string]*Object)

- [ ] **2.3 Jump Feature**
  - [ ] Press J → show object list (text overlay)
  - [ ] Arrow keys to select
  - [ ] Enter → camera flies to object (animated)
  - [ ] Return to free-fly mode after animation

- [ ] **2.4 Track Feature**
  - [ ] Press T → show object list
  - [ ] Enter → camera locks onto object
  - [ ] Camera follows object continuously
  - [ ] Mouse wheel: adjust distance
  - [ ] Mouse drag: orbit around object
  - [ ] Press F or ESC: return to free-fly

**Success Criteria**:
- 20+ objects moving smoothly
- Jump navigation works (smooth animation)
- Track mode follows objects correctly
- Still 60 FPS with all objects

---

### Milestone 3: Visual Polish ✅ (Target: Session 2-3)
**Goal**: Materials, lighting, text labels

- [ ] **3.1 Materials**
  - [ ] Emissive sun (glowing yellow)
  - [ ] Diffuse planets (matte colors)
  - [ ] Metallic asteroids (gray, slight shine)
  - [ ] Mirror satellites (reflective, ray tracing test)

- [ ] **3.2 Lighting**
  - [ ] Point light at sun position
  - [ ] Proper shading on planets/asteroids
  - [ ] Falloff with distance

- [ ] **3.3 Environment**
  - [ ] Starfield background (skybox or procedural)
  - [ ] Ground plane with grid (spatial reference)
  - [ ] Depth fog (optional)

- [ ] **3.4 Text & UI**
  - [ ] 3D text labels above objects (names)
  - [ ] 2D HUD (FPS, camera mode, tracked object)
  - [ ] Selection highlight (outline selected object)

- [ ] **3.5 Polish**
  - [ ] Smooth camera transitions
  - [ ] Object selection feedback
  - [ ] Help overlay (H key shows controls)

**Success Criteria**:
- Mirror satellites show reflections
- Lighting looks realistic
- Text labels readable
- UI responsive

---

### Milestone 4: Lessons Learned ✅ (Target: After Milestone 3)
**Goal**: Document findings for main implementation

- [ ] **4.1 Performance Analysis**
  - [ ] Profile rendering (frame time breakdown)
  - [ ] Profile simulation (update time)
  - [ ] Identify bottlenecks

- [ ] **4.2 Architecture Validation**
  - [ ] Thread model works? (any issues?)
  - [ ] State snapshot efficient? (copy overhead?)
  - [ ] Object lookup fast enough? (name → object)

- [ ] **4.3 Raylib Findings**
  - [ ] Ray tracing performance (mirrors)
  - [ ] Material system usage
  - [ ] Text rendering approach
  - [ ] Input handling patterns

- [ ] **4.4 Documentation**
  - [ ] Update phase plans based on findings
  - [ ] Note any architecture adjustments needed
  - [ ] List recommended Raylib patterns

**Deliverable**: 
- `raylib-lessons-learned.md` (Raylib-specific)
- `architecture-lessons-learned.md` (Framework-specific)

---

## Code Structure

```
cmd/raylib-smoke/
  └── main.go                 # Entry point, Raylib init, main loop

internal/smoke/
  ├── state.go               # SimulationState, DoubleBuffer, Object
  ├── simulation.go          # Simulation goroutine, orbital mechanics
  ├── renderer.go            # Rendering logic (Raylib calls)
  ├── camera.go              # Camera modes (free-fly, jump, track)
  ├── objects.go             # Object factory functions
  ├── input.go               # Input handling (keyboard, mouse)
  └── ui.go                  # Text rendering, HUD, menus
```

## Design Principles

### SOLID
- **Single Responsibility**: Each file has clear purpose (camera, renderer, simulation)
- **Open/Closed**: Object factory extensible (add new object types easily)
- **Liskov Substitution**: Camera modes implement common interface
- **Interface Segregation**: Small focused interfaces (Updater, Renderer, etc.)
- **Dependency Inversion**: Depend on interfaces, not concrete types

### GRASP
- **Information Expert**: Object knows its own properties (position, radius)
- **Creator**: Factory functions create related objects
- **Controller**: Simulation goroutine controls state updates
- **Low Coupling**: Renderer only depends on state snapshot (not simulation logic)
- **High Cohesion**: Related functionality grouped (all camera logic in camera.go)

### DRY
- Shared vector math functions (don't duplicate)
- Common rendering patterns (sphere drawing helper)
- Reusable camera transition logic

### Go Best Practices
- Error handling: Check all errors, fail gracefully
- Concurrency: Use channels for goroutine communication
- Locks: Minimize lock scope (double-buffer avoids locks in render path)
- Naming: Clear, concise names (SimulationState not SS)
- Comments: Explain *why*, not *what*
- Testing: Test orbital mechanics, buffer swaps, camera math

## Controls Reference

| Key | Action |
|-----|--------|
| **WASD** | Move camera (free-fly mode) |
| **Mouse** | Look around (free-fly mode) |
| **J** | Jump to object (show list) |
| **T** | Track object (continuous follow) |
| **F** | Free-fly mode (cancel track) |
| **ESC** | Cancel menu / free-fly mode |
| **H** | Show/hide help overlay |
| **1-9** | Quick-jump to first 9 objects |
| **Mouse Wheel** | Adjust distance (track mode) |
| **Mouse Drag** | Orbit object (track mode) |
| **Arrow Keys** | Navigate menus |
| **Enter** | Confirm selection |

## Performance Targets

| Metric | Target | Critical |
|--------|--------|----------|
| Frame rate | 60 FPS | 45 FPS |
| Frame time | <16.7ms | <22ms |
| Simulation step | <5ms | <10ms |
| Object count | 20+ | 10+ |
| State copy | <1ms | <5ms |

## Dependencies

```bash
go get github.com/gen2brain/raylib-go/raylib
```

**Version**: Latest stable (5.0+)

## Build & Run

```bash
# Build
go build -o bin/raylib-smoke cmd/raylib-smoke/main.go

# Run
./bin/raylib-smoke

# Run with race detector
go run -race cmd/raylib-smoke/main.go
```

## Known Constraints

- **Raylib main thread**: All Raylib calls must be on main thread (macOS requirement)
- **State copying**: Snapshot creates copy (acceptable for 20 objects, may need optimization for 1000+)
- **Simplified physics**: Circular orbits only (not elliptical/Newtonian)
- **No collision**: Objects pass through each other
- **Fixed star field**: No parallax scrolling

## Future Enhancements (Post-Smoke Test)

- Shape morphing (sphere → cube transition)
- Particle trails
- HDR skybox
- Shadow mapping
- Bloom effect on sun
- Group selection (select all asteroids)
- Time controls (pause, speed up, slow down)
- Export camera path for replay

---

**Next Steps**: Begin Milestone 1.1 (Project Structure)

**Status Legend**:
- ✅ Complete
- 🚧 In Progress
- ⏸️ Blocked
- ❌ Failed
- 📋 Not Started
