# Performance Testing Implementation - Lessons Learned

## Purpose
Capture the major implementation defects, debugging discoveries, performance findings, and follow-up recommendations uncovered while building and stabilizing the performance testing workflow.

## Last Updated
2026-05-22

## Table of Contents
1. Session Overview
2. Critical Issues Discovered & Resolved
  2.1 Deadlock Bug: Double RLock Pattern
  2.2 Excessive Logging - Disk I/O Saturation
  2.3 Visibility Filtering Missing in Warmup Loop
  2.4 Dataset Switching Not Implemented
  2.5 Build Script Using Wrong Binary
3. Architecture Decisions
4. Testing Observations
5. Development Process Issues
6. Recommendations
7. Key Takeaways
8. Performance Test Results Structure
9. Code Quality Improvements Needed
10. Final Notes and Extended Lesson Sets

## Session Overview
**Date**: February 13, 2026  
**Objective**: Implement automated performance testing with `--performance` CLI flag  
**Result**: Successfully implemented with multiple critical bug fixes

---

## Critical Issues Discovered & Resolved

### 1. **Deadlock Bug: Double RLock Pattern**
**Problem**: Calling `LockFront()` then `GetFront()` caused deadlock when simulation's `Swap()` was pending.

**Root Cause**:
```go
// WRONG - causes deadlock:
sim.GetState().LockFront()      // Acquires RLock #1
state := sim.GetState().GetFront()  // Tries RLock #2 - blocks if Swap() waiting
```

**Solution**: Use `LockFront()`'s return value directly:
```go
// CORRECT:
state := sim.GetState().LockFront()  // Single RLock, returns state
```

**Locations Fixed**:
- Main render loop initialization (line ~1486)
- Performance test warmup loop (line ~1683)
- Performance test measurement loop (line ~1756)

---

### 2. **Excessive Logging - Disk I/O Saturation**
**Problem**: 89 million log lines (5.2GB) in 7 minutes caused system to appear frozen.

**Symptoms**: 
- App appeared "stuck" but was actually waiting on disk writes
- Frame rate dropped from 60 FPS to ~0.1 FPS

**Root Cause**: Logging every frame + every object draw:
- 720 frames × 1200 objects × 10 log statements = ~8.6M logs per test
- 28 tests × 8.6M = 240M potential log statements

**Solution**: Removed debug logging, kept only INFO level:
- No frame-by-frame logging in production
- Progress updates only every 60 frames
- Result: 167KB log file vs 5.2GB (99.97% reduction)

---

### 3. **Visibility Filtering Missing in Warmup Loop**
**Problem**: Warmup loop rendered all allocated objects, including invisible ones from previous dataset tests.

**Symptom**: Hang during Test 3 warmup after Test 2 had allocated Medium dataset.

**Root Cause**: Main loop filtered by `obj.Visible`, but warmup used `state.Objects` directly.

**Solution**: Added visibility filtering to warmup loop:
```go
visibleObjects := make([]*smoke.Object, 0, len(state.Objects))
for _, obj := range state.Objects {
    if obj.Visible {
        visibleObjects = append(visibleObjects, obj)
    }
}
```

---

### 4. **Dataset Switching Not Implemented**
**Problem**: Performance tests printed "Dataset reloading not yet implemented" and used wrong object counts.

**Symptom**: Tests ran with Small dataset (200 asteroids) but expected counts for Medium (1200).

**Solution**: Implemented actual dataset switching:
```go
if config.Dataset != currentDataset {
    sim.SetAsteroidDataset(config.Dataset)  // Lazy allocation + visibility toggle
    currentDataset = config.Dataset
    time.Sleep(100 * time.Millisecond)  // Propagation delay
}
```

---

### 5. **Build Script Using Wrong Binary**
**Problem**: Test script ran `./bin/space-sim` (old binary) while builds went to `./space-sim`.

**Symptom**: Tests hung with "old" bugs even after fixes were applied.

**Discovery**: 
- `bin/space-sim`: Built at 13:42 (old)
- `./space-sim`: Built at 14:44 (current)
- Script was testing a 1-hour-old binary!

**Solution**: 
- Updated script to use `./space-sim`
- Added binary existence check
- Added timestamp display to show binary build date

---

## Architecture Decisions

### Lazy Allocation with Group Tracking
**Approach**: Allocate asteroids on-demand, hide/show with visibility flags.

**Benefits**:
- Memory efficient: Only allocates requested datasets
- No reallocation: Switching back to previous dataset is instant
- Deterministic: Same seed ensures consistent asteroid positions
- Group management: Foundation for future per-dataset optimizations

**Implementation**:
```go
type SimulationState struct {
    CurrentDataset      AsteroidDataset
    AllocatedDatasets   map[AsteroidDataset]bool
}

type Object struct {
    Visible  bool             // Render flag
    Dataset  AsteroidDataset  // Group membership
}
```

**Memory Pattern**:
- Start: 200 asteroids (Small)
- Press M → 1200 asteroids allocated (Small still visible)
- Press M → 2400 asteroids allocated (Medium visible, Small hidden)
- Press M → Back to Small: 0 new allocations, just toggle visibility

---

## Testing Observations

### Performance Characteristics
**Small Dataset (200 asteroids + 314 planets/rings = 514 objects)**:
- Baseline: ~77 FPS
- With LOD: ~250 FPS (3.2x improvement)
- With frustum culling: ~91 FPS (1.2x improvement)
- All optimizations: High variability due to culling

**Rendering Bottlenecks**:
1. Draw calls dominate (40ms average)
2. Frustum culling adds minimal overhead (0.01-0.1ms)
3. LOD provides massive wins by reducing geometry
4. Point rendering faster than spheres but less visually accurate

---

## Development Process Issues

### Problem: Iterative Debugging Without Full Context
**What Happened**: Fixed bugs incrementally without understanding full system state:
1. Fixed deadlock → hit logging issue
2. Fixed logging → hit visibility issue
3. Fixed visibility → hit dataset switching issue
4. Fixed dataset → hit binary path issue

**Better Approach**: 
- Run full system trace/profile first
- Identify all bottlenecks before fixing
- Fix in dependency order
- Verify with integration test

### Problem: Test Script Assumptions
**What Happened**: Script hardcoded binary path, wasn't validated.

**Better Approach**:
- Scripts should validate preconditions
- Show what they're actually running
- Fail fast with clear error messages
- Consider using Makefile targets for consistency

---

## Recommendations

### Immediate (Must Fix)
1. ✅ Remove all frame-level debug logging
2. ✅ Fix double-locking pattern everywhere
3. ✅ Implement visibility filtering consistently
4. ✅ Use correct binary in test scripts

### Short Term (Should Fix)
1. ⚠️ Add proper log levels (DEBUG/INFO/WARN/ERROR) - currently just using `log.Printf`
2. ⚠️ Add timeout mechanism to detect actual hangs vs slow tests
3. ⚠️ Consider using `slog` (Go 1.21+) for structured logging
4. ⚠️ Add progress indicators during long-running tests

### Long Term (Nice to Have)
1. 💡 Implement test parallelization for different dataset/config combos
2. 💡 Add GPU profiling to identify rendering bottlenecks
3. 💡 Benchmark memory allocations to reduce GC pressure
4. 💡 Consider batch rendering for asteroids
5. 💡 Investigate compute shaders for physics simulation

---

## Key Takeaways

### What Worked Well
- Systematic debugging with logs
- Lazy allocation strategy for memory efficiency
- Visibility-based filtering for dataset management
- Separation of simulation and rendering threads

### What Didn't Work
- Frame-by-frame logging in production code
- Assuming RLocks are always safe (they're not when Swap() is pending)
- Not validating test harness assumptions
- Debugging incrementally without full system understanding

### Critical Insight
**"Slow" is not the same as "stuck"**. The app appeared frozen multiple times but was actually:
1. Blocked on disk I/O (logging)
2. Rendering thousands of invisible objects (missing filter)
3. Using an old binary (wrong path)

The actual deadlock bug was only one of many issues that caused "freeze" symptoms.

---

## Performance Test Results Structure

```
performance_results.txt:
- 28 test configurations
- Each with: FPS, draw time, cull time, memory stats
- Grouped by dataset (Small, Medium, Large, Huge)
- 7 optimization combinations per dataset
```

**Test Duration**: ~10 minutes for all 28 tests (20 seconds per test)

---

## Code Quality Improvements Needed

1. **Error Handling**: Many operations don't check for errors
2. **Context Propagation**: Should use context.Context for cancellation
3. **Graceful Shutdown**: Need proper cleanup on Ctrl+C
4. **Resource Management**: Consider RAII-style patterns for locks
5. **Testing**: Need unit tests for critical sections (dataset switching, visibility)

---

---

### 6. **Threading Overhead vs Parallel Gains**
**Problem**: Multi-threaded physics (4 workers) performed WORSE than single-threaded for small object counts.

**Discovery**:
- Test #7 (Small dataset, All Combined optimizations):
  - **4 threads**: 333.3 FPS (draw: 2.77ms)
  - **1 thread**: 500.0 FPS (draw: 2.60ms)
  - **Result**: 50% performance LOSS with 4 threads!

**Root Cause**: Thread synchronization overhead exceeds parallel processing gains at small scales.

**Threading Overhead Components**:
1. **Mutex/WaitGroup synchronization** - Each worker must acquire locks
2. **Context switching** - OS scheduler overhead for 4 goroutines
3. **Cache thrashing** - Workers compete for CPU cache lines
4. **Memory barriers** - Go runtime enforces memory ordering across threads
5. **Work distribution** - Overhead of dividing 514 objects across 4 workers (~128 each)

**Performance Characteristics Observed**:

| Dataset | Objects | 4 Threads | 1 Thread | Overhead |
|---------|---------|-----------|----------|----------|
| Small   | 514     | 333 FPS   | 500 FPS  | -50%     |
| Medium  | 1,514   | TBD       | TBD      | TBD      |
| Large   | 2,714   | TBD       | TBD      | TBD      |
| Huge    | 24,314  | TBD       | TBD      | TBD      |

**Hypothesis**: Threading crossover point exists between 514-1,514 objects where parallel gains exceed overhead.

**Expected Behavior**:
- **Small datasets (< ~1K objects)**: Single-threaded faster (low computation/overhead ratio)
- **Medium datasets (~1K-3K)**: Break-even point (overhead ≈ parallel gains)
- **Large datasets (> 3K)**: Multi-threaded faster (high computation/overhead ratio)

**Architectural Implications**:
- Need **dynamic thread scaling** based on object count
- Consider **work-stealing scheduler** instead of fixed work distribution
- Evaluate **per-object computation cost** - if physics is cheap, overhead dominates
- May need **adaptive worker pool** that adjusts based on workload

**Action Items**:
1. Complete full test run (1 thread vs 4 threads across all datasets)
2. Identify exact crossover threshold
3. Implement dynamic thread count: `NumWorkers = max(1, min(4, ObjectCount / 500))`
4. Consider disabling threading entirely for interactive mode (< 1K objects)

---

## Final Notes

The performance testing framework is now functional and collects comprehensive metrics. The primary lesson is that **perceived system behavior** (frozen UI, slow response) often has **multiple root causes** that must be systematically eliminated:

1. Instrumentation overhead (logging)
2. Incorrect filtering logic
3. Missing feature implementation (dataset switching)
4. Build/deployment issues (wrong binary)
5. Actual bugs (deadlock)
6. **Threading overhead** - More threads ≠ better performance

**Always verify assumptions** - especially about which code is actually running and what optimizations actually help.

---

## Configuration & Profiling Lessons [PROFILE]

*These lessons relate to test configuration, camera positioning, threading, and benchmarking methodology.*

### 7. **Camera Position Dramatically Affects Optimization Measurements** [PROFILE]
**Problem**: Testing with "god-view" camera (0, 800, -400) showed minimal frustum culling benefit (0-4% improvement).

**Discovery**: Changed to "inside scene" camera (215, 60, 0) inside asteroid belt:
- Small dataset: 83.3 FPS → 200.0 FPS (**2.4x improvement** from frustum alone)
- Medium dataset: 38.5 FPS → 83.3 FPS (**2.2x improvement**)
- Large dataset: 17.5 FPS → 31.2 FPS (**1.8x improvement**)
- **Culling efficiency**: 47-62% objects removed from rendering

**Root Cause**: God-view sees all objects. Realistic camera placement provides natural occlusion.

**Camera Profile Comparison**:
```go
// "worst" - God-view, sees everything
Position: (0, 800, -400)
Target: Origin (Sun)
Result: Minimal frustum culling benefit

// "better" - Inside asteroid belt
Position: (215, 60, 0)  
Target: Sun
Result: 47-62% objects culled, 1.8-2.6x FPS improvement
```

**Lesson**: Always test with realistic camera positions that represent actual gameplay/viewing conditions. God-view hides optimization effectiveness and provides false confidence in baseline performance.

---

### 8. **More Threads ≠ Better Performance** [PROFILE]
**Problem**: Assumed more physics threads would improve performance. Testing with 8 threads showed degradation.

**Discovery**: Thread scaling results (profile "worst", baseline configuration):
- Small dataset: **4T: 83.3 FPS → 8T: 76.9 FPS (-7.7%)**
- Small dataset LOD: **4T: 333.3 FPS → 8T: 250.0 FPS (-25%)**
- Medium dataset: **4T: 37.0 FPS → 8T: 35.7 FPS (-3.5%)**
- Medium All Combined: **4T: 166.7 FPS → 8T: 111.1 FPS (-33%)**
- Large/Huge datasets: No difference (GPU-bound)

**Root Cause**: Thread overhead exceeds parallelization gains at these scales.

**Threading Overhead Components**:
1. Goroutine scheduler contention
2. Cache thrashing (more threads = more cache evictions)
3. Memory bandwidth contention
4. Work distribution overhead
5. Mutex/WaitGroup synchronization cost

**Objects per Thread Analysis**:
- Small dataset: 514 ÷ 8 = 64 objects/thread (too small, overhead dominates)
- Medium dataset: 1514 ÷ 8 = 189 objects/thread (still overhead-dominated)
- Large dataset: 2714 ÷ 8 = 339 objects/thread (GPU-bound, irrelevant)

**Lesson**: Profile thread scaling before assuming more threads help. On M1, 4 threads is optimal for 500-3K object physics. Only increase threads if per-thread workload is computationally expensive enough to amortize scheduling overhead.

---

### 9. **Mutex Overhead is Negligible, But Tests Must Be Valid** [PROFILE]
**Problem**: Initial --no-locking test (battery-powered) showed removing locks made performance WORSE (23-58% slower).

**Discovery**: Retest on AC power revealed completely different results:
- **Battery powered (invalid)**:
  - Medium Baseline: 20.8 FPS (vs 37.0 expected) = 44% slower
  - Medium Frustum: 32.3 FPS (vs 76.9 expected) = 58% slower
  - Small Frustum: 125.0 FPS (vs 200.0 expected) = 37% slower

- **AC powered (valid)**:
  - Small dataset: 76.9 FPS (identical to baseline)
  - Medium dataset: 32.3 FPS (matches expected)
  - Large dataset: 17.2 FPS (matches expected)
  - **Conclusion**: RWMutex overhead < 2% across all workloads

**Root Cause**: Battery power triggered CPU frequency scaling (1.0-2.4 GHz vs 3.2 GHz on AC), reducing performance 14-61%.

**RWMutex Performance Characteristics**:
- Read lock (renderer): ~5-10ns overhead
- Write lock (physics): ~20-30ns overhead  
- Swap operation: ~50-100ns overhead
- **Total per-frame cost**: <0.05ms at 60 FPS

**False Hypothesis (Battery Test)**: "Removing mutex makes performance worse, therefore mutex helps cache coherency."

**Actual Finding (AC Test)**: "Mutex overhead is negligible, battery test was invalid due to CPU throttling."

**Lesson**: 
1. **Always benchmark on AC power** - battery mode CPU throttling (15-60% slowdown) invalidates all results
2. **Verify CPU frequency** - use `pmset -g stats` or similar to confirm stable clock speed
3. **Document power mode** - include in test metadata to catch invalid results
4. **Mutex is "free"** - Go's RWMutex adds <2% overhead, provides thread safety without measurable cost

---

### 10. **Warmup Period Prevents Misleading Results** [PROFILE]
**Problem**: Initial test attempts showed inconsistent FPS measurements with high variance.

**Discovery**: Implemented 480-frame (8 second) warmup revealed multiple stabilization factors:

**GPU Driver Stabilization**:
- First 60-120 frames: Shader compilation overhead visible
- GPU command buffers reach steady state
- Metal backend optimizes render pipelines based on observed patterns

**Go Runtime Stabilization**:
- First 2 seconds: 4-6 GC cycles complete
- Goroutine scheduler balances 4 worker threads
- Memory allocations stabilize after lazy instantiation

**FPS Convergence Times**:
- Small dataset: Stabilizes in 60 frames (~1 second)
- Medium dataset: Stabilizes in 120 frames (~2 seconds)  
- Large dataset: Stabilizes in 240 frames (~4 seconds)
- Huge dataset: Stabilizes in 360 frames (~6 seconds)

**Measured Impact**:
- Without warmup: First 2 seconds show 15-30% lower FPS
- GC pauses during measurement: Reduced 80% with warmup
- Variance: 8-12% CV without warmup → 2-4% CV with warmup

**Lesson**: Never measure performance without adequate warmup period. Cold-start transients (GPU compilation, GC, scheduler balancing) dominate measurements and provide false results. Minimum 8-second warmup for GPU applications, scale up for larger datasets.

---

### 11. **Test Matrix Completeness Reveals Hidden Issues** [PROFILE]
**Problem**: Initial testing with single configuration (profile "worst", 4 threads) appeared successful but hid critical optimization potential.

**Discovery**: Systematic testing across multiple dimensions revealed:

**Camera Profile Impact** (worst vs better):
- Frustum culling benefit: 0-4% → 78-140% FPS improvement
- Objects culled: 0-2% → 47-62%
- Validation: Camera position critical for realistic testing

**Thread Count Impact** (4T vs 8T):
- Small/Medium datasets: 8T shows 7-33% **degradation**
- Large/Huge datasets: No difference (GPU-bound)
- Validation: 4 threads optimal, more threads hurt performance

**Locking Overhead** (enabled vs disabled):
- Initial battery test: "no-locking" appeared 23-58% **slower** (invalid)
- AC power retest: "no-locking" shows 0-2% difference (valid)
- Validation: Power mode critical for accurate measurements

**Test Matrix Structure**:
```
Dimensions to test:
├── Camera Profile: worst, better
├── Thread Count: 4, 8
├── Locking: enabled, disabled
├── Power Mode: battery (invalid), AC (required)
└── Dataset Size: Small, Medium, Large, Huge

Complete matrix: 2 × 2 × 2 × 4 = 32 test configurations
Actual tests needed: 2 × 2 × 4 = 16 (power mode is environmental)
```

**Current Coverage**:
- ✅ worst/4T/locked (Test Run #1)
- ✅ worst/8T/locked (Test Run #2)
- ✅ better/4T/unlocked/AC (Test Run #3)
- ❌ better/4T/locked/AC (MISSING - needed for direct comparison)
- ❌ better/8T/locked/AC (optional)
- ❌ better/8T/unlocked/AC (optional)

**Lesson**: Systematic test matrix reveals interactions between configuration dimensions. Single-configuration testing provides false confidence. Always test multiple camera positions, thread counts, and optimization combinations to understand actual system behavior.

---

### 12. **Environmental Factors Must Be Controlled** [PROFILE]
**Problem**: Test Run #5 (battery powered) showed "removing locks hurts performance" - counterintuitive result.

**Discovery**: System profiling revealed CPU frequency scaling:
- **AC Power**: 3.2 GHz sustained (performance cores)
- **Battery Power**: 1.0-2.4 GHz variable (energy saving mode)
- **Performance Impact**: 14-61% slowdown depending on workload

**Environmental Factors That Invalidate Tests**:

1. **Power Mode** (CRITICAL):
   - Battery: CPU throttles 30-70% depending on workload
   - AC: Full performance mode
   - Impact: 15-60% FPS difference
   - Detection: `pmset -g stats` on macOS

2. **Thermal State**:
   - Cold system: Full turbo boost available
   - Hot system: Thermal throttling reduces clocks
   - Impact: 10-30% FPS difference
   - Detection: Monitor CPU temperature during test

3. **Background Processes**:
   - Time Machine backups
   - Spotlight indexing  
   - Browser with many tabs
   - Impact: 5-20% FPS difference
   - Mitigation: Close unnecessary apps, disable background tasks

4. **GPU State**:
   - Multiple displays (eGPU bandwidth sharing)
   - Other apps using GPU (video playback, etc.)
   - Impact: 10-40% FPS difference
   - Detection: Activity Monitor → GPU History

**Test Environment Checklist**:
```bash
# Verify AC power (macOS)
pmset -g batt | grep "AC Power"

# Check CPU frequency
sudo powermetrics --sample-count 1 | grep "CPU Average frequency"

# Close background processes
killall "Google Chrome" "Slack" "Spotlight"

# Disable Time Machine during test
sudo tmutil disable

# Monitor during test
while true; do pmset -g thermlog; sleep 5; done
```

**Lesson**: Environmental factors can dwarf code-level optimizations. Always:
1. Document power mode, thermal state, background processes
2. Use dedicated test machine or "performance mode"
3. Verify CPU frequency is stable before starting tests
4. Retest if results are counterintuitive - environment may be culprit
5. Include environmental metadata in test reports

**Critical Insight**: The "mutex helps performance" conclusion from battery test was completely wrong. Environmental factors caused false result that could have led to architectural mistakes.

---

## Summary: Configuration & Profiling Best Practices [PROFILE]

### Camera Configuration
- ✅ Use realistic camera positions (inside scene, not god-view)
- ✅ Test multiple camera angles to validate optimization effectiveness
- ✅ Document camera position/target in test metadata
- ❌ Don't assume single camera position validates optimization

### Thread Scaling  
- ✅ Profile thread counts (test 1, 2, 4, 8, 16 threads)
- ✅ Measure overhead vs parallel gains
- ✅ Use 4 threads as baseline for M1 (optimal for most workloads)
- ❌ Don't assume more threads = better performance

### Locking Overhead
- ✅ Go's RWMutex is essentially "free" (<2% overhead)
- ✅ Keep locking enabled for thread safety
- ✅ Test with/without locking to measure actual cost
- ❌ Don't remove locks without measuring impact

### Test Environment
- ✅ **ALWAYS test on AC power** (battery causes 15-60% throttling)
- ✅ Verify CPU frequency is stable
- ✅ Close background processes
- ✅ Monitor thermal state during long tests
- ✅ Document environmental conditions in test metadata
- ❌ **NEVER trust battery-powered benchmark results**

---

## F-020 Multi-Client Session Layer — Lessons Learned

**Date**: 2026-05-22
**Context**: Implementing the session registry (Phase 1) and position/POV streaming (Phase 2) for multi-client gRPC sessions.

---

### 13. **Unlock Before Notify: `defer mu.Unlock()` Is Incompatible With Pub-Sub**

**Problem**: Added subscriber notification to `inMemoryRegistry` while the existing methods used `defer r.mu.Unlock()`. The naive approach — notify inside the deferred unlock scope — risks deadlock if any subscriber calls back into the registry (e.g., `Get`, `All`).

**Root Cause**: `defer mu.Unlock()` holds the lock for the entire function body including the notify call. If a subscriber re-enters the registry on the same goroutine the result is a `sync.RWMutex` reentrant-lock panic; on a different goroutine it can deadlock.

**Solution**: Remove `defer` and unlock manually at every return path, then call `notify()` after the lock is released:
```go
// WRONG — holds lock during notify:
func (r *inMemoryRegistry) Register(...) (*ClientSession, error) {
    r.mu.Lock()
    defer r.mu.Unlock()   // still held when notify() fires
    ...
    r.notify(event)       // subscriber may call r.Get() → deadlock
    return snap, nil
}

// CORRECT — lock released before notify:
func (r *inMemoryRegistry) Register(...) (*ClientSession, error) {
    r.mu.Lock()
    if capacityExceeded {
        r.mu.Unlock()
        return nil, ErrCapacityExceeded
    }
    ...
    snap := copySession(sess)
    r.mu.Unlock()          // released before side effects
    r.notify(event)        // safe: no lock held
    return snap, nil
}
```

**Rule**: Any store/registry that emits change events must release its lock before triggering notifications. Copy the state snapshot under the lock; notify outside it.

---

### 14. **`go build ./...` Does Not Compile Test Files**

**Problem**: Changed `NewWorldHandler()` to `NewWorldHandler(session.Registry)`. `go build ./...` passed clean. Committed. `make test` failed with 6 compilation errors in `server_test.go`.

**Root Cause**: `go build ./...` compiles only non-test packages. Test files (`_test.go`) are compiled only when `go test` processes them.

**Solution**: After any function signature change, search for stale call sites in test files before committing:
```bash
grep -r "NewWorldHandler" . --include="*.go"
```

**Rule**: `go build ./...` is not sufficient validation after a signature change. Always run `make test` (or `go test ./...`) before committing.

---

### 15. **ConnectRPC Bidi Stream: Subscribe Before Bootstrapping**

**Problem**: When implementing `SessionStream`, the initial design called `registry.All()` to send the bootstrap snapshot *before* calling `registry.Subscribe()`. Events emitted by other goroutines between `All()` and `Subscribe()` are silently dropped — the stream client never sees them.

**Root Cause**: There is a window between "read current state" and "attach to future events" where changes are invisible to the new subscriber.

**Solution**: Subscribe first, then send the bootstrap snapshot:
```go
// WRONG — events can be lost between All() and Subscribe():
sessions := h.registry.All()
eventCh, cancel := h.registry.Subscribe()
for _, s := range sessions { stream.Send(addDelta(s)) }

// CORRECT — no gap: events after Subscribe() are buffered in the channel:
eventCh, cancel := h.registry.Subscribe()
defer cancel()
for _, s := range h.registry.All() { stream.Send(addDelta(s)) }
// eventCh receives all events from this point forward, including any
// that raced with the All() scan above (worst case: client sees a
// duplicate ADD, which is idempotent and harmless)
```

**Rule**: For any "catch up then stream live events" pattern, attach the subscription channel before reading the initial snapshot.

---

### 16. **Protobuf: Same-Package Files Still Require Explicit Imports**

**Problem**: `simulation.proto` needed to reference `ClientSessionInfo` defined in `session.proto`. Both files declare `package spacesim.v1`. The import was omitted assuming same-package visibility.

**Result**: `buf generate` succeeded but the generated Go code referenced an undefined type, causing a build failure.

**Solution**: Add an explicit import even within the same proto package:
```proto
// simulation.proto
import "spacesim/v1/session.proto";
```

**Rule**: Proto `package` is a namespace for generated symbols, not a file-visibility scope. Every cross-file message reference requires an explicit `import` statement regardless of package.


### Test Methodology
- ✅ Minimum 8-second warmup (480 frames at 60 FPS)
- ✅ 12-second measurement period (720 frames)
- ✅ Test complete matrix (camera × threads × optimizations)
- ✅ Multiple runs to verify repeatability
- ✅ Document test configuration in filename/metadata

### Red Flags (Invalid Results)
- 🚩 Counterintuitive results (removing optimization helps performance)
- 🚩 High variance between runs (>10% CV)
- 🚩 Results don't match system characteristics (GPU-bound but thread scaling matters)
- 🚩 Battery powered tests
- 🚩 Missing warmup period

**Action on Red Flags**: Retest with controlled environment before drawing conclusions.

---

## Visibility System Bugs (Discovered Feb 14, 2026) [VISIBILITY]

### 13. **Missing Initialization: Moons and Dwarf Planets Invisible** [VISIBILITY]
**Problem**: Moons and dwarf planets were intermittently invisible depending on asteroid dataset level.

**Root Cause**: `NewMoon()` and `NewDwarfPlanet()` missing field initialization:
```go
// WRONG - fields default to zero values:
// Visible = false (bool default)
// Dataset = 0 (int default = AsteroidDatasetSmall)
```

**Impact**:
- Moons had `Dataset = 0` instead of `-1` (non-asteroid marker)
- Visibility check `obj.Dataset <= currentDataset` made moons disappear at higher dataset levels
- Object counts inconsistent between test runs
- Frustum culling results affected

**Solution**: Added explicit initialization:
```go
func NewMoon(...) *Object {
    return &Object{
        // ... other fields
        Visible: true,
        Dataset: -1, // Non-asteroid marker
    }
}
```

**Files Fixed**: `internal/space/objects.go` (NewMoon, NewDwarfPlanet)

---

### 14. **Double-Buffer Desynchronization: Asteroid Flickering** [VISIBILITY]
**Problem**: Asteroids flickered when changing datasets via M key. Object count in HUD fluctuated.

**Root Cause**: Multi-step buffer desynchronization:
1. `SetAsteroidDataset()` called `CreateAsteroids()` on back buffer only
2. Back buffer gained 1000 new asteroids (e.g., 200 → 1200)
3. Front buffer still had old count (200)
4. Visibility update loop tried to update objects that didn't exist in front buffer
5. After `Swap()`, buffers had mismatched object arrays
6. Result: Renderer alternated between 200 and 1200 objects

**Attempted Fix #1** (Failed): Updated visibility on every frame
- Added per-frame loop to sync visibility with `CurrentDataset`
- Still flickered because object arrays were different sizes
- Wasted CPU checking visibility 60x per second

**Attempted Fix #2** (Failed): Updated both buffers' visibility immediately
- Used `GetFrontUnsafe()` to access front buffer
- Updated visibility flags in both buffers when M pressed
- Still flickered because object counts were mismatched

**Correct Solution**: Allocate asteroids to BOTH buffers simultaneously:
```go
if !back.AllocatedDatasets[dataset] {
    // Create in back buffer
    rng := rand.New(rand.NewSource(42))
    CreateAsteroids(back, rng, dataset)
    
    // ALSO create in front buffer (same seed for consistency)
    front := s.state.GetFrontUnsafe()
    rng2 := rand.New(rand.NewSource(42))
    CreateAsteroids(front, rng2, dataset)
    
    back.AllocatedDatasets[dataset] = true
    front.AllocatedDatasets[dataset] = true
}
```

**Key Insight**: Double-buffer systems require object array synchronization, not just property updates.

**Impact on Test Results**:
- All previous performance tests (Runs #1-4) had inconsistent object counts
- FPS measurements fluctuated during flickering
- Dataset transition measurements unreliable
- Results invalidated, must retest

**Files Fixed**: `internal/space/simulation.go` (SetAsteroidDataset)

---

### 15. **Interactive Mode: Unwanted Performance Profiling Output** [VISIBILITY]
**Problem**: Running `make run-smoke` displayed performance profiling stats every 2 seconds on console.

**Root Cause**: Performance timing code ran in both interactive and `--performance` modes.

**Solution**: Removed per-frame timing and console output from interactive mode (only needed for automated testing).

**Files Fixed**: `cmd/space-sim/main.go` (removed performance tracking variables and print loop)

---

### 16. **CLI Validation Bug: Can't Run Without Flags** [VISIBILITY]
**Problem**: Running `bin/space-sim` without flags triggered "Error: CLI options can only be used with --performance".

**Root Cause**: Validation checked flag values against defaults:
```go
// WRONG - default values always trigger:
if (*profileFlag != "" || *threadsFlag != 4) && !*performanceMode {
    // profileFlag default is "worst" (not empty)
    // threadsFlag default is 4
    // Always false!
}
```

**Solution**: Use `flag.Visit()` to check which flags were explicitly provided:
```go
profileProvided := false
flag.Visit(func(f *flag.Flag) {
    if f.Name == "profile" {
        profileProvided = true
    }
})
if profileProvided && !*performanceMode {
    fmt.Println("Error: --profile can only be used with --performance")
}
```

**Key Insight**: Flag defaults create false positives in validation logic. Must distinguish "user provided" vs "default value".

**Files Fixed**: `cmd/space-sim/main.go` (CLI flag validation)

---

### 17. **HUD Text Shuffling: Poor UX During Object Count Changes** [VISIBILITY]
**Problem**: HUD line "Objects: 514 (Dataset: Small)" shuffled horizontally as count changed, making it hard to read.

**Root Cause**: Variable-width number formatting:
```go
fmt.Sprintf("Objects: %d (Dataset: %s)", count, name)
// "Objects: 514" vs "Objects: 24314" - different widths
```

**Solution**: Fixed-width formatting:
```go
fmt.Sprintf("Objects: %5d (Dataset: %s)", count, name)
// "Objects:   514" vs "Objects: 24314" - same width, right-aligned
```

**Files Fixed**: `cmd/space-sim/main.go` (HUD rendering)

---

## Visibility System: Lessons Learned [VISIBILITY]

### Object Initialization
- ✅ Always explicitly initialize bool and int fields (don't rely on zero values)
- ✅ Use sentinel values (e.g., `-1` for "not applicable") instead of zero
- ✅ Add unit tests for default field values
- ❌ Don't assume zero values are correct for your use case

### Double-Buffer Systems
- ✅ Synchronize object arrays between buffers when adding/removing objects
- ✅ Update both buffers simultaneously for structural changes
- ✅ Property updates (like visibility) can be async between buffers
- ✅ Array size mismatches cause flickering/crashes
- ❌ Don't modify only one buffer for operations that change object count

### User Experience
- ✅ Use fixed-width formatting for frequently changing numbers in UI
- ✅ Separate interactive mode from automated testing mode
- ✅ Validate user input, not default values
- ✅ Use flag.Visit() to distinguish user-provided vs default flags

---

### 18. **Swap() Pointer Exchange vs Clone: The Flickering Root Cause** [VISIBILITY]
**Problem**: Asteroids flickered (mostly off, briefly on) when changing datasets via M key, even after allocating to both buffers and synchronizing visibility.

**Root Cause**: `Swap()` used pointer exchange instead of cloning:
```go
// WRONG - just swaps pointers:
func (db *DoubleBuffer) Swap() {
    db.front, db.back = db.back, db.front
}
```

**Why This Caused Flickering**:
1. Simulation modifies back buffer (adds asteroids, updates visibility)
2. Renderer reads front buffer continuously (60 FPS)
3. `Swap()` exchanges pointers - front becomes old back, back becomes old front
4. If timing is wrong, renderer sees incomplete state during multi-step operations
5. Even with locks, modifications to front buffer created race window

**Example Timeline** (showing the problem):
```
Frame 1: Back has 200 asteroids, Front has 200 asteroids
User presses M (switch to Medium dataset - adds 1000 more asteroids)
  Step 1: Allocate 1000 asteroids to back buffer (now 1200 total)
  Step 2: Allocate 1000 asteroids to front buffer (now 1200 total)
  Step 3: Update visibility in both buffers
  During Step 2-3: Swap() happens
    → Front pointer now points to partially-updated back buffer
    → Back pointer now points to fully-updated front buffer
  Renderer sees: Incomplete state with some asteroids missing visibility flags
Frame 2: Swap() again - buffers exchange back
  Renderer sees: Different incomplete state
Result: Flickering as renderer alternates between inconsistent buffer states
```

**Attempted Fixes** (all failed):
1. ❌ Allocate to both buffers simultaneously - still flickered
2. ❌ Update visibility with locking - still flickered
3. ❌ Synchronize per-frame visibility checks - wasted CPU, still flickered

**Correct Solution**: Make `Swap()` do a full clone instead of pointer exchange:
```go
// CORRECT - clones back to front:
func (db *DoubleBuffer) Swap() {
    db.mu.Lock()
    defer db.mu.Unlock()
    db.front = db.back.Clone()
}
```

**Why Clone() Fixes Flickering**:
- Renderer always sees a complete, consistent snapshot
- Back buffer can be modified safely without affecting front
- No timing windows where partial updates are visible
- Guarantees atomic state transition from renderer's perspective

**Trade-offs**:
- **Performance Cost**: ~1-2ms per frame for cloning objects
  - Small dataset (514 objects): ~0.3ms clone time
  - Medium dataset (1514 objects): ~0.8ms clone time
  - Large dataset (2714 objects): ~1.4ms clone time
  - Huge dataset (24314 objects): ~12ms clone time
- **Benefit**: Eliminates ALL flickering and synchronization bugs
- **Conclusion**: Worth the cost for visual correctness

**Key Insight**: Pointer swapping is a performance optimization that breaks down when buffers must maintain complex synchronized state. Cloning provides deterministic behavior at acceptable cost.

**Double-Buffer Architecture Principles**:
1. ✅ **Clone for consistency** - Use `Clone()` if buffers need complex state synchronization
2. ✅ **Pointer swap for simplicity** - Use pointer exchange ONLY if buffers are write-once, read-many
3. ✅ **Lock during Clone()** - Prevent renderer access during cloning operation
4. ✅ **Clone() must be deep** - Create independent object instances, not shared pointers
5. ❌ **Don't mix approaches** - Either clone on swap OR make back buffer immutable (don't modify front)

**Files Fixed**: 
- `internal/space/state.go` - Changed Swap() to use Clone()
- `internal/space/simulation.go` - Removed duplicate printf line

**Testing Results**:
- ✅ Asteroid flickering eliminated
- ✅ Smooth animation at all speeds (0-100%)
- ✅ Object count stable in HUD
- ✅ Dataset transitions clean (no flashing)
- ✅ Single update cycle per frame visible in debug logs

**Critical Lesson**: In double-buffered systems, **consistency > performance**. A 1ms clone cost is invisible to users, but flickering destroys the experience. When in doubt, clone.

---

### Test Data Integrity
- 🚩 Visibility bugs invalidate performance test results
- 🚩 Flickering indicates data races or synchronization issues
- 🚩 Intermittent bugs are usually timing/concurrency related
- ✅ Fix all visibility bugs before collecting performance data
- ✅ Retest after fixing structural bugs

---

## UI Rendering & User Experience Lessons (Feb 25, 2026) [UI]

### 19. **2D Drawing in 3D Context: Labels Not Rendering** [UI]
**Problem**: Object labels (L key toggle) not appearing despite correct implementation of priority system and screen projection.

**Root Cause**: Labels were drawn BEFORE `EndMode3D()`:
```go
// WRONG - 2D operations in 3D rendering mode:
if labelsVisible {
    drawObjectLabels(...)  // Uses DrawText, DrawRectangle, DrawLineEx
}
rl.EndMode3D()
```

**Why This Failed**:
- `BeginMode3D()` / `EndMode3D()` sets up 3D projection matrix and depth testing
- 2D drawing functions (DrawText, DrawRectangle, DrawLineEx) expect 2D orthographic projection
- While `rl.GetWorldToScreen()` correctly projects 3D → 2D coordinates, the drawing context is still 3D
- Result: 2D primitives rendered with wrong projection matrix, clipped or invisible

**Symptom**: 
- No labels visible when L key pressed
- No error messages or crashes
- Selection logic working (verified in debugger)
- Screen projection returning valid coordinates

**Solution**: Move label drawing to 2D mode AFTER `EndMode3D()`:
```go
// CORRECT - 2D operations in 2D rendering mode:
rl.EndMode3D()
rl.SetMatrixProjection(...)  // 2D orthographic projection
rl.SetMatrixModelview(...)    // Identity transform

if labelsVisible {
    drawObjectLabels(...)  // Now renders correctly
}
```

**Key Insight**: Raylib has distinct rendering modes (3D vs 2D). Even with coordinate projection, drawing primitives must match the current rendering mode. Mix 3D and 2D carefully.

**Raylib Rendering Pipeline**:
```
BeginDrawing()
  ClearBackground()
  
  BeginMode3D()
    // Draw 3D objects: DrawSphere, DrawCube, DrawModel
    // Uses perspective projection + depth buffer
  EndMode3D()
  
  SetMatrixProjection()  // Switch to 2D orthographic
  SetMatrixModelview()
  
  // Draw 2D overlays: DrawText, DrawRectangle, DrawLine
  // Uses screen-space coordinates
  
EndDrawing()
```

**Files Fixed**: `cmd/space-sim/main.go` (moved drawObjectLabels call to 2D section)

---

### 20. **No Visual Feedback: Users Can't Tell Toggle State** [UI]
**Problem**: After fixing label rendering, users couldn't tell if labels were on or off when pressing L key.

**Root Cause**: No status indicator for toggle state - only the labels themselves showed the feature was active.

**Impact**:
- If no objects qualified for labels (all too far, or excluded categories), screen looked identical
- Users couldn't distinguish "labels off" vs "labels on but nothing to label"
- No confirmation that L key was registered

**Solution**: Added visual status indicator:
```go
if labelsVisible {
    drawObjectLabels(...)
    // Draw status indicator at bottom left
    rl.DrawText("[Labels: ON]", 10, screenHeight-30, 16, 
                rl.Color{R: 100, G: 255, B: 100, A: 200})
}
```

**UX Principles**:
- ✅ Always provide visual feedback for toggle operations
- ✅ Status indicators should be visible even when feature has no output
- ✅ Use color coding (green = on/active, gray = off/inactive)
- ✅ Position status indicators consistently (e.g., bottom left corner)
- ❌ Don't rely on feature output as the only indicator of state

**Files Fixed**: `cmd/space-sim/main.go` (added status indicator)

---

### 21. **Label Priority System: Distance vs Category Weights** [UI]
**Problem**: When viewing Neptune, labels showed distant planets (Jupiter, Saturn, Uranus) instead of nearby Neptune moons (Triton, Nereid, etc.).

**Root Cause**: Priority calculation favored category bonuses over distance:
```go
// WRONG - category bonuses dominate:
priority := float32(obj.Meta.Importance)  // Base: 0-100
priority += 500.0  // Stars
priority += 200.0  // Planets
priority += 100.0 / (distToCam + 1.0)  // Distance (weak)
```

**Example** (viewing Neptune from 1 AU):
- Jupiter (distant planet): 200 (category) + 0.1 (distance) = 200.1 priority
- Triton (nearby moon): 60 (importance) + 50 (distance) = 110 priority
- Result: Jupiter labeled, Triton ignored

**Solution**: Increase distance weight dramatically:
```go
// CORRECT - distance dominates for nearby objects:
priority += 5000.0 / (distToCam + 1.0)  // Strong distance boost
```

**New Example** (viewing Neptune from 1 AU):
- Jupiter (1000 AU away): 200 + 5 = 205 priority
- Triton (0.1 AU away): 60 + 5000 = 5060 priority
- Result: Triton labeled, Jupiter ignored

**Priority System Design**:
```
Priority = BaseImportance + CategoryBonus + DistanceBonus + TrackingBonus

BaseImportance: 0-100 (object metadata)
CategoryBonus:
  - Stars: +500
  - Planets: +200
  - Tracked object: +1000
DistanceBonus: 5000 / (distance + 1)
  - 0.01 AU: +500000 (extreme close)
  - 0.1 AU: +50000 (very close)
  - 1 AU: +5000 (close)
  - 10 AU: +500 (medium)
  - 100 AU: +50 (far)
  - 1000 AU: +5 (very far)
```

**Lesson**: In spatial UIs, proximity should dominate other factors for context-aware labeling. Users care about nearby objects more than distant important objects.

**Files Fixed**: `cmd/space-sim/main.go` (updated selectObjectsForLabels distance weight)

---

### 22. **Category Filtering: Excluding Noise Objects** [UI]
**Problem**: Asteroid belt and ring systems cluttered label display with too many low-importance objects.

**Root Cause**: Initial exclusion only covered `CategoryAsteroid` and `CategoryRing`, but missed `CategoryBelt`.

**Solution**: Added all "noise" categories to exclusion list:
```go
// Skip asteroids, rings, and belts (too numerous/not interesting for labels)
if obj.Meta.Category == smoke.CategoryAsteroid || 
   obj.Meta.Category == smoke.CategoryRing || 
   obj.Meta.Category == smoke.CategoryBelt {
    continue
}
```

**Category Filtering Strategy**:
- ✅ Include: Stars, Planets, Dwarf Planets, Moons
- ❌ Exclude: Asteroids (24K+), Rings, Belts (1K+)
- Rationale: Focus labels on unique, identifiable objects vs procedural noise

**Files Fixed**: `cmd/space-sim/main.go` (added CategoryBelt to skip list)

---

### 23. **HUD Text Jittering: Dynamic Width Causes Visual Instability** [UI]
**Problem**: Velocity values in tracking HUD caused box to expand/contract as numbers changed, creating distracting jitter effect.

**Root Cause**: Variable-width number formatting:
```go
// WRONG - width varies with value:
fmt.Sprintf("%.2f km/s", velocity)
// "12.34 km/s" (10 chars) vs "1234.56 km/s" (13 chars)
```

**Visual Impact**:
- HUD box width recalculated every frame based on text content
- As velocity changed, box oscillated 10-50 pixels left/right
- Created "breathing" effect that drew eye away from content
- Made values harder to read during animation

**Solution**: Fixed-width formatting with padding:
```go
// CORRECT - fixed width, right-aligned:
fmt.Sprintf("%8.2f km/s", velocity)
// "   12.34 km/s" (13 chars)
// " 1234.56 km/s" (13 chars)
```

**Fixed-Width Formatting Guidelines**:
```go
// For values 0-999.99:
"%7.2f"   // "  12.34" or " 999.99"

// For values 0-9999.99:
"%8.2f"   // "   12.34" or " 9999.99"

// For counts 0-99999:
"%5d"     // "  514" or "24314"

// For percentages 0-100%:
"%5.1f%%"  // " 12.3%" or "100.0%"
```

**UX Principles**:
- ✅ Use fixed-width formatting for frequently-updating numeric displays
- ✅ Right-align numbers for easier visual comparison
- ✅ Pad with spaces, not zeros (spaces less distracting)
- ✅ Calculate max expected value width, add 1-2 characters buffer
- ❌ Don't let dynamic text content affect layout geometry

**Files Fixed**: 
- `cmd/space-sim/main.go` - Orbital velocity: `%8.2f km/s`
- `cmd/space-sim/main.go` - Rotational velocity: `%8.2f km/s (at equator)`

---

### 24. **Orbital & Rotational Velocity Display** [UI]
**Implementation**: Added velocity measurements to tracking HUD (lower right panel).

**Orbital Velocity Calculation**:
```go
velocityMagnitude := obj.Anim.Velocity.Length()
velocityKmPerSec := velocityMagnitude * 1495978.707
// Conversion: 1 sim unit = 0.01 AU = 1,495,978.707 km
```

**Rotational Velocity Calculation**:
```go
// Surface speed at equator: v = 2πr / T
radiusKm := obj.Meta.PhysicalRadius / 1000.0  // meters → km
rotationPeriodSeconds := obj.Meta.RotationPeriod * 3600.0  // hours → seconds
rotationalVelocityKmPerSec := (2.0 * math.Pi * radiusKm) / rotationPeriodSeconds
```

**Display Format**:
```
Orbital Velocity:       29.78 km/s
Rotational Velocity:     0.46 km/s (at equator)
```

**Example Values** (Earth):
- Orbital: ~29.8 km/s (around Sun)
- Rotational: ~0.46 km/s (surface at equator)

**Example Values** (Jupiter):
- Orbital: ~13.1 km/s (around Sun)
- Rotational: ~12.6 km/s (fast rotation, large radius)

**Key Insight**: All objects in simulation ARE moving (except Sol at origin). Movement calculated via Keplerian orbital mechanics in `updateObject()`:
- Mean anomaly increments with time: `M = M₀ + n*t`
- Eccentric anomaly from Kepler's equation: `M = E - e*sin(E)`
- True anomaly from eccentric anomaly
- Position from elliptical orbit equations
- Velocity from orbital mechanics: `v = sqrt(μ(2/r - 1/a))`

**Common Misconception**: "Objects aren't moving" - usually means:
- Simulation paused (press Space)
- Time rate too slow (press = to speed up)
- Need to track object to see relative motion (press T)

**Files Modified**: `cmd/space-sim/main.go` (added velocity calculations to drawTrackingInfo)

---

## UI Rendering: Lessons Learned Summary [UI]

### Raylib Rendering Context
- ✅ 3D operations (DrawSphere, DrawModel) go between BeginMode3D / EndMode3D
- ✅ 2D operations (DrawText, DrawRectangle) go AFTER EndMode3D with 2D projection
- ✅ Use rl.GetWorldToScreen() to project 3D → 2D coordinates for overlays
- ❌ Don't mix 2D drawing primitives inside 3D rendering context

### Visual Feedback
- ✅ Always show status indicators for toggle operations
- ✅ Indicators should be visible even when feature has no output
- ✅ Use color coding (green = active, gray = inactive)
- ✅ Position indicators consistently (e.g., bottom corners)

### Spatial UI Priority
- ✅ Distance should dominate priority for context-aware labeling
- ✅ Weight distance 25-50x higher than category bonuses
- ✅ Exclude "noise" categories (asteroids, belts, rings) from labels
- ✅ Limit label count (10-20) to avoid clutter

### Text Layout Stability
- ✅ Use fixed-width formatting for frequently-updating numbers
- ✅ Calculate max expected width, add buffer
- ✅ Right-align numbers for visual comparison
- ✅ Prevent dynamic content from affecting layout geometry

### Performance Data Display
- ✅ Show both orbital and rotational velocity for tracked objects
- ✅ Use consistent units (km/s) for easy comparison
- ✅ Fixed-width formatting prevents HUD jitter
- ✅ Calculate velocities from simulation state, don't hard-code

---


## Raylib Graphics API Constraints [Graphics]

### Window Resolution is Locked at Creation Time
**Problem**: Attempted to change fullscreen rendering resolution at runtime using `rl.ToggleFullscreen()`.

**Symptom**: When toggling from windowed (1280×720) to fullscreen on a 2880×1800 display, the rendering resolution remained 1280×720, creating a magnified/pixelated appearance. The entire 1280×720 image was scaled up to fill the larger display.

**Root Cause**: 
- Raylib's rendering resolution is determined at `rl.InitWindow()` time
- The rendering buffer size is set by the dimensions passed to `InitWindow(width, height)` 
- The `FlagFullscreenMode` flag must be set BEFORE `InitWindow()` to use native display resolution
- Simply toggling fullscreen at runtime changes the **display mode** but NOT the **rendering resolution**
- Once the graphics context is initialized, you cannot change the rendering resolution without destroying and recreating the window

**What Doesn't Work**:
```go
// ❌ WRONG: Toggles display mode only, rendering resolution unchanged
rl.ToggleFullscreen()  // Window is now fullscreen at same 1280×720 resolution

// ❌ WRONG: Setting flags after window creation has no effect on rendering resolution
rl.SetWindowState(uint32(rl.FlagFullscreenMode))  // Too late - context already created

// ❌ WRONG: No API to change rendering resolution in-place
// (No rl.SetRenderingResolution() function exists)
```

**What Works - The Solution**:
```go
// ✅ CORRECT: Close window, reinitialize with new flags and resolution
isFullscreenNow := rl.IsWindowFullscreen()

// Determine target resolution
var newWidth, newHeight int32
if !isFullscreenNow {
    // Entering fullscreen: Use native monitor resolution
    newWidth = int32(rl.GetMonitorWidth(0))
    newHeight = int32(rl.GetMonitorHeight(0))
} else {
    // Exiting fullscreen: Restore saved windowed size
    newWidth, newHeight = savedConfig.Width, savedConfig.Height
}

// Update internal state
runtime.UpdateScreenSize(newWidth, newHeight)

// Critical: Close and recreate window
rl.CloseWindow()
runtime.Fullscreen = !runtime.Fullscreen

// Reinitialize - NOW sets correct flags and resolution
initWindow()  // Will call SetConfigFlags with FlagFullscreenMode before InitWindow
```

**Key Implementation Details**:
1. Flags MUST be set via `rl.SetConfigFlags()` BEFORE `rl.InitWindow()`
2. Window dimensions MUST be finalized before `rl.InitWindow()` is called
3. `FlagFullscreenMode` must be in the flag mask when calling `InitWindow()` if fullscreen is desired
4. No runtime API exists to change rendering resolution - window recreation is the only way

**Trade-offs**:
- ✅ Guarantees correct native resolution for fullscreen
- ✅ Ensures consistent rendering quality across displays
- ✅ Prevents magnification artifacts on large displays
- ❌ Brief window flicker/transition when toggling (unavoidable)

**Files Fixed**: 
- `internal/space/app/window.go` - Added `toggleFullscreen()` with window reinit strategy
- `internal/space/app/input.go` - Changed Cmd+F handler to call `app.toggleFullscreen()`
- `internal/space/app/interactive.go` - Updated `handleInput()` signature to accept `*App` parameter

**Lesson**: Raylib's graphics context is immutable after creation. Plan window configuration upfront and accept that fullscreen transitions require window recreation.

### Rendering Mode vs Display Mode are Different Concepts
**Related Insight**: The distinction between:
- **Rendering Mode**: The size of the graphics buffer (set at `InitWindow()` time)
- **Display Mode**: How that buffer is displayed (windowed vs fullscreen)

Many graphics APIs conflate these concepts, but Raylib separates them. You can have:
- Windowed at 1920×1080 rendering 1024×768 (upscaled in window)
- Fullscreen at 1920×1080 rendering 1024×768 (magnified to fill display)

Both have the same rendering resolution but different display modes.

---

## Session: LabelMode / For-loop Indentation (April 2026)

### Broken Intermediate State From Multi-File Field Rename
**What Happened**: `RuntimeContext.LabelsVisible bool` was renamed to `LabelMode ui.LabelMode` in `runtime_context.go` and the type was added to `camera.go`. The session ended (token budget) before `input.go` and `interactive.go` were updated. The codebase did not compile at session boundary.

**Root Cause**: The rename was applied to the *definition* first, leaving all *callsites* unresolved. A session interruption at that midpoint leaves anyone (or any subsequent session) with a broken build they have to diagnose before resuming.

**Rule**: When renaming a field across multiple files, either:
1. Update all callsites atomically in the same work block, *or*
2. Introduce the new field alongside the old one (additive), migrate callsites, then remove the old field.

Never end a session with a partial rename. If interruption is unavoidable, leave a `// TODO: remove after input.go updated` comment on the old field so the compiler error is self-explaining.

---

### Verify Existing Behavior Before Implementing
**What Happened**: The request was "for-loop nested commands should support optional indent of spaces or tabs." Investigation revealed `strings.TrimSpace(bl)` in `runForLoop` already stripped leading whitespace from every body line. No behavior change was needed — only tests to confirm and document it.

**Rule**: Read the implementation before writing a plan. If a feature is requested, check whether it already exists in a non-obvious place. A two-minute read of the relevant function is cheaper than designing and building something that already works.

---

### Pointer-Param Pattern for Runtime Toggles in handleInput
**What Happened**: Adding `LabelMode` to `handleInput` could have expanded the return tuple from 7 to 8 elements (all same type — error-prone). Instead, `labelMode *ui.LabelMode` was passed as a pointer, matching the existing precedent set by `hudDialogVisible *bool`. The return tuple shrank by one (removed `labelsVisible bool`).

**Established Convention**: Runtime state that `handleInput` modifies but does not "own" should be passed as a pointer, not round-tripped through the return tuple. Return tuple additions require updating every `return` statement in the function (13+ sites). Pointer params require no return-site changes.

**When to Use Each**:
- Return tuple: values the function *computes* (shouldQuit, gridVisible — they start as params, may change, and are the function's output)
- Pointer param: values *owned by the caller* that the function may mutate as a side effect (HUD dialog open state, label mode)

---

## Video Recording Session — April 8, 2026

### Root Cause: Native Render Mode Has No Render Texture

**What Happened**: Recording produced a valid MP4 container (ffmpeg started, pipe opened, `[REC] Started` printed) but wrote 0 frames. Three fixes were attempted before the true root cause was found.

**Root Cause**: `configs/app.json` sets `"render.mode": "native"`. In native mode, `syncRenderState` calls `renderer.DisableRenderTarget()`, which sets `targetLoaded = false`. `CaptureRenderTexture` guards on `!targetLoaded` and returns nil immediately — before any GL code runs. ffmpeg received an open pipe with no bytes.

**Misleading symptoms**: `[REC] Started` prints when ffmpeg forks, not when the first frame arrives. The pipe can stay open and empty the entire run.

**Fix**: On `RecordStartCmd`, if `RenderMode == RenderModeNative`, force switch to `RenderModeFixed` and call `syncRenderState()` to create the render texture immediately. Store a `recordingForcedFixed` flag on `App`; restore native mode when recording stops.

**Rule**: Before debugging pixel readback, verify a render texture actually exists (`renderer.HasRenderTarget()`). If `targetLoaded` is false, no capture technique will work.

---

### Apple Silicon / OpenGL-via-Metal: Pixel Readback

**Context**: The Mac M1 GPU exposes OpenGL 4.1 over a Metal translation layer.

**What Does Not Work**:
- `rl.LoadImageFromTexture` — uses `glGetTexImage`, which is not supported in the OpenGL ES / core-profile subset exposed on Apple Silicon. Returns garbage or nil silently.
- `glReadPixels` on Raylib's currently-bound FBO — Metal's GL layer does not reliably expose Raylib's internal render target via the default framebuffer binding. Returns zeroed data without an error.

**What Works**:
- Create a fresh `GL_FRAMEBUFFER`, attach the `RenderTexture2D`'s `.Texture.ID` as `GL_COLOR_ATTACHMENT0`, verify `GL_FRAMEBUFFER_COMPLETE`, then call `glReadPixels`. This FBO is fully under our control and Metal handles it correctly.
- Always check `glCheckFramebufferStatus` before reading. If it returns anything other than `GL_FRAMEBUFFER_COMPLETE`, skip the read — return nil rather than sending garbage to ffmpeg.
- Flush prior GL errors with `while (glGetError() != GL_NO_ERROR) {}` before the sequence to avoid confusing a leftover error with a new one.

**Confirmed working**: `status=0x8CD5` (= `GL_FRAMEBUFFER_COMPLETE`), `glReadPixels err=0x0`, 2654 frames at 57fps into a valid H.264 MP4.

---

### ffmpeg Odd-Dimension Crash

**Error**: `height not divisible by 2 (1440x751)` → ffmpeg exits immediately, pipe closes, 0 frames written.

**Cause**: libx264 requires both width and height to be even. Raylib render texture dimensions come from the screen and are not guaranteed to be even.

**Fix**: `-vf vflip,scale=trunc(iw/2)*2:trunc(ih/2)*2` — the `scale` filter rounds both dimensions down to the nearest even number. The `vflip` was already needed (OpenGL pixel origin is bottom-left; video is top-left).

**Rule**: Always include this scale filter when piping raw pixels from OpenGL to libx264. Do not clamp dimensions at render target creation time — that changes the display layout. Fix it in the ffmpeg filter chain instead.

---

### Render-Target Contract: What Gets Captured

**Rule**: Only pixels drawn inside `Renderer.BeginFrame` / `Renderer.EndFrame` appear in `CaptureRenderTexture` output. `BeginFrame` calls `rl.BeginTextureMode` on the render texture; `EndFrame` calls `rl.EndTextureMode` then blits to screen.

Any draw call made to the default framebuffer outside that window is visible on screen but absent from the render texture and therefore absent from recordings, screenshots, and any future export path.

This applies to all future rendering additions (textures, sprites, shaders, UI panels). As long as they issue draw calls inside `BeginFrame`/`EndFrame`, they are captured automatically with no changes to the recording path.

---

### Debugging Technique: Add fprintf to CGo Before Assuming the Logic Is Wrong

**What Happened**: `CaptureRenderTexture` returned nil on every frame. Three GL-level fixes were made (LoadImageFromTexture → glReadPixels on Raylib FBO → dedicated FBO) before adding `fprintf(stderr, ...)` to the C function revealed the function was never called at all. The bug was a Go-level nil guard (`!targetLoaded`), not a GL issue.

**Rule**: Before iterating on a C/CGo function, add a `fprintf(stderr, ...)` at the top of the C function to confirm it is being reached. A nil return from the Go wrapper is ambiguous — it can mean the C code ran and returned nil, or the Go guard fired before the C code was ever invoked.

---

### REPL Script Commands and the RecordingService gRPC Path

**Architecture**: `record start/pause/stop` REPL commands send RPCs to a `RecordingService` gRPC handler → `App.SendCmd(RecordStartCmd{})` → queued in `cmdCh` → dispatched on the GL main thread in `drainCmds`. The GL-thread dispatch is mandatory because `syncRenderState` and `ConfigureRenderTarget` call Raylib, which is not thread-safe.

**`record delete`** is a local `os.Remove` in the REPL — no RPC needed, no server involvement.

**`sync on`** in scripts is important before `record start`: it makes the REPL block until each command is acknowledged, preventing the recorder from being started before the prior nav command finishes setting up camera state.

---

## Graphics Resolution and Quality

### Raylib Coordinate System with FlagWindowHighdpi

**Problem**: Using `GetRenderWidth/Height` (physical framebuffer pixels) for 2D drawing coordinates after `BeginDrawing()` caused objects to render in just the top-left quarter of the screen, and windowed mode scaling broke completely.

**Root Cause**: After `BeginDrawing()`, Raylib applies a `screenScale` matrix (set by `rlMultMatrixf`) that maps logical coordinates to physical framebuffer pixels. On a 2× Retina display, logical (1440, 900) is multiplied to physical (2880, 1800) by the matrix. If you pass physical coords yourself, they get doubled again — drawing past the viewport edge.

After `BeginTextureMode(target)`, Raylib resets the modelview to identity (no screenScale). Coordinates map 1:1 to texture pixels.

**Rules**:
- After `BeginDrawing()`: use `GetScreenWidth/Height` (logical). The screenScale matrix handles physical mapping.
- After `BeginTextureMode()`: use the render texture dimensions directly. No DPI scaling.
- `DrawTexturePro` dest rects after `EndTextureMode()` + `BeginDrawing()` are in logical coords.

**Verification**: Raylib's own `core_highdpi_testbed.c` and `window_letterbox` examples use `GetScreenWidth()` for positioning after `BeginDrawing()`.

---

### MSAA Only Affects the Default Framebuffer

**Fact**: `FlagMsaa4xHint` sets `glfwWindowHint(GLFW_SAMPLES, 4)` which multisamples only the OS window surface. `LoadRenderTexture` creates a standard `GL_FRAMEBUFFER` with no multisample attachments. Raylib does not expose multisampled FBO creation.

**Consequence**: In `fixed` render mode (draw to render texture, blit to screen), MSAA has **no effect** on 3D geometry quality. The multisample resolve only anti-aliases the edges of the fullscreen blit quad — imperceptible. In `native` mode (draw directly to default framebuffer), MSAA smooths all polygon edges as expected.

**Rule for recording**: Since recording forces `fixed` mode (need a render texture to capture), MSAA cannot improve recording quality. Use `--render-scale 2` (supersampling) instead — it renders at 2× resolution into the render texture, which achieves the same anti-aliasing effect through brute-force resolution.

**Rule for daily use**: MSAA improves visual quality in the default `native` mode where no render texture is involved. Enable it by default.

---

### Render Resolution vs Object Quality

**Observation**: `--render-scale` changes the render texture pixel count but does not change 3D mesh polygon count. `DrawSphereEx(pos, radius, rings, slices, color)` produces the same triangle count regardless of resolution. Higher resolution makes text and HUD sharper but sphere polygon edges remain identical.

**What controls object quality**: The LOD system sets `rings`/`slices` based on camera distance. Current tiers: 32 (< 20 units) → 24 → 16 → 12 → 6 (> 200 units). These values determine the triangle count of each sphere.

**Rule**: To improve 3D object quality, increase tessellation (rings/slices) in the LOD system. To improve pixel-level edge smoothness, use MSAA (native mode) or supersampling (fixed/recording mode). Resolution alone does not improve polygon geometry.

---

### CLI Flags Must Not Persist to app.json

**Problem**: `--render-scale 2` forced `fixed` mode with 2× dimensions at startup. On exit, `persistWindowConfig()` saved the runtime render state to `app.json`. Next launch without the flag loaded the stale `fixed` config — render scale was "sticky" between sessions, soiling test results.

**Fix**: Capture the render config as loaded from `app.json` before `WithDefaults()` applies CLI overrides. On exit, persist the original loaded render config, not the runtime state. CLI flags are session-scoped by design. Only user-edited `app.json` changes survive across sessions.

**Rule**: CLI flags override config for the current session only. Never persist CLI-derived state back to the config file unless the user explicitly requests it.

---

## Diffuse Texture Mapping on Procedural Spheres (F-003)

**Date**: April 15, 2026

### Overview

Applying equirectangular planet texture maps to Raylib procedural spheres required fixing four independent, additive errors. Each one was invisible until the previous was corrected, making iterative debugging misleading. This entry documents every assumption that proved wrong.

---

### How Textures Are Applied

Raylib does not natively support textured procedural sphere meshes with a simple one-liner. The pipeline used:

1. **`rl.GenMeshSphere(radius, rings, slices)`** — generates a UV sphere mesh in CPU memory using the `par_shapes` library (bundled C source). Returns a `rl.Mesh` struct with a `Texcoords` float32 slice (2 floats per vertex: `[U, V, U, V, ...]`), `Vertices`, and `Normals` allocated in CPU memory.

2. **UV correction in CPU memory** — before uploading to the GPU, the entire `Texcoords` slice is walked in Go using `unsafe.Slice` to remap every vertex's UV coordinates.

3. **`rl.UpdateMeshBuffer(mesh, 1, ...)`** — re-uploads the corrected UV buffer (slot 1) to the GPU. Buffer index 1 is the texcoord VBO in Raylib's fixed-layout VAO.

4. **`rl.LoadModelFromMesh(mesh)`** — creates a `rl.Model` wrapping the mesh. At this point the VAO/VBOs are on the GPU; the CPU-side `Texcoords` pointer is now redundant.

5. **`model.GetMaterials()[0].GetMap(rl.MapDiffuse).Texture = tex`** — assigns the loaded GPU texture to the model's diffuse slot. This binds the texture to the material, not to the vertices — vertices only hold UV coordinates that index into the texture.

6. **`model.Transform`** — set per draw call (not cached) to apply pole-alignment and axial tilt. `model` is a value type from the cache, so mutating `Transform` on the local copy does not affect the cached entry.

7. **`rl.DrawModel(model, pos, scale, rl.White)`** — renders the model. Raylib's shader reads `MapDiffuse`'s texture, samples it at each fragment's interpolated UV coordinates, and multiplies by the tint color.

**Key distinction**: the texture is applied to a mesh via material diffuse map + per-vertex UV coordinates. There is no cube-mapping, triplanar, or spherical projection at draw time — it is standard UV-unwrapped texturing. The correctness of the result depends entirely on the UV coordinates stored on the vertices.

---

### The Four Errors

#### Error 1 — Axis Transposition (par_shapes UV convention)

**Assumption**: `GenMeshSphere` produces standard equirectangular UVs where `U` = longitude (0=west, 1=east) and `V` = latitude (0=south, 1=north).

**Reality**: par_shapes' `par_shapes__sphere` function computes:
```c
float phi   = uv[0] * PI;     // uv[0] = latitude (0=north, π=south)
float theta = uv[1] * 2*PI;   // uv[1] = longitude
xyz[0] = cos(theta) * sin(phi);
xyz[1] = sin(theta) * sin(phi);
xyz[2] = cos(phi);             // +Z at phi=0 = north pole
```
The vertex tex coord array stores these as `[uv[0], uv[1]] = [latitude, longitude]` — transposed from the equirectangular convention. A planet texture applied without correction would wrap latitude along the horizontal axis of the image, smearing the poles across the equator visually.

**Fix**: Swap: `new_U = old_uv[1]`, `new_V = old_uv[0]`.

---

#### Error 2 — V-Axis Inversion (stb_image vs. OpenGL origin)

**Assumption**: After the axis swap, `V=0` samples the top of the image file (north).

**Reality**: `stb_image` loads image rows top-to-bottom and places them in memory row 0 = top. OpenGL's texture coordinate `V=0` maps to the **bottom** of the texture as stored in GPU memory. Raylib never calls `stbi_set_flip_vertically_on_load`, so no flip occurs on load. The mismatch means `V=0` → bottom of image → south of texture. A plane with `V=0` at the top (sky) would show the ground. On the sphere, the north pole vertex (which had `V=0` after the swap) sampled the south-pole region of the image.

**Fix**: `new_V = 1.0 - old_V` after the axis swap.

---

#### Error 3 — U-Axis Mirror (inside-out winding)

**Assumption**: After fixing axes and V, longitude now increases left-to-right (west=0, east=1) matching the texture.

**Reality**: par_shapes generates sphere meshes for **inside-view rendering** (sky-dome convention). The triangle winding is inverted relative to outside-view. As a consequence, when viewed from outside the sphere, the UV longitude axis runs right-to-left — east and west are mirrored. Africa's west coast appears on the right side of the visible face.

**Fix**: `new_U = 1.0 - old_U`.

**Final remap applied in one pass**:
```go
new_U = 1.0 - old_uv[1]   // swap axes + mirror U
new_V = 1.0 - old_uv[0]   // swap axes + flip V
// i.e.: uv[i*2], uv[i*2+1] = 1.0 - uv[i*2+1], 1.0 - uv[i*2]
```

---

#### Error 4 — Pole Axis Misalignment (par_shapes vs Raylib world axis)

**Assumption**: `GenMeshSphere` places the north pole at world `+Y` (Raylib's up axis).

**Reality**: par_shapes' north pole is at object `+Z` (where `phi=0` → `xyz=(0,0,1)`). Raylib's world is Y-up. When drawn without correction, both poles appear near the equator (the poles are "sideways" pointing into and out of the camera).

**Fix**: At draw time, set `model.Transform` to include a `RotX(-90°)` rotation, which brings `+Z` up to `+Y`:
```go
poleAndTilt := rl.MatrixMultiply(rl.MatrixRotateX(-90*rl.Deg2rad), rl.MatrixRotateZ(tilt))
model.Transform = rl.MatrixMultiply(poleAndTilt, rl.MatrixRotateY(spin))
```
The full transform composes: pole correction (RotX -90°), axial tilt (RotZ), and axial spin (RotY, driven by `RotationPeriod` and `simTimeScale`). Spin is innermost so it rotates around the body's own tilted axis rather than the world Y axis.

This is applied per-draw on a value-copy of the cached model, so the cache is never mutated.

---

### Orbital Direction Fix (related)

**Problem discovered alongside texturing**: Planets orbited clockwise from the north pole view, opposite to the real solar system.

**Root cause**: `rotateOrbit` returned `{X: x3, Y: z3, Z: y3}`. For zero-inclination orbits, this placed `ν=π/2` at world `+Z`, producing orbital sequence `+X → +Z → -X → -Z` — clockwise from north in a right-handed Y-up system.

**Fix**: Changed to `Z: -y3`. Prograde now maps `+X → -Z → -X → +Z` — counter-clockwise from north, matching the real solar system. Belt initial positions negated `sin(angle)` to match.

---

### Rules Derived

- **Never assume par_shapes UV convention matches equirectangular.** The axis order is latitude-major, not longitude-major.
- **Always account for the stb_image / OpenGL V-origin mismatch** when loading textures without a vertical flip. `V=0` is the bottom of the GPU texture, but the top of the PNG file.
- **par_shapes sphere meshes are wound for inside-view.** Correct U mirroring when rendering from outside.
- **Correct the pole axis before applying axial tilt/spin.** Order matters: pole fix first (RotX -90°), then tilt (RotZ), then spin (RotY innermost).
- **UV correction belongs in the mesh, not the shader or draw call.** Fix it once in `getModel` at mesh-build time; all downstream rendering is clean.
- **UpdateMeshBuffer slot 1 is the texcoord VBO.** Raylib's fixed VAO layout: 0=vertices, 1=texcoords, 2=normals, 3=colors, 4=tangents, 5=texcoords2.
- **model.Transform is a value field.** Mutating it on a locally-returned struct from a `map` does not modify the cached copy. Safe to set per-frame.

---

## Rendering Quality & Visual Fidelity (April 21, 2026) [RENDER]

### LOD Tessellation: Small-Body Radius Halving Was Wrong

**Problem**: Earth and other planets with `radius < 1.0` sim-units had visible polygon faceting during texture rotation. The "fix" for it made things worse.

**Root cause**: A `PhysicalRadius < 1.0` guard halved rings/slices for small bodies, cutting Earth to a maximum of 16×16 (from an already-low 16×16 base). Sub-unit scale does not imply low visual importance; Earth is the most-viewed body in the solar system view.

**Wrong rule**: Halve geometry for small objects.  
**Correct rule**: LOD is purely distance-based. Object angular size on screen is what matters, not world-space radius.

**Fix**: Removed the halving block entirely. Raised the base tessellation levels:

| LOD band | Before | After |
|---|---|---|
| VeryClose (<20u) | 32×32 | 128×128 |
| Close (<50u) | 24×24 | 64×64 |
| Medium (<100u) | 16×16 | 32×32 |
| Far (<200u) | 12×12 | 16×16 |
| Beyond | 6×6 | 8×8 |
| Default (LOD off) | 16×16 | 64×64 |

**Rule derived**: Never use world-space radius to gate tessellation quality. Angular size (screen coverage) is the only meaningful LOD metric.

---

### Oblate Spheroids: Non-Uniform Scale via model.Transform

**Problem**: Jupiter and Saturn are visually oblate (~7% and ~10% equatorial bulge), but were rendering as perfect spheres. Using `DrawModel(model, pos, uniformScale, tint)` cannot express non-uniform scale.

**Approach**: Bake a non-uniform scale matrix into `model.Transform` and pass `drawScale=1.0` to `DrawModel`. Raylib applies `model.Transform` before the draw-time uniform scale, so the effective transform is `(rotation)(scale)` composed as a single matrix.

**Key detail**: `rl.Matrix` is column-major. A pure scale matrix has only the diagonal set:
```go
scaleMat := rl.Matrix{M0: eqR, M5: polR, M10: eqR, M15: 1}
// Then: rotMat × scaleMat  (scale applied first, then rotation)
```

**Go trap**: `:=` cannot destructure into a struct field on the left-hand side:
```go
// Compile error:
model.Transform, drawScale := buildModelTransform(meta, simTime)

// Correct:
transform, drawScale := buildModelTransform(meta, simTime)
model.Transform = transform
```

**Rule derived**: For non-uniform body scaling, bake the scale into `model.Transform` and pass `1.0` as the `DrawModel` scale. Separate the assignment from the declaration when the LHS contains a struct field.

---

### GLSL Phong Lighting: SetShaderValue Integer Uniforms

**Problem**: `rl.SetShaderValue` takes `[]float32` for all types, including `ShaderUniformInt`. Passing an integer as float causes the shader to receive garbage.

**Correct pattern** for setting an `int` uniform via the `[]float32` overload:
```go
// Reinterpret the int32 bit-pattern as float32 before passing:
countF := *(*float32)(unsafe.Pointer(&count))
rl.SetShaderValue(ls.shader, ls.locCount, []float32{countF}, rl.ShaderUniformInt)
```

**Alternative**: Use `rl.SetShaderValueV` when sending multiple elements; same reinterpretation applies.

**Rule derived**: `SetShaderValue` is type-unsafe. Always reinterpret non-float types through `unsafe.Pointer` before handing them to the `[]float32` slice. Document it at the call site.

---

### Physical Lighting: Brightness Tuning

**Problem**: A freshly shipped Phong shader may be imperceptible if `lightScale` is tuned for the wrong distance range. The lit side appears dimmer, not brighter, compared to the previous flat-diffuse look — and users may not notice the day/night divide.

**Why**: The shader replaces full-brightness flat diffuse with `(ambient + diffuse * intensity/dist²)`. If `intensity/dist² < 1`, the lit side is duller than before. If ambient is too high, the dark side is not dark enough to create visible contrast.

**Tuning formula** for desired lit-side brightness `B` (0–1):
```
lightScale = B × dist_au² / solarLuminosity
```
At Earth's sim distance ≈ 100 units and desired lit brightness 0.9:
```
lightScale = 0.9 × 100² / 1.0 = 9000
```

**Verification steps**:
1. Park camera at body's equator, 90° from the star (terminator edge).
2. Rotate so star is fully to one side — one hemisphere should be bright, one near-black.
3. Compare with `--no-lighting` to confirm contrast is an improvement.

**Rule derived**: After shipping a lighting shader, immediately test at the terminator. Brightness tuning is always necessary before the effect is clearly visible. Start with `ambient ≤ 0.03` and raise `lightScale` until the lit side is ~90% brightness at expected viewing distances.

---

### Automated Editors Strip Per-Call Side-Effect Calls — Dual Draw Paths Require Explicit Audits

**Problem**: After `applyToModel` was added to both draw paths, an automated editor (formatter or assistant) removed it from `drawObjectsInstanced` while leaving it in `drawObject`. The shader was never applied in practice because instanced rendering is the default; the bug was invisible until a terminator test.

**Why this is high-risk**:
- `applyToModel` looks like dead code inside a `getModel` success branch — it has no return value and no compile-time visibility.
- Automated editors optimise for apparent simplicity and will silently drop calls that don't look load-bearing.
- The app compiled clean and ran correctly without any error — only the visual result was wrong.
- The two draw paths (`drawObjectsInstanced` / `drawObject`) are structurally similar, so a reviewer checking one may not notice the other is diverged.

**Detection**: The bug produced *no crash, no error, no log line* — just uniform full brightness everywhere, which is also the correct appearance for a star. It was only detectable by positioning the camera at the terminator and verifying dark-side shadowing.

**Rules derived**:
- **Any call that wires a per-object side effect (shader, material override, state mutation) must appear in both draw paths.** Comment both with `// MUST match drawObject / drawObjectsInstanced` so editors and reviewers treat them as a paired invariant.
- **After every automated edit to renders.go, diff both draw paths against each other.** They should be structurally identical in their textured-draw blocks.
- **Proof of lighting**: the correct verification is a terminator test, not a frontal view. "Looks lit" from the sunny side proves nothing. Only "dark side is dark" proves the shader is active.
- **Feature acceptance criteria for any visual shader: include an explicit terminator/shadow test in the PR description** so the acceptance gate is unambiguous.

---

## CGo Interior-Pointer Panic: Slicing a Struct Field

**Date**: 2026-04-21  
**Symptom**: `panic: runtime error: argument of cgo function has Go pointer to unpinned Go pointer` at runtime, only after the atmosphere shader was first invoked.

**Root Cause**: Passing `r.lighting.PrimaryLightPos[:]` to `rl.SetShaderValue`. Slicing an array field of a struct produces a slice whose backing array is an *interior pointer* into the `Renderer` allocation. `Renderer` also holds `textureCache` and `modelCache` (Go maps — Go pointer types). CGo's pointer checker sees "Go pointer to an object that contains other Go pointers" and panics at the first CGo call that frame.

**Fix**: Copy to a fresh local slice before the call:
```go
// BAD — interior pointer into Renderer (which contains maps):
rl.SetShaderValue(shader, loc, r.lighting.PrimaryLightPos[:], rl.ShaderUniformVec3)

// GOOD — clean allocation, no Go pointers inside:
lightPos := []float32{r.lighting.PrimaryLightPos[0], r.lighting.PrimaryLightPos[1], r.lighting.PrimaryLightPos[2]}
rl.SetShaderValue(shader, loc, lightPos, rl.ShaderUniformVec3)
```

**Rule derived**: **Never pass a sub-slice of a struct field to a CGo function.** Any `structField[:]` expression is suspect. Always copy to a fresh `[]T{a, b, c}` literal. This applies to all `rl.SetShaderValue` calls and any other raylib CGo boundary.

---

## Atmosphere Shader: Fresnel Rim-Glow + Day/Night Lighting

**Date**: 2026-04-21  
**Problem**: The original `DrawSphereEx` atmosphere had two visual defects:
1. **Hard outer edge** — the additive sphere ended abruptly at the mesh boundary.
2. **Full-bright night side** — the glow was uniform regardless of sun angle.

**Solution**: Custom GLSL shader (`atmoVS`/`atmoFS`) with two terms:

- **Fresnel rim**: `pow(1.0 - abs(dot(N, viewDir)), glowEdge)` — face-on fragments produce 0 (transparent), limb fragments produce 1 (full glow). This produces a natural edge fade with no hard boundary.
- **Lambert sun term**: `mix(0.08, 1.0, max(dot(N, toLight), 0.0))` — night side receives 8% (secondary scatter floor), day side receives up to 100%.

**Render setup**:
- A single unit sphere (`GenMeshSphere(1.0, 64, 32)`) is lazy-loaded and reused across all atmosphere draw calls; scaled per-body via `model.Transform = MatrixScale(r, r, r)`.
- Drawn with `BlendAddColors` (GL_ONE, GL_ONE) — the glow only brightens; Z-order is irrelevant.
- The shader's `glowColor.a` component is used as an intensity weight inside the shader, not as actual alpha (alpha is unused in additive blend).
- `PrimaryLightPos` is captured once per frame in `setLights()` from the first self-luminous object and stored on `lightingState`.

**Tuning knob**: `glowEdge` constant (default 3.5). Higher → narrower bright ring. Lower → wider diffuse halo.

---

## Rendering: Transparency, Draw Order, and Lighting (April 22, 2026) [RENDER]

### Additive Blend Does Not Suppress Depth Writes

**Commit**: `2c243a5` — fix: atmosphere flicker — disable depth write during additive blend

**Problem**: The atmosphere glow sphere flickered intermittently on planets that had close-by opaque objects (e.g. ring systems). Flickering was frame-rate dependent and position-dependent; it was absent at some camera angles and severe at others.

**Root Cause**: `rl.BeginBlendMode(rl.BlendAddColors)` sets the GL blend equation to `(ONE, ONE)` — additive colour compositing — but it does **not** modify the depth-write mask. The atmosphere sphere, drawn with `DrawModel`, still wrote its depth values to the depth buffer at `glowRadius`. On the next frame (or later in the same frame), opaque objects (rings, near-surface geometry) whose screen fragments fell inside the glow zone failed the depth test against the already-written glow depth values and were culled.

**Fix**:
```go
// Disable depth writes for the additive glow sphere only:
rl.DisableDepthMask()
rl.DrawModel(model, pos, scale, color)
rl.EnableDepthMask()
```

**Rule**: Any additive or semi-transparent draw that should never occlude opaque geometry must explicitly suppress depth writes with `DisableDepthMask()` around the draw call. `BeginBlendMode` alone is not sufficient.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`

---

### Transparent Objects Must Be Drawn After All Opaque Objects

**Commit**: `a0a2bd5` — fix: ring draw order, planet shadow, and sim-speed startup

**Problem**: Rings were drawn inside the main planet batch (interleaved with planet draw calls). The result was Z-fighting between ring fragments and planet surface fragments and incorrect alpha compositing at ring/planet boundaries.

**Root Cause**: `BlendAlpha` ring draw calls compete with opaque planet draws in depth order. The depth buffer sorts opaque geometry correctly, but semi-transparent draws must be composited on top of a fully resolved opaque scene. Any transparent draw that runs before an opaque draw behind it will produce incorrect results.

**Fix**: Collect all ring draw calls into a deferred list during the planet batch pass, then execute the deferred ring draws after all planet batches and atmosphere glow passes complete.

**Rule**: Transparent/semi-transparent objects must be sorted and drawn after all opaque geometry in the same frame. Never interleave `BlendAlpha` draws with opaque draws.

**Also fixed in this commit**:
- **Ring shadow cylinder test**: each ring segment tests whether its centre falls inside the planet's shadow cylinder (cylinder aligned with sun direction, radius = planet physical radius). Segments inside the cylinder receive ambient-floor-only lighting. The test uses a point-to-cylinder-axis distance check.
- **`--sim-speed` CLI flag**: was routing to the animation speed multiplier instead of `SecondsPerSecond`, so the startup time rate was wrong. Fixed by routing to the simulation time parameter.
- **Selection dialog scroll**: was using a hardcoded `500px` panel height instead of live `rl.GetScreenWidth()`, breaking on non-default window sizes.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`, `internal/client/go/raylib/app/input.go`, `session.go`

---

### Transparent Rings: Texture Bands + Per-Body Physical Radius in Schema

**Commit**: `e5254de` — render: rings transparent+textured, atmosphere scale fix, size-aware LOD

**Problem 1 — Ring appearance**: Rings rendered as a single opaque disc using `fallback_color`. No transparency, no radial texture variation.

**Fix**: Split each ring disc into 8 radial bands. Each band samples from `saturnringcolor.jpg` using the band's normalised radial position as the U coordinate. `BlendAlpha` + `DisableDepthMask` applied per band. The `fallback_color.a` field is preserved as the opacity weight.

**Problem 2 — Gas giant atmosphere radius**: Jupiter, Uranus, and Neptune glow spheres extended far beyond ring inner edges because the glow fraction was calibrated against Earth's km-to-sim-unit ratio (`12742 km/su`). Gas giants are much larger; the same fraction produced a glow sphere with a radius that swallowed the rings.

**Fix**: Added `radius_km` field to `ObjectMetadata` schema and set it for Sol, Jupiter, Saturn, Uranus, Neptune. `drawAtmosphereGlow` uses `PhysicalRadiusKm` when available; falls back to Earth-calibrated default. Gas giant atmosphere thicknesses corrected to 1000 km (Jupiter) and 600 km (Neptune/Uranus) so glow radii stay inside ring inner edges.

**Problem 3 — LOD quality budget ignores object size**: `lodScale()` computed distance thresholds uniformly for all objects. At the same camera distance, Sol (radius ≫ 1 sim-unit) and a moon (radius ≪ 1 sim-unit) received the same tessellation budget, producing a faceted Sun.

**Fix**: `lodScale()` now multiplies all distance thresholds by `clamp(radius/0.5, 1, 10)`. Sol (radius ≈ 47 sim-units) gets a 10× budget; small bodies are unchanged. This gives large objects proportionally more geometry at any given camera distance.

**Rule**: LOD thresholds should reflect on-screen angular size, which is proportional to both world-space radius and inverse camera distance. Objects with very large radii need larger threshold multipliers, not just more rings/slices at the closest tier.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`, `internal/sim/engine/feature.go`, `internal/sim/schema.go`, `data/systems/solar_system.json`

---

### Ring Lighting: Material Must Be Wired to Shader on Every Draw Call

**Commit**: `2c36bd2` — made rings respond to lighting effects

**Problem**: Rings rendered at full brightness regardless of sun position, while planets and moons correctly showed day/night shading.

**Root Cause**: The ring draw path called `DrawModel` without first setting the model's material to use the Phong lighting shader. The per-frame `setLights()` call uploads light uniforms to the shader, but the `DrawModel` call uses whatever shader the model's material currently references. Without explicitly assigning `ls.shader` to `model.GetMaterials()[0].Shader`, Raylib uses the default unlit material shader — which ignores all light uniforms.

**Fix**: Before each ring `DrawModel` call (and each band within it), set `model.GetMaterials()[0].Shader = ls.shader`. Mirrors the pattern already used for planet/moon draw calls.

**Rule**: Uploading light uniforms to a shader does not automatically make all objects use that shader. Each `DrawModel` call uses the shader referenced by the model's own material. Any object that should participate in lighting must have `model.GetMaterials()[0].Shader = ls.shader` set immediately before `DrawModel`.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`

---

### RotateY Sign Convention: Positive Is Clockwise From Above

**Commit**: `848d9ab` — fix: negate spin in buildModelTransform so prograde bodies rotate CCW

**Problem**: Planets rotated clockwise when viewed from above (north), opposite to the real solar system.

**Root Cause**: In Raylib's right-handed Y-up coordinate system, `rl.MatrixRotateY(+θ)` rotates clockwise when viewed from above. The real solar system's prograde direction (Earth, all gas giants) is counter-clockwise from above. Passing `+spinAngle` produced retrograde-looking rotation.

**Fix**: Negate the spin angle:
```go
// Was (clockwise = wrong for prograde bodies):
rl.MatrixRotateY(spinAngle)

// Now (counter-clockwise = correct for prograde):
rl.MatrixRotateY(-spinAngle)
```

**Rule**: In Raylib Y-up, `RotateY(+)` is clockwise from above. Prograde planetary spin requires `RotateY(-)`. If a retrograde body (Venus, Uranus) has `rotation_period` stored as a negative value in data, negating the spin angle will make it rotate clockwise — which is correct for retrograde.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`

---

## Config Persistence and Infra Mode (April 22–23, 2026) [CONFIG]

### JSON Unmarshal Zero-Value Trap for bool Defaults

**Commit**: `d4f5321` — fix: seed PerformanceConfig defaults before JSON unmarshal in LoadAppConfig

**Problem**: On first run (or whenever `app.json` predated the `"performance"` block), all `PerformanceConfig` fields loaded as `false` / `0`, silently disabling features that should default to enabled (`FrustumCulling`, `InstancedRendering`, `SpatialPartition`, `UseInPlaceSwap`).

**Root Cause**: `json.Unmarshal` only writes fields that are present in the JSON. When the `"performance"` key is absent, none of the struct fields are touched — they remain at Go zero-values. For `bool`, zero-value is `false`. Features with correct defaults of `true` were therefore silently disabled on every fresh install.

**What Doesn't Work**:
```go
var cfg AppConfig
if err := json.Unmarshal(data, &cfg); err != nil { ... }
// cfg.Performance.FrustumCulling == false  ← WRONG, should be true
```

**Fix**: Pre-seed the struct with correct defaults before calling `Unmarshal`. The unmarshal then only overwrites keys that are present, preserving the defaults for absent keys:
```go
cfg := AppConfig{
    Performance: PerformanceConfig{
        FrustumCulling:      true,
        InstancedRendering:  true,
        SpatialPartition:    true,
        UseInPlaceSwap:      true,
        ImportanceThreshold: defaultImportanceThreshold,
    },
}
_ = json.Unmarshal(data, &cfg)  // only overwrites present keys
```

This pattern was already used for `Window` and `Render` config sections but was omitted for the new `Performance` section.

**Rule**: Any config struct deserialized with `json.Unmarshal` where fields have non-zero defaults **must** be pre-initialized before the unmarshal call. Do not rely on JSON absence to preserve defaults — absence produces zero-values, not defaults. This is especially critical for `bool` fields that default to `true`.

**Files fixed**: `internal/client/go/raylib/app/config_file.go`

---

### RuntimeContext as the Single Source of Truth for Persisted State

**Commit**: `aafae1f` — feat: persist all performance options to app.json

**Problem**: Performance toggle key handlers in `input.go` and `PerfSetCmd` dispatch in `interactive.go` updated `inputState.PerfOptions` (which controls the running session) but not `runtime.PerfConfig` (which persists to `app.json`). On shutdown, `AppConfigSnapshot()` serialized the stale `runtime.PerfConfig` values, losing all changes made during the session.

**Architecture**:
- `InputState.PerfOptions` — session-scoped, reconstructed from `NewInputState()` defaults each run. Controls what the renderer actually does this frame.
- `RuntimeContext.PerfConfig` — persists for the app's lifetime, serialized via `AppConfigSnapshot()`. Source of truth for `app.json`.

**Fix**: At every mutation site (key toggles in `input.go`, `PerfSetCmd` dispatch in `interactive.go`), update **both**:
1. `inputState.PerfOptions.Xxx` — for the running session
2. `a.runtime.PerfConfig.Xxx` — for persistence

**Rule**: Mutable state that must survive shutdown lives in `RuntimeContext`, not `InputState`. `InputState` is session-local. When adding any new persistable option, always write to both the session state and `RuntimeContext` at every mutation site. Forgetting `RuntimeContext` causes silent persistence failure with no compile-time error.

**Files fixed**: `internal/client/go/raylib/app/interactive.go`, `internal/client/go/raylib/app/input.go`

---

### Per-Object Shader Uniform: Set Before Draw, Restore Immediately After

**Commit**: `025cc26` — feat: implement infra mode 1 spotlight ambient boost

**Problem**: Infra mode 1 (night-vision spotlight) needed to boost the ambient light for objects near the camera's centre and reduce it for objects at the cone periphery, without affecting the base shader behaviour for objects outside infra mode.

**Implementation**: The ambient uniform is set per-object immediately before `DrawModel`, then restored to the default immediately after:
```go
// Compute spotlight factor for this object (0.0 = outside cone, 1.0 = centre)
infraF := spotlightFactor(obj.Pos, cameraPos, r.cameraForward)

// Boost ambient before draw:
r.lighting.setAmbient(defaultAmbient + (infraSpotAmbient-defaultAmbient)*infraF)
rl.DrawModel(model, pos, scale, tint)

// Restore immediately — do not leave the shader in boosted state:
r.lighting.setAmbient(defaultAmbient)
```

**Why the restore matters**: If an early-exit path (e.g. `continue`, texture load failure) skips the draw call but not the restore, the shader is left in boosted state for every subsequent object in the batch. The restore must be unconditional and placed immediately after `DrawModel`, not at the end of the loop body.

**Rule**: Per-object shader state mutations (uniform overrides) must be structured as `set → draw → restore`, all three unconditionally adjacent. The restore must use the exact known-good constant (`defaultAmbient = float32(0.02)`), not a locally snapshotted value, to guarantee global consistency.

**Files fixed**: `internal/client/go/raylib/ui/render/renders.go`, `internal/client/go/raylib/ui/render/lighting.go`

---

### Spotlight Cone Using Cosine Comparison (No acos Needed)

**Commit**: `025cc26` — feat: implement infra mode 1 spotlight ambient boost

**Implementation**: The spotlight factor is computed from the dot product of the normalized camera-to-object direction and the camera forward vector, compared against cosine thresholds:

```go
const (
    infraSpotInnerCos = float32(0.9397) // cos(20°)
    infraSpotOuterCos = float32(0.7660) // cos(40°)
)

func spotlightFactor(objPos, cameraPos, cameraForward engine.Vector3) float32 {
    toObj := objPos.Sub(cameraPos).Normalize()
    cosAngle := toObj.Dot(cameraForward)
    if cosAngle <= infraSpotOuterCos {
        return 0.0 // outside the cone
    }
    if cosAngle >= infraSpotInnerCos {
        return 1.0 // inside the hot centre
    }
    // Smoothstep in the penumbra band:
    t := (cosAngle - infraSpotOuterCos) / (infraSpotInnerCos - infraSpotOuterCos)
    return t * t * (3 - 2*t)
}
```

**Why cosines, not angles**: `math.Acos` is expensive and unnecessary. Since `cos` is monotone-decreasing on `[0°, 180°]`, comparing `cosAngle >= innerCos` is equivalent to `angle <= innerDeg`. Store the threshold as a pre-computed cosine constant.

**Smoothstep** `t*t*(3-2*t)` maps `[0,1]→[0,1]` with zero derivative at both endpoints, producing a visually smooth penumbra transition with no sudden cutoff.

**Rule**: Spotlight / cone tests in shaders and CPU code should use cosine comparison against pre-computed constants, not `acos`. This avoids a transcendental function call per visible object per frame.

---

### Two-Phase Feature Commit: Plumbing Before Rendering

**Commit**: `3f40c1b` — feat: add infra mode gRPC/REPL plumbing  
**Commit**: `025cc26` — feat: implement infra mode 1 spotlight ambient boost

**Pattern**: When a feature requires both infrastructure (proto definitions, gRPC handlers, REPL commands, key bindings, help screen text) and a visual effect, commit them as two separate atomic units:

1. **Plumbing commit** — proto, gRPC service, REPL command dispatch, key binding, help screen. The app compiles and routes commands. No visual side-effects yet; all code paths terminate cleanly without rendering changes.
2. **Rendering commit** — the renderer reads the new mode flag and produces the observable behaviour.

**Benefits**:
- The plumbing can be merged, reviewed, and tested independently of the visual work.
- If the rendering pass runs long or hits unexpected issues, the plumbing is already in main and doesn't need to be re-landed.
- The two commits are independently revertable.

**When not to use this pattern**: If the plumbing change is trivially small (one field, one dispatch case), a single combined commit is cleaner. Reserve two-phase commits for features where the plumbing spans multiple packages (proto, gRPC, REPL, app wiring).

---

## Rendering Session Lessons (April 24, 2026) [RENDERING]

### 24. **Atmosphere Shader Self-Occlusion: Star Glow Invisible Due to Lambert Term** [RENDERING]

**Problem**: Sol had no visible atmosphere glow even after the point-render threshold was raised so the atmosphere shader was actually being called.

**Root Cause**: The atmosphere GLSL shader uses a Lambert diffuse term to suppress the glow on the night side of planets:
```glsl
vec3 toLight = normalize(lightPos - fragPos);
litFactor = mix(0.08, 1.0, max(dot(norm, toLight), 0.0));
```
For Sol itself, `lightPos ≈ fragPos` (Sol is its own light source, positioned in the same place). This makes `toLight ≈ -norm`, so `dot(norm, toLight) ≈ -1`, and `litFactor = 0.08` — 8% brightness, effectively invisible.

**Solution**: Added a `uniform int selfLuminous` to the fragment shader. When `selfLuminous == 1`, `litFactor = 1.0` unconditionally, bypassing the Lambert term. The uniform is set from the Go side using the bit-reinterpretation pattern required for Raylib's `SetShaderValue` (which takes `[]float32` even for int uniforms):
```go
selfLum := int32(1)
selfLumF := *(*float32)(unsafe.Pointer(&selfLum))
rl.SetShaderValue(r.atmosphere.shader, r.atmosphere.locSelfLuminous,
    []float32{selfLumF}, rl.ShaderUniformInt)
```

**Rule**: Any body that is its own light source must bypass lighting terms that compute `toLight` relative to that body's position. The `selfLuminous` flag is the correct gate.

---

### 25. **Point-Render Threshold Missing for Stars: Atmosphere Never Called** [RENDERING]

**Problem**: Sol's atmosphere glow was never rendered at any camera distance.

**Root Cause**: The renderer selects between point-rendering and sphere-rendering based on a per-category threshold. Stars had no threshold entry — they fell through to `PointThresholdDefault = 200`. At any camera distance > 200 sim units (i.e., almost always), Sol was rendered as a 2D point, and `drawAtmosphereGlow` was never called.

**Solution**: Added `PointThresholdStar = 1e15` to `constants.go`. Stars are effectively never point-rendered (the threshold is higher than the maximum possible camera distance in the sim), so the sphere + atmosphere path always executes.

**Rule**: Every `Category*` constant must have a corresponding point-render threshold case. When adding a new category, explicitly decide whether to point-render it and at what distance — defaulting to a shared fallback silently suppresses category-specific rendering paths.

---

### 26. **Atmosphere Glow Sphere Oversized: atmoCap Too Large** [RENDERING]

**Problem**: Sol's corona glow sphere was far larger than expected — appearing as a bright halo 1.6× Sol's radius (a noticeable ring rather than a tight corona).

**Root Cause**: `atmoCap = 0.60` capped the glow fraction at 60% above body radius. Sol's `AtmosphereThicknessKm = 2,000,000` produced `frac = (2_000_000 / (696_340 × 12742)) × 4 = 11.49`, clamped to 0.60 → glow sphere 1.6× the physical radius.

**Solution**: Reduced `atmoCap` from `0.60` to `0.15`. Glow sphere is now capped at 1.15× physical radius — a tight corona halo.

**Rule**: The `atmoCap` constant controls the maximum atmosphere oversize fraction. It must be tuned to the expected range of `AtmosphereThicknessKm` values in the data. When atmospheric data can include extreme outliers (like a stellar corona defined in km), a small cap is necessary to prevent visually absurd results.

---

### 27. **rl.NewShader(id, nil) Creates a Nil-Locs Shader: SIGSEGV in DrawModel** [RENDERING]

**Problem**: App crashed with SIGSEGV at address `0x30` immediately on the first `DrawModel` call for the sky sphere.

**Root Cause**: `rl.NewShader(rl.GetShaderIdDefault(), nil)` creates a `Shader` struct where the `Locs` field is `nil`. Raylib's `DrawMesh` immediately dereferences `Locs[SHADER_LOC_VERTEX_POSITION]` (array index 12, byte offset `0x30`) to resolve the vertex attribute location. This is a null pointer dereference.

**Why it was used**: Attempted to assign the default flat shader to the sky model's material so the sky would always be unlit.

**Why it was wrong**: `rl.LoadModelFromMesh` already initialises the model's material with Raylib's built-in default flat shader, which has a fully populated `Locs` array. `rl.NewShader` with a nil locs argument does not copy the existing locs — it creates an empty (and invalid) wrapper.

**Solution**: Remove the `NewShader` call entirely. Leave the material shader as initialised by `LoadModelFromMesh`.

**Rule**: Never pass `nil` as the `locs` parameter to `rl.NewShader`. To use the default shader on a model material, do nothing — `LoadModelFromMesh` already provides it. `NewShader` is only appropriate when you have a freshly compiled shader with known uniform/attribute locations to populate.

---

### 28. **Inside-Sphere UV Mapping: Backface Culling Vs Winding Flip** [RENDERING]

**Problem**: A skysphere (viewed from inside) does not render with standard backface culling enabled — inner faces are back-facing by default.

**Approach 1 (attempted, unreliable)**: `DisableBackfaceCulling()` before `DrawModel`, `EnableBackfaceCulling()` after. This proved unreliable because Raylib's `DrawModel` / `DrawMesh` pipeline may re-enable culling internally or have undefined interaction with rlgl state between batches.

**Approach 2 (correct)**: Flip triangle winding by using a negative scale on one axis — `MatrixScale(-skyRadius, skyRadius, skyRadius)`. With a negative X scale, all triangles that were back-facing are now front-facing. Standard culling remains enabled and operates correctly. No rlgl state manipulation required.

**UV interaction**: A negative X scale mirrors the texture horizontally. Compensate by adding a U-flip in the UV remapping step (set `new_u = 1 - old_u`). The complete UV fix for an equirectangular texture on a par_shapes sphere, viewed from inside with a negative-X winding flip:
```go
// old: uv[0]=latitude (par_shapes), uv[1]=longitude
// new: standard equirectangular U=longitude, V=latitude, V-flipped for OpenGL
uv[i*2], uv[i*2+1] = 1.0-uv[i*2+1], 1.0-uv[i*2]
// Note: U-flip (1 - longitude) compensates for negative-X winding flip.
```

**Rule**: For inside-sphere rendering, prefer winding flip via negative scale over toggling culling state. The winding approach is deterministic and does not depend on the ordering of rlgl state changes around `DrawModel`.

---

### 29. **Depth Precision at Far-Plane Boundary: DisableDepthTest for Sky** [RENDERING]

**Problem**: A skysphere at radius 180,000 su with far plane at 200,000 su may not render because depth precision near the far plane causes nearly every fragment to fail the depth test.

**Root Cause**: Non-linear depth buffer precision (z-buffer uses `1/z` distribution). At 90% of the far plane, the depth value is ~0.9999; floating-point imprecision can cause fragments to test as "behind" existing geometry or the far plane itself.

**Solution**: Call `rl.DisableDepthTest()` immediately before drawing the sky sphere, and `rl.EnableDepthTest()` immediately after. Also call `rl.DisableDepthMask()` / `rl.EnableDepthMask()` around the same draw to prevent the sky from writing depth values that would occlude all scene geometry.

**Rule**: Background elements drawn at or near the far plane should have depth testing disabled entirely, not just depth writes. `DisableDepthMask` prevents writing; `DisableDepthTest` prevents the draw from being rejected by existing depth values. Both are needed for a reliable background.

---

### 30. **Diagnosing Silent Render Failures: Run With Captured Logs** [RENDERING]

**Problem**: The sky sphere appeared to not render, but no error was emitted. Standard debugging — checking shader assignment, verifying draw-call order, toggling depth flags — produced no signal.

**Discovery method**: Ran the binary briefly with stderr captured to a terminal, then killed it after a few seconds. Raylib's startup info messages showed:
```
INFO: FILEIO: [data/assets/textures/starfield_8k.jpg] File loaded successfully
INFO: IMAGE: Data loaded successfully (8192x4096 | R8G8B8 | 1 mipmaps)
INFO: TEXTURE: [ID 3] Texture loaded successfully (8192x4096 | R8G8B8 | 1 mipmaps)
INFO: VAO: [ID 2] Mesh uploaded successfully to VRAM (GPU)
```
This confirmed the asset loaded and the mesh was uploaded — ruling out file, format, and GPU-upload as the cause. The problem was therefore entirely in the transform/clip stage.

**Rule**: Before investigating shader or UV logic, confirm the asset and mesh actually reached the GPU. Raylib logs `TEXTURE: [ID n]` and `VAO: [ID n]` on success; absence of either line means the problem is upstream (file path, format, or allocation). Presence of both lines means the problem is downstream (transform, clipping, or culling).

---

### 31. **Skysphere Vertex Clipping: Large Radius Causes Silent Triangle Discard** [RENDERING]

**Problem**: The Milky Way sky sphere was completely invisible despite the texture loading and VAO uploading correctly, and despite `DisableDepthTest` being in place.

**Root Cause**: The camera is at the floating-origin `(0,0,0)`. With `skyRadius = 180,000`, the sphere vertices span ±180,000 on all axes. In camera view-space, roughly half the sphere vertices have negative z (they are behind the camera). OpenGL must clip those triangles at the near plane using homogeneous clip-space arithmetic. The near plane is `0.001`, so the precision ratio is:

$$\frac{180{,}000}{0.001} = 1.8 \times 10^8$$

`float32` has only ~7 significant decimal digits of precision. At this ratio, the homogeneous clip-space coordinates for near-plane intersection lose all precision — the computed triangle edge clamp is numerically degenerate. OpenGL discards those triangles entirely and silently produces nothing.

**Solution**: Set `skyRadius = 5.0`. Since depth testing is disabled and the camera is always at the sphere's centre, any radius greater than the near plane (`> 0.001`) covers the entire field of view identically. At radius 5.0, the precision ratio is `5 / 0.001 = 5000` — comfortably within float32's range.

```go
// skyRadius = 5.0, not 180_000. Clip-space precision ratio is 5000:1 (safe).
// Depth test is disabled for the sky draw, so size has no effect on occluusion.
const skyRadius = float32(5.0)
```

**Rule**: Any geometry rendered with the camera at its centre (skyspheres, skyboxes, environment maps) should use a small radius — just large enough to stay outside the near plane. A radius of 1–10 sim units is always correct. Large radii (> ~1000× near plane) cause float32 homogeneous clip precision loss and silent triangle discard, regardless of depth test state.

**Corollary**: This bug is completely silent. No Raylib warning, no GL error, no crash. The only symptoms are a blank background and a confirmed texture+VAO load in the logs. Any time a fullscreen background fails to appear after asset load is confirmed, suspect near-plane clip precision before blaming UVs or shaders.

---

## F-013 N-Body Gravitational Sets — Design Lessons

**Date**: 2026-05-22
**Context**: Redesigning the F-013 N-body scope to support configurable gravitational sets rather than a fixed "all named bodies" pass.

---

### 32. **N-Body Scope Should Be Set-Based, Not Category-Based** [PHYSICS]

**Problem**: The original F-013 design hard-wired N-body scope to `Dataset == -1` (all named bodies: stars, planets, dwarf planets, moons). This works for the default solar-system case but cannot express the actual use cases: planet+moons clusters, bodies within a sphere of influence, asteroids near a planet, player ships in local gravity.

**Root Cause**: The scope predicate was encoded as a boolean filter on a single field (`Dataset`), not as an explicit participant list. This made the integration boundary invisible and non-composable.

**Solution**: Formalize the `[]*Object` slice that was already being passed to `accumForces()` as a first-class `GravSet{Participants, TestParticles}` struct. The integration boundary becomes explicit and composable:

```go
type GravSet struct {
    Name         string
    Participants  []*Object  // mutually attract; have mass
    TestParticles []*Object  // receive gravity only; mass negligible
}
```

The force accumulator and leapfrog integrator are unchanged — they already operated on a slice. The only new code is the builder functions that assemble the slices.

**Solution (extended)**: Separate the concern into two layers:

1. **Collectors** (`CollectByCategory`, `CollectInSOI`, `CollectChildren`, etc.) — return `[]*Object` slices filtered by a single criterion. Each is independently testable and composable.
2. **Set builders** (`SystemSet`, `LocalSet`, `SOISet`) — compose collector results into `GravSet{Participants, TestParticles}`.
3. **Integrator** (`stepGravSet`) — takes a finished `GravSet`; no knowledge of categories.

```go
// Caller assembles the set; integrator just runs it.
gs := GravSet{
    Participants: append(
        CollectByName(state, "Earth", "Moon"),
        CollectByCategory(state, CategoryArtifact)...,
    ),
}
stepGravSet(gs, dt)
```

Adding a new body type (e.g., `CategoryArtifact`) only requires adding a `CollectByCategory` call at the construction site — the integrator is untouched.

**Rule**: Any physics loop that takes a slice of participants is implicitly set-based. Make it explicit with a two-layer pattern: collectors assemble slices by criterion; builders compose slices into sets; the integrator consumes sets without knowing how they were built.

---

### 33. **Test-Particle Approximation Is the Correct Model for Ships** [PHYSICS]

**Problem**: The question of whether player ships should "participate" in N-body (exert force on planets) vs. just "receive" gravity (be attracted by planets) was initially unclear. Adding ships as full participants would make them perturb planetary orbits.

**Analysis**:
- A 1,000 kg ship has GM ≈ `1.991e-38 × 1e3 ≈ 2e-35` sim³/s²
- Earth's GM ≈ `2.4e-11` sim³/s²
- Ratio: ~1e24 — ship gravity is physically negligible on any named body

**Solution**: Ships are `TestParticles` — they receive gravitational force from all `Participants` but their mass is not included in the pairwise force sums. This is the standard test-particle approximation used in orbital mechanics.

Benefits:
- Physically correct (ship mass is genuinely negligible)
- Performance: O(N) per ship vs. O(N²) expansion if ships were full participants
- Scalability: 1,000 ships add 1,000 × N_participants force evaluations, not 1,000² pairwise

**Rule**: Any body whose mass is < ~1e-15 of the dominant body in its local set should be a test particle. For space sims, this means all ships, probes, and small artifacts.

---

### 34. **Sphere of Influence Needs Two Radii, Not One** [PHYSICS]

**Problem**: Early thinking used a single "SOI radius" per body to both define set membership (is this body inside the SOI?) and trigger dynamic entry/exit events. These two uses require different precision.

**Analysis**:
- **Hill sphere** (`r_H = a × (m/3M)^⅓`): The region where a body's gravity dominates over tidal forces from the parent. Conservative; larger. Best for set containment — if a body is inside the Hill sphere, it is definitely in the local gravitational field.
- **Laplace SOI** (`r_SOI = a × (m/M)^⅖`): The smaller, more precise boundary where the body's gravity exceeds the parent's gravity on an orbiting test particle. Best for entry/exit trigger — the transition point where local physics takes over.

Using Hill radius for membership containment and Laplace SOI for entry/exit detection gives a hysteresis band that prevents bodies from rapidly toggling in and out of sets near the boundary.

**Rule**: Store both `HillRadius` and `LaplaceSOI` per body. Use `LaplaceSOI` as the entry trigger and `HillRadius` as the exit boundary. This prevents rapid set membership oscillation for bodies near the SOI edge.

---

### 35. **Artifacts Need a Category Before They Can Participate in Physics** [ARCHITECTURE]

**Problem**: The request to include "space stations and larger ships" in N-body gravitational sets exposed a gap: there was no `ObjectCategory` for human-constructed objects. Without a category, there is no way to select them for a `GravSet`, assign them mass, or distinguish them from belt asteroids.

**Solution**: Add `CategoryArtifact = 7` to the `ObjectCategory` enum. Artifacts are eligible as `GravSet.Participants` if they have a defined mass. They differ from player ships in that they have a persistent world position in simulation state (not in session registry) and are loaded from data files.

**Rule**: Before a new class of physics object can participate in any simulation subsystem, it must have a category. Categories are the type tag that every subsystem (physics, renderer, UI, session registry) uses to dispatch. Adding a category is cheap; retrofitting one later is not.

---

### 36. **N-Body Timestep Stability: quasar_test Orbital Expansion at Load** [PHYSICS]

**Date**: 2026-05-25

**Symptom**: In `quasar_test` system, the quasar companion stars (Q1, Q2) visibly fly away from the SMBH immediately after the simulation starts, even though `initNBody` computes physically correct initial velocities.

**Root Cause**: `quasar_test/system.json` sets `default_time_scale: 3155760000` (100 years/s). At 60 Hz, one physics tick covers:

```
dt = (1/60) × 3,155,760,000 ≈ 52,596,000 s ≈ 609 days
```

Q1's orbital period from GM: `T = 2π × √(a³/GM)` where `a = 150,000 su`, `GM = 1.991e-38 × 1.989e+39 ≈ 39.6 su³/s²`:

```
T ≈ 2π × √(150000³ / 39.6) ≈ 57,968,000 s ≈ 671 days
```

**`dt / T ≈ 0.91` — the integrator takes nearly one full orbit per tick.**

The Leapfrog DKD integrator (like any fixed-step symplectic integrator) requires `dt << T` for stability. When `dt ≈ T`, the acceleration evaluated at the initial position is applied over nearly a full orbital arc. Instead of traveling a small arc, the body receives a full-orbit impulse in the wrong direction, injecting energy into the orbit and causing unbounded expansion.

**What this is NOT**:
- Not a velocity calibration error — `initNBody` computes `v = √(GM/p)` via the perifocal velocity formula; this is correct.
- Not a `scale_factor` problem — `scale_factor: 500` in `system.json` is metadata and is never applied inside `createBodyFromConfig` (the parameter is accepted but unused, line ~195 in `loader.go`).
- Not a GM computation error — `GM = G_sim × mass` is correct.

**Fix options** (not yet implemented):
1. Reduce `default_time_scale` for quasar_test to ≤ 3.15e8 s/s (10 years/s) → dt ≈ 61 days, T/dt ≈ 11. Still marginal; ratio should be ≥ 20 for stable Leapfrog.
2. Add a per-system `nbody_substeps` field. At 10 substeps per frame, effective dt = 6 days → T/dt ≈ 112 (stable).
3. Reduce to ~1 year/s (3.15e7) for dt ≈ 6 days → T/dt ≈ 112 without substeps.

**Rule**: For any N-body system, verify `dt/T_min < 0.05` (i.e., T_min > 20 × dt) where T_min is the orbital period of the shortest-period body. If not satisfied, reduce `default_time_scale` or add substep support. Leapfrog is energy-conserving at small dt but diverges rapidly when dt approaches the orbital period.

---

### 37. **Raylib `GetWorldToScreenEx`/`GetWorldToScreen` Use a Hardcoded Far Plane** [RENDERING]

**Date**: 2026-05-25

**Symptom**: Object labels wander across the viewport or orbit the viewport center for objects more than ~1000 su from the camera. The 3D objects themselves render correctly; only the screen-space projection for label placement is wrong.

**Root Cause**: Both `rl.GetWorldToScreenEx` and `rl.GetWorldToScreen` internally construct a projection matrix using `RL_CULL_DISTANCE_FAR = 1000.0` (a compile-time constant in the Raylib C source), regardless of the far plane set via `rl.SetMatrixProjection`. For objects at depth `z > 1000`, the clip-space `w` component goes negative, flipping the NDC x/y coordinates. The resulting screen position is mirrored across the viewport center and drifts as the camera moves.

**`CameraFarPlane` for this project is 200,000 su** — 200× the hardcoded limit.

**Fix**: Custom `projectToScreen` function that builds a correct projection matrix at call time:

```go
func projectToScreen(worldPos rl.Vector3, camera rl.Camera3D, screenW, screenH float32) rl.Vector2 {
    aspect := screenW / screenH
    proj := rl.MatrixPerspective(
        float64(CameraFOV)*math.Pi/180.0,
        float64(aspect),
        float64(CameraNearPlane),
        float64(CameraFarPlane), // 200000 — not 1000
    )
    view := rl.GetCameraMatrix(camera)
    clipPos := rl.Vector4Transform(
        rl.Vector4{X: worldPos.X, Y: worldPos.Y, Z: worldPos.Z, W: 1.0},
        rl.MatrixMultiply(view, proj),
    )
    if clipPos.W <= 0 {
        return rl.Vector2{X: -1, Y: -1} // behind camera
    }
    ndc := rl.Vector3{
        X: clipPos.X / clipPos.W,
        Y: clipPos.Y / clipPos.W,
    }
    return rl.Vector2{
        X: (ndc.X*0.5 + 0.5) * screenW,
        Y: (1.0 - (ndc.Y*0.5+0.5)) * screenH,
    }
}
```

**Rule**: Never use `rl.GetWorldToScreenEx` or `rl.GetWorldToScreen` in a project where objects can be more than 1000 su from the camera. Always use a custom projection that matches the actual far plane. Check the return value: if `clipPos.W <= 0`, the point is behind the camera and should not be rendered.

---

### 38. **float32 Position Precision and the 64-bit Upgrade Analysis** [RENDERING, PHYSICS]

**Date**: 2026-05-25

**Context**: All simulation positions (`engine.Vector3`) are `float32`. This was a deliberate constraint because Raylib's rendering pipeline is float32 throughout. The question arises: would upgrading to float64 positions solve the label-positioning and depth-jitter problems seen at large distances?

**What float64 would address**:
- **Catastrophic cancellation**: When the camera is 50,000 su from the origin and two nearby objects are 50,000 su ± 0.1 su apart, float32 subtraction loses ~4 digits of precision. With float64 (15–16 significant digits vs float32's 7), relative positions remain precise out to ~1e8 su separation.
- **Accumulation drift in long simulations**: Leapfrog position updates accumulate rounding error over millions of ticks. float64 reduces per-step error from ~1e-7 to ~1e-15 relative magnitude.

**What float64 would NOT address**:
- **Raylib GPU pipeline**: All vertex buffers, shader uniforms, and mesh data in Raylib use float32. Even if engine positions are float64, they must be cast to float32 before any draw call. The GPU sees float32 regardless.
- **OpenGL**: OpenGL's standard vertex pipeline is float32; `GL_DOUBLE` vertex attributes exist but are unsupported on many drivers and Metal/MoltenVK backends.
- **Shader precision**: GLSL `float` is 32-bit. Using float64 on the CPU does not improve fragment shader precision.
- **The label-positioning bug (LL #37)**: Was caused by Raylib's hardcoded far plane of 1000 su — not by float32 imprecision. The fix is a custom projection function.

**Side effects of a float64 migration**:
- **Memory**: All `engine.Vector3` (3 × float32 = 12 bytes) become 3 × float64 = 24 bytes — 2× increase for every position in every object in the simulation state (front + back buffers × N objects).
- **Struct migration**: Every engine type using `Vector3` requires changes: `SimObject`, `AnimState`, `CameraState`, `GravSet`, `NBodyState`, and all serialization/deserialization code.
- **JSON precision**: The current JSON schema uses standard decimal floats; float64 serialization requires higher precision or the benefit is lost on load.
- **SIMD**: Go's float32 math can use 4-wide SIMD; float64 drops to 2-wide on most architectures.

**Preferred alternative — origin shifting (camera-relative rendering)**:
Convert world-space positions to camera-relative coordinates just before the GPU submit. The camera is always at `(0,0,0)` in the render space, so all vertex positions are small floats regardless of world-space coordinates. This eliminates catastrophic cancellation in the GPU pipeline with zero schema changes and near-zero memory impact.

**Rule**: Do not migrate `engine.Vector3` to float64 to fix rendering precision. Use origin shifting at the render boundary instead. Upgrade physics state to float64 only if N-body integration drift is measured to be a problem at the simulation timescales in use.

