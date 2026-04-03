# Lessons Learned: Double Buffering & Thread Safety

## Purpose
Document the key concurrency, cloning, synchronization, and configuration lessons learned while implementing and debugging the double-buffer simulation architecture.

## Last Updated
2026-03-26

## Table of Contents
1. Go Pointer Semantics & Deep Copy Pitfalls
    1.1 Problem
    1.2 Root Cause
    1.3 Solution
2. Double Buffer Contract Violations
3. Cross-Thread Deadlocks
4. Visibility State Synchronization
5. Configuration Management & Single Source of Truth
6. Type Safety in Graphics Code
7. Architecture Principles
8. Summary

**Date**: February 18, 2026  
**Context**: Fixing asteroid stuttering, disappearing objects, and dataset switching bugs

---

## 1. Go Pointer Semantics & Deep Copy Pitfalls

### Problem
Initial `Clone()` implementation created shallow copies:
```go
// WRONG: Creates pointer to copy, but both buffers point to same objects
for i, obj := range s.Objects {
    objCopy := *obj
    cloned.Objects[i] = &objCopy  // All pointers reference same memory!
}
```

### Root Cause
- `objCopy := *obj` copies the struct by value to the stack
- `&objCopy` creates a pointer to that stack copy
- But the pointer is stored, and the stack location is reused in next iteration
- Result: All `cloned.Objects[i]` pointers end up pointing to the last object copied

### Solution
Explicitly allocate new objects with their own memory:
```go
// CORRECT: Each object gets its own allocation
for i, obj := range s.Objects {
    objCopy := *obj
    newObj := &Object{
        Meta:    objCopy.Meta,
        Anim:    objCopy.Anim,
        Visible: objCopy.Visible,
        Dataset: objCopy.Dataset,
    }
    cloned.Objects[i] = newObj  // Independent allocation
}
```

**Key Lesson**: In Go, `&structCopy` creates a pointer to a stack-allocated copy. For true deep copies in double buffering, explicitly allocate each struct with `&StructType{...}`.

---

## 2. Double Buffer Contract Violations

### Problem
`SetAsteroidDataset()` was modifying BOTH front and back buffers directly:
```go
// WRONG: Violates double buffering contract
func (s *Simulation) SetAsteroidDataset(dataset AsteroidDataset) {
    front := s.state.GetFrontUnsafe()  // Direct access to front buffer!
    back := s.state.GetBack()
    
    // Modifying both buffers breaks renderer/simulation isolation
    updateVisibility(front, dataset)
    updateVisibility(back, dataset)
}
```

### Root Cause
- Front buffer is being read by renderer (potentially with read lock held)
- Direct modification during rendering = race condition
- Violates the fundamental rule: **ONLY simulation modifies back buffer, ONLY renderer reads front buffer**

### Solution
1. **Removed `GetFrontUnsafe()` entirely** - prevents architectural violations
2. **Added critical warnings** to `GetBack()` and `Swap()` documentation
3. **Back-buffer-only modifications** - all changes go through back → Swap() cycle

```go
// CORRECT: Only modify back buffer
func (s *Simulation) applyAsteroidDatasetChange(dataset AsteroidDataset) {
    back := s.state.GetBack()  // Only access back buffer
    updateVisibility(back, dataset)
    // Changes become visible on next Swap()
}
```

**Key Lesson**: Double buffering requires strict discipline. Never modify the front buffer. Remove unsafe access methods to enforce the contract at the language level.

---

## 3. Cross-Thread Deadlocks

### Problem (Attempt 1: Mutex Lock)
```go
// WRONG: Deadlock when called from main thread
func (s *Simulation) SetAsteroidDataset(dataset AsteroidDataset) {
    s.updateLock.Lock()  // Main thread acquires lock
    defer s.updateLock.Unlock()
    
    // Modify back buffer
    s.state.Swap()  // Tries to acquire db.mu.Lock() - DEADLOCK
}
```

**Deadlock scenario**:
- Simulation goroutine: Has `updateLock` → calls `Swap()` → needs `db.mu`
- Main thread: Calls `SetAsteroidDataset()` → needs `updateLock` → blocks
- If renderer has `db.mu.RLock()`, `Swap()` blocks waiting for write lock
- **Classic deadlock**: Circular wait on locks across threads

### Problem (Attempt 2: Direct Swap)
```go
// WRONG: Double swapping
func (s *Simulation) applyAsteroidDatasetChange(dataset AsteroidDataset) {
    // Modify back buffer
    s.state.Swap()  // Swap 1
}

func (s *Simulation) update(dt float64) {
    // Update physics
    s.state.Swap()  // Swap 2
}
```

**Result**: Two swaps per frame → buffers flip-flop → visible stuttering (forward/backward motion)

### Solution: Command Queue Architecture
```go
type Simulation struct {
    datasetChangeCh chan AsteroidDataset  // Async command queue
}

// Non-blocking: just queue the request
func (s *Simulation) SetAsteroidDataset(dataset AsteroidDataset) {
    select {
    case s.datasetChangeCh <- dataset:
    default:  // Skip if queue full
    }
}

// Process in simulation goroutine
func (s *Simulation) Start(ctx context.Context) {
    for {
        select {
        case <-ticker.C:
            // Check for pending changes BEFORE physics update
            select {
            case dataset := <-s.datasetChangeCh:
                s.applyAsteroidDatasetChange(dataset)
            default:
            }
            s.update(dt)  // Single swap at end
        }
    }
}
```

**Key Lesson**: When rendering and simulation threads need to communicate, use **channels** (Go's CSP model), not shared locks. Commands from main thread → queued → processed in simulation thread → single swap per frame.

---

## 4. Visibility State Synchronization

### Problem
After swap, the old front buffer (now back buffer) had stale visibility flags:

**Frame N**:
- Back buffer: Asteroids for dataset 1 visible
- Front buffer: Asteroids for dataset 0 visible
- `Swap()` → Front shows dataset 1 ✓

**Frame N+1**:
- Back buffer: Now has old front buffer data (dataset 0 visibility!) ✗
- Physics update doesn't touch visibility
- `Swap()` → Front shows dataset 0 again = **flashing**

### Solution
Re-sync visibility after every swap:
```go
func (s *Simulation) update(dt float64) {
    // Physics update
    s.state.Swap()
    
    // Immediately sync visibility in new back buffer
    back = s.state.GetBack()
    for _, obj := range back.Objects {
        if obj is asteroid {
            obj.Visible = obj.Dataset <= back.CurrentDataset
        }
    }
}
```

**Key Lesson**: In double buffering, **derived state** (like visibility based on dataset) must be synchronized to BOTH buffers. After swap, the new back buffer may have stale derived state that needs updating.

---

## 5. Configuration Management & Single Source of Truth

### Problem
Rendering thresholds scattered across codebase:
- Point rendering: `200.0`, `100.0`, `500.0`, `300.0` hardcoded in multiple places
- LOD distances: `20.0`, `50.0`, `100.0`, `200.0` repeated
- Spatial grid: `50.0`, `1000.0`, `5000.0` magic numbers
- 30+ locations with hardcoded values

**Issues**:
- Inconsistent values across different code paths
- Impossible to tune without finding all locations
- No documentation of why specific values were chosen
- Copy-paste errors led to bugs

### Solution
Created `internal/space/constants.go`:
```go
package smoke

const (
    // Point rendering thresholds
    PointThresholdDefault  = 200.0
    PointThresholdAsteroid = 100.0
    PointThresholdPlanet   = 500.0
    PointThresholdMoon     = 300.0
    
    // LOD distances
    LODVeryClose = 20.0
    LODClose     = 50.0
    LODMedium    = 100.0
    LODFar       = 200.0
    
    // ... etc
)
```

Replaced all 30+ hardcoded values with constant references.

**Key Lesson**: **Runtime state must be single sourced**. Constants should be:
- Colocated in one file
- Semantically named (not just values)
- Documented with their purpose
- Used consistently across all code paths

---

## 6. Type Safety in Graphics Code

### Problem
```go
const (
    PointSizeDefault = 2.0  // Untyped float literal
    PointSizeMoon    = 4.0
)

// Later in code:
pointSize := PointSizeDefault  // Infers float64
rl.DrawSphere(pos, pointSize, ...)  // Expects float32 - ERROR
```

**Compiler error**: `cannot use pointSize (variable of type float64) as float32 value`

### Solution
Explicitly type float constants for graphics APIs:
```go
const (
    PointSizeDefault = float32(2.0)  // Explicit type
    PointSizeMoon    = float32(4.0)
)
```

**Key Lesson**: Graphics APIs (OpenGL, Raylib, etc.) use `float32` for performance. When defining constants for graphics rendering, explicitly cast to `float32` to avoid type mismatches.

---

## Architecture Principles

### Double Buffering Fundamentals
1. **Two complete copies** of state (front + back)
2. **Renderer reads ONLY from front** (with read lock)
3. **Simulation writes ONLY to back** (no lock needed)
4. **Swap is the ONLY synchronization point** (write lock required)
5. **Deep copy on initialization** ensures true independence

### Thread Communication Patterns
1. **Avoid shared locks** between rendering and simulation threads
2. **Use channels** for commands from main → simulation
3. **Process commands in simulation thread** before physics update
4. **One swap per frame** maintains consistency

### State Management
1. **Derived state** must be synced after swaps
2. **Configuration** centralized in constants
3. **Language-level protection** (remove unsafe methods)
4. **Document critical sections** with warnings

---

## Summary

The bugs stemmed from multiple architectural issues:

1. **Shallow copy bug** → Objects shared between buffers → stuttering
2. **Front buffer modification** → Race conditions → disappearing objects
3. **Cross-thread locking** → Deadlocks when changing datasets
4. **Double swapping** → Buffer flip-flopping → forward/backward motion
5. **Stale visibility** → Inconsistent state after swaps → flashing asteroids

The fixes required:
- True deep copying with explicit allocations
- Strict buffer access discipline (back-only for simulation)
- Channel-based async commands (no cross-thread locks)
- Single swap per frame (one in update, not in dataset change)
- Visibility sync after every swap

**Core takeaway**: Double buffering is simple in theory but requires strict architectural discipline. Go's type system and CSP model (channels) can enforce correct patterns when used properly.

---

## 9. Map Iteration Order Breaks Double-Buffer Parity (2026-04-03)

### Problem
After switching to a higher asteroid dataset, some belt objects flickered between two different visual sizes every other frame.

### Root Cause
`CreateBelt` iterated `config.ObjectTypes` (a `map[string]BeltObjectTypeConfig`) with `for typeName, typeConfig := range config.ObjectTypes`. Go randomises map iteration order per map and per iteration. Because dataset allocation calls `CreateBelt` **once for the back buffer and once for the front buffer** with separately seeded-but-equal RNGs:

- Call 1 (back): iterates types `["carbonaceous", "rocky"]` → consumes RNG → object at index K = large, dark
- Call 2 (front): iterates types `["rocky", "carbonaceous"]` → same RNG, different consumption sequence → object at index K = small, bright

`SwapInPlace` then assumed index correspondence and copied `front.Objects[K].Anim` onto `back.Objects[K].Meta` — cross-contaminating mismatched physical properties. The renderer alternated between two completely different objects at the same screen position every tick.

**The invariant violated**: `SwapInPlace` is only correct when `back.Objects[i]` and `front.Objects[i]` represent the same logical object across both buffers.

### Solution
Sort the type name keys before iterating in `CreateBelt`:

```go
// WRONG — non-deterministic between calls:
for typeName, typeConfig := range config.ObjectTypes { ... }

// CORRECT — identical order in every call for the same config:
typeNames := make([]string, 0, len(config.ObjectTypes))
for typeName := range config.ObjectTypes {
    typeNames = append(typeNames, typeName)
}
sort.Strings(typeNames)
for _, typeName := range typeNames {
    typeConfig := config.ObjectTypes[typeName]
    ...
}
```

### Key Lessons

1. **Any function that must produce identical output for two independent calls must never use map range as its iteration driver.** Collect keys, sort, then iterate.

2. **`SwapInPlace` is a structural contract**: it demands that `front.Objects[i]` and `back.Objects[i]` always represent the same object. Any code path that populates both buffers via separate RNG-driven calls must guarantee this invariant through sorted, deterministic ordering.

3. **Non-deterministic generation bugs are timing-sensitive** — they may not appear in tests that only call `CreateBelt` once or with single-type configs. Always write parity tests that call the generator twice with the same seed and assert matching output.

### Test Added
`TestCreateBelt_Deterministic` in `internal/sim/belts_test.go` — calls `CreateBelt` twice with the same seed and a two-type config, then asserts name, radius, and position match at every index.
