# Debug Logging Implementation Summary

## Purpose
Describe the debug logging instrumentation, test scripts, and diagnostic workflow used to investigate hangs or slowdowns in performance testing.

## Last Updated
2026-03-26

## Table of Contents
1. Changes Made
2. Key Diagnostic Points
3. Test Scripts Created
4. How to Use
5. Expected Behavior
6. Next Steps After Test

## Changes Made

### 1. Application Exit Tracking
**Location**: `main()` function

**Features**:
- Defer function with panic recovery to log ALL exit scenarios
- Exit reason tracking variable that gets set based on how app terminates
- Logs both to console and debug log file

**Exit Scenarios Tracked**:
- `panic: <error>` - Application crashed with panic
- `user interrupt (Ctrl+C)` - User pressed Ctrl+C or SIGTERM
- `performance test completed` - Performance tests finished normally
- `user quit (ESC key)` - User pressed ESC in free-fly mode
- `window closed by user` - User clicked window close button

### 2. Frame-Level Detailed Logging
**Location**: `runSingleTest()` measurement loop

**Logs Every Frame**:
- `[FRAME N] Starting frame render` - Frame begins
- `[FRAME N] State locked` - Simulation state locked
- `[FRAME N] Got state, X objects` - State retrieved with object count
- `[FRAME N] BeginDrawing` - Raylib BeginDrawing called
- `[FRAME N] ClearBackground` - Screen cleared
- `[FRAME N] SetMatrixProjection` - Projection matrix set
- `[FRAME N] BeginMode3D` - 3D mode started
- `[FRAME N] Starting culling` - Culling phase begins
- `[FRAME N] Culling complete, X visible objects, took Yms` - Culling done
- `[FRAME N] Starting draw of X objects` - Draw phase begins
- `[FRAME N] Using individual object rendering` - Render method chosen
- `[FRAME N] Drawing object X/Y (cat:C, r:R)` - Every 100th object logged
- `[FRAME N] Draw complete, took Yms` - All objects drawn
- `[FRAME N] EndMode3D` - 3D mode ended
- `[FRAME N] EndDrawing` - Drawing ended
- `[FRAME N] UnlockFront` - State unlocked
- `[FRAME N] Frame complete` - Frame finished

### 3. Object-Level Detailed Logging
**Location**: `drawObject()` function

**Logs For Each Object**:
- `[DRAW] Start: cat=X, r=Y, LOD=true/false, Point=true/false` - Object properties
- `[DRAW] Calculating distance` - Distance calculation begins
- `[DRAW] Distance calculated: X` - Distance from camera
- `[DRAW] Checking point rendering threshold` - Point rendering check
- `[DRAW] Using point rendering` - If using point mode
- `[DRAW] Point drawn, returning` - Point rendered
- `[DRAW] Detected ring object` - If object is a ring
- `[DRAW] Ring drawn, returning` - Ring rendered
- `[DRAW] Drawing sphere, LOD=true/false` - Sphere rendering mode
- `[DRAW] Applying LOD based on distance X` - LOD calculation
- `[DRAW] Small object, reducing LOD` - LOD reduction for small objects
- `[DRAW] LOD determined: rings=X, slices=Y` - Final LOD parameters
- `[DRAW] Calling DrawSphereEx with radius=X, rings=Y, slices=Z` - **CRITICAL: Right before Raylib call**
- `[DRAW] DrawSphereEx complete` - **CRITICAL: After Raylib call**
- `[DRAW] Drawing wireframe` - Wireframe phase
- `[DRAW] Wireframe complete` - Wireframe done
- `[DRAW] Object complete` - Object fully rendered

### 4. Test Progress Logging
**Already implemented**:
- Test start/end with configuration details
- Memory tracking at test boundaries
- Mid-test memory snapshots
- FPS and timing results

## Key Diagnostic Points

### To Identify Raylib Hang:
Look for logs that show:
```
[DRAW] Calling DrawSphereEx with radius=X, rings=Y, slices=Z
```
**WITHOUT** the following log:
```
[DRAW] DrawSphereEx complete
```

This means the hang is **inside Raylib's DrawSphereEx()** function.

### To Identify Frame Processing Issues:
Check which frame operation doesn't complete:
- Missing `[FRAME N] BeginDrawing` → Issue before rendering
- Missing `[FRAME N] EndMode3D` → Issue during 3D rendering
- Missing `[FRAME N] EndDrawing` → Issue completing frame
- Missing `[FRAME N] Frame complete` → Issue unlocking state

### To Identify Object-Specific Issues:
Check which object/category causes the hang:
- Last `[FRAME N] Drawing object X/Y (cat:C, r:R)` shows which object
- Category codes: 0=Star, 1=Planet, 2=Moon, 3=Asteroid, 4=Ring
- Radius values help identify specific objects

## Test Scripts Created

### 1. `scripts/run_simple_test.sh`
- Runs test without timeout
- User can Ctrl+C if it hangs
- Shows summary at end
- **Recommended for testing**

### 2. `scripts/run_debug_test.sh`
- Has timeout mechanism (120s)
- More complex error handling
- May need timeout command installed

## How to Use

### Run the test:
```bash
chmod +x scripts/run_simple_test.sh
./scripts/run_simple_test.sh
```

### Monitor progress:
In another terminal:
```bash
tail -f performance_debug.log | grep "\[FRAME\|DrawSphereEx"
```

### If it hangs:
1. Press Ctrl+C
2. Check last log entry:
```bash
tail -20 performance_debug.log
```

3. Find exactly where it stopped:
```bash
grep "DrawSphereEx" performance_debug.log | tail -5
```

### Analyze results:
```bash
# How many frames completed?
grep -c "Frame complete" performance_debug.log

# How many DrawSphereEx calls?
grep -c "Calling DrawSphereEx" performance_debug.log

# Did they all complete?
grep -c "DrawSphereEx complete" performance_debug.log

# Find incomplete DrawSphereEx calls
grep -A 1 "Calling DrawSphereEx" performance_debug.log | grep -B 1 -v "complete"
```

## Expected Behavior

### Normal Operation:
- Each "Calling DrawSphereEx" is followed by "DrawSphereEx complete"
- Each frame shows all steps completing
- Application terminates with "performance test completed"

### Hang Detected:
- "Calling DrawSphereEx" without "complete"
- Frame processing stops mid-operation
- Application terminates with "user interrupt (Ctrl+C)"
- Exit code will indicate the reason

## Next Steps After Test

1. **If hang is in DrawSphereEx**:
   - Note the exact parameters (radius, rings, slices)
   - Note the LOD settings and distance
   - Note the object category and count
   - Check Raylib version: `grep "raylib-go" go.mod`
   - Create minimal reproduction case

2. **If hang is elsewhere**:
   - Check which frame operation didn't complete
   - Look for patterns (specific test, object count, etc.)
   - Review that code section for locks/blocks

3. **Reporting to Raylib**:
   - Collect: Raylib version, OS version, GPU info
   - Create standalone test case
   - Include exact parameters that cause hang
   - Show consistent reproducibility
