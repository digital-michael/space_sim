# Belt Features - Asteroid Belt & Kuiper Belt

## Purpose
Document the procedural belt feature behavior, UI integration, dataset scaling, and technical implementation details for the asteroid belt and Kuiper belt systems.

## Last Updated
2026-03-26

## Table of Contents
1. Overview
2. Spatial Distribution Fix
3. Belt Tracking Category
   3.1 New UI Feature
   3.2 Belt Selection Options
   3.3 Random Object Selection
   3.4 Usage
   3.5 Kepler's Third Law Implementation
4. Technical Details
5. Files Modified
6. Dataset Scaling
7. Performance Notes
8. Future Enhancements

## Overview
The simulation now includes two procedurally generated belts with realistic astronomical properties and full tracking UI integration.

## Spatial Distribution Fix

### Problem
Previously, all asteroids and Kuiper Belt Objects (KBOs) spawned at the same starting position (X=distance, Z=0), creating a visible "wall" that required time to spread out as objects followed their different orbital speeds.

### Solution
Each object now has its initial position calculated from its randomly assigned orbit angle:

```go
orbitAngle := float32(rng.Float64()) * 2 * math.Pi
asteroid.Anim.OrbitAngle = orbitAngle
asteroid.Anim.Position.X = distance * float32(math.Cos(float64(orbitAngle)))
asteroid.Anim.Position.Z = distance * float32(math.Sin(float64(orbitAngle)))
```

This distributes objects evenly around their orbital circumference from the start, eliminating the "wall" effect.

## Belt Tracking Category

### New UI Feature
The tracking dialog now includes a "Belts" category alongside Stars, Planets, Dwarf Planets, and Moons.

### Belt Selection Options

When selecting the "Belts" category, two options appear:

1. **Asteroid Belt**
   - Distance: 195-240 AU (1.95-2.4 AU)
   - Orbital Period: 3-4 years
   - Color indicator: Gray (150, 150, 150)
   - Displays as "195-240 AU" in the list

2. **Kuiper Belt**
   - Distance: 3000-5000 AU (30-50 AU)
   - Orbital Period: 164-353 years
   - Classical Belt: 60% concentrated at 4200-4800 AU (42-48 AU)
   - Color indicator: Reddish-brown (200, 150, 130)
   - Displays as "3000-5000 AU" in the list

### Random Object Selection

When you select a belt and press Enter, the system:
1. Finds all visible objects from that belt (name prefix "Asteroid-" or "KBO-")
2. Randomly selects one object from the belt
3. Jumps to or tracks that randomly selected object

This allows you to quickly navigate to the belt regions and explore random objects within them.

### Usage

1. Press **J** (Jump) or **T** (Track) to open the selection dialog
2. Use **TAB** to switch to the "Belts" category
3. Select "Asteroid Belt" or "Kuiper Belt" with arrow keys
4. Press **Enter** to jump/track to a random object in that belt
5. Press **ESC** to cancel

### Kepler's Third Law Implementation

Both belts use realistic orbital mechanics:

```
T(years) = a^1.5    where a is in AU
T(days) = T(years) × 365.256
```

Examples:
- Object at 2.0 AU: T = 2.0^1.5 = 2.83 years
- Object at 40 AU: T = 40^1.5 = 252.98 years

## Technical Details

### Virtual Belt Indices

The belt category uses virtual indices to represent belt containers:
- `-1` = Asteroid Belt
- `-2` = Kuiper Belt

These are converted to actual object indices when selected by:
1. Filtering objects by name prefix
2. Checking visibility status
3. Using `rl.GetRandomValue()` to select a random index

### Object Naming Convention

- Asteroids: `Asteroid-Small-XXXX` or `Asteroid-Large-XXXX`
- KBOs: `KBO-Small-XXXX` or `KBO-Large-XXXX`

Where XXXX is a unique number (0-padded to 5 digits).

### CategoryBelt Enum

Added to `internal/space/state.go`:

```go
const (
    CategoryPlanet ObjectCategory = iota
    CategoryDwarfPlanet
    CategoryMoon
    CategoryAsteroid
    CategoryRing
    CategoryStar
    CategoryBelt  // Virtual category for belt tracking
)
```

## Files Modified

1. **internal/space/asteroids.go**
   - Fixed large asteroid position distribution (lines 69-74)
   - Fixed medium asteroid position distribution (lines 109-114)

2. **internal/space/kuiper.go**
   - Fixed large KBO position distribution (lines 77-82)
   - Fixed medium KBO position distribution (lines 119-124)

3. **internal/space/state.go**
   - Added `CategoryBelt` to ObjectCategory enum (line 79)

4. **cmd/space-sim/main.go**
   - Added "Belts" tab to category selector (line 1517)
   - Updated `filterObjectsByCategory()` to return virtual belt indices (lines 1694-1723)
   - Updated `filterObjectsByCategoryAndText()` to handle belt filtering (lines 1726-1783)
   - Updated display code to show belt names and distances (lines 1578-1613)
   - Updated selection confirmation to pick random belt objects (lines 741-779)

## Dataset Scaling

Both belts scale with dataset size:

### Asteroid Belt
- Small: 200 large asteroids
- Medium: 200 large + 1000 medium
- Large: 400 large + 2000 medium
- Huge: 4000 large + 20000 medium

### Kuiper Belt
- Small: 200 large KBOs
- Medium: 200 large + 1000 medium
- Large: 400 large + 2000 medium
- Huge: 4000 large + 20000 medium

Press **+** or **-** to cycle through dataset sizes.

## Performance Notes

- Virtual belt indices are filtered out of distance cache calculations
- Random selection only considers visible objects
- Belt tracking works across all dataset sizes
- Distance cache updates every 5 seconds for performance

## Future Enhancements

Potential improvements:
- Add filter text search for "asteroid" or "kuiper" in Belts category
- Show count of visible objects in each belt
- Add option to track multiple random objects in sequence
- Display belt statistics (min/max distance, object count)
