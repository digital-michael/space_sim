# Solar System JSON Schema Documentation

## Purpose
Describe the structure, conventions, examples, validation rules, and generated-object behavior for the solar system JSON configuration used by the application.

## Last Updated
2026-03-26

## Table of Contents
1. File Structure
2. Body Definition
3. Orbit Object
4. Physical Object
5. Rendering Object
6. Examples
  6.1 Complete Example
  6.2 Moon Example
  6.3 Ring System Example
7. Data Sources and Validation
8. Common Mistakes
9. Version History and Related Documents
10. Dynamically Generated Objects

This document describes the structure and conventions for `solar_system.json`, which defines celestial bodies, their orbital mechanics, and physical properties for the Space Sim Solar System Simulator.

## File Structure

```json
{
  "name": "Solar System",
  "version": "1.0",
  "scale_factor": 50,
  "time_scale": 1,
  "bodies": [ ... ]
}
```

### Top-Level Properties

| Property | Type | Description |
|----------|------|-------------|
| `name` | string | Human-readable name of the system |
| `version` | string | Schema version (currently "1.0") |
| `scale_factor` | number | **DEPRECATED** - Not currently used. Originally intended for distance scaling. |
| `time_scale` | number | **DEPRECATED** - Use `SecondsPerSecond` in simulation state instead. Controls time flow rate. |
| `bodies` | array | Array of celestial body definitions |

## Body Definition

Each body in the `bodies` array has the following structure:

```json
{
  "type": "planet|star|moon|dwarf_planet|ring",
  "name": "Earth",
  "parent": "Sun",
  "orbit": { ... },
  "physical": { ... },
  "rendering": { ... },
  "importance": 80
}
```

### Body Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `type` | string | Yes | Body type: `"star"`, `"planet"`, `"dwarf_planet"`, `"moon"`, `"ring"` |
| `name` | string | Yes | Unique identifier for the body |
| `parent` | string | Conditional | Required for moons and rings. Name of the parent body. |
| `orbit` | object | Yes | Orbital mechanics parameters (see below) |
| `physical` | object | Yes | Physical properties (see below) |
| `rendering` | object | Yes | Rendering properties (see below) |
| `importance` | integer | Yes | Rendering priority (0-100, see priority table below) |

### Importance Values

The `importance` field controls rendering priority and Level of Detail (LOD) culling:

| Category | Importance | Notes |
|----------|------------|-------|
| Stars | 100 | Never culled, always rendered |
| Gas Giants | 90 | Jupiter, Saturn, Uranus, Neptune |
| Rocky Planets | 80 | Mercury, Venus, Earth, Mars |
| Major Moons | 60 | Earth's Moon, Galilean moons, Titan, etc. |
| Dwarf Planets | 50 | Pluto, Ceres, Eris, etc. |
| Minor Moons | 40 | Small moons, irregular satellites |
| Rings | 30 | Planetary ring systems |
| Large Asteroids | 8-15 | 100+ km diameter |
| Medium Asteroids | 4-10 | 30-100 km diameter |

## Orbit Object

The `orbit` object defines Keplerian orbital elements and motion parameters.

```json
"orbit": {
  "semi_major_axis": 100,
  "eccentricity": 0.017,
  "inclination": 0.0,
  "longitude_ascending_node": 0.0,
  "argument_periapsis": 1.796158,
  "orbital_period": 365.256,
  "initial_mean_anomaly": "random"
}
```

### Orbit Properties

| Property | Type | Unit | Required | Description |
|----------|------|------|----------|-------------|
| `semi_major_axis` | number | simulation units | Yes | Average orbital radius. **1 AU = 100 units** (Earth = 100) |
| `eccentricity` | number | dimensionless | Yes | Orbital eccentricity. 0=perfect circle, <1=ellipse, ≥1=hyperbola |
| `inclination` | number | **radians** | Yes | Tilt from reference plane. Earth ecliptic = 0.0 |
| `longitude_ascending_node` | number | **radians** | Yes | Longitude of ascending node (Ω). Orientation of orbital plane |
| `argument_periapsis` | number | **radians** | Yes | Argument of periapsis (ω). Direction to closest approach |
| `orbital_period` | number | **days** | **Yes** | Orbital period in Earth days. **DO NOT USE `orbital_speed`** |
| `initial_mean_anomaly` | string/number | radians or "random" | Yes | Starting position. Use `"random"` for randomized distribution |

### ⚠️ CRITICAL: Orbital Period vs Orbital Speed

**Always use `orbital_period` in Earth days. Never use `orbital_speed`.**

The `orbital_speed` field is **deprecated** and will cause incorrect orbital motion. The physics engine calculates speed from the orbital period using:

```
Mean Motion (n) = 2π / (orbital_period_in_seconds)
```

### Distance Scale

The simulation uses **100 units = 1 AU**:

- **Sun**: `semi_major_axis: 0` (stationary at origin)
- **Mercury**: `semi_major_axis: 39` (0.39 AU)
- **Earth**: `semi_major_axis: 100` (1.0 AU)
- **Mars**: `semi_major_axis: 152` (1.52 AU)
- **Jupiter**: `semi_major_axis: 520` (5.2 AU)
- **Asteroid Belt**: ~195-240 units (1.95-2.4 AU)
- **Kuiper Belt**: ~3000-5000 units (30-50 AU)

### Calculating Orbital Period

Use **Kepler's Third Law** to calculate realistic periods:

```
T² = a³  (where T is in years, a is in AU)
T = a^1.5 years
T_days = T * 365.256
```

**Examples:**
- **Earth** (1.0 AU): T = 1^1.5 = 1 year = **365.256 days**
- **Mars** (1.52 AU): T = 1.52^1.5 ≈ 1.88 years = **686.98 days**
- **Jupiter** (5.2 AU): T = 5.2^1.5 ≈ 11.86 years = **4332.59 days**
- **Asteroid** (2.3 AU): T = 2.3^1.5 ≈ 3.49 years = **1275 days**

## Physical Object

The `physical` object defines size, mass, and appearance.

```json
"physical": {
  "radius": 0.5,
  "inner_radius": 0.0,
  "mass": 5.972e+24,
  "color": [100, 149, 237, 255]
}
```

### Physical Properties

| Property | Type | Unit | Required | Description |
|----------|------|------|----------|-------------|
| `radius` | number | simulation units | Yes | Physical radius (or outer radius for rings). Visual size in 3D |
| `inner_radius` | number | simulation units | No | Inner radius for ring systems only. 0 for spherical bodies |
| `mass` | number | kilograms | Yes | Mass in kg. Used for gravitational calculations |
| `color` | array[4] | RGBA 0-255 | Yes | Color as `[R, G, B, A]` where each value is 0-255 |

### Size Scale

Sizes are relative for visibility, not to astronomical scale:

- **Sun**: `radius: 27.25` (massive for visibility)
- **Jupiter**: `radius: 5.6` (largest planet)
- **Earth**: `radius: 0.5` (reference size)
- **Moon**: `radius: 0.14`
- **Large Asteroids**: `radius: 0.15-0.25`
- **Medium Asteroids**: `radius: 0.05-0.13`

## Rendering Object

The `rendering` object controls visual appearance.

```json
"rendering": {
  "material": "diffuse"
}
```

### Rendering Properties

| Property | Type | Options | Description |
|----------|------|---------|-------------|
| `material` | string | See below | Material type for shading |

### Material Types

| Material | Use Case | Visual Effect |
|----------|----------|---------------|
| `"emissive"` | Stars (Sun) | Self-illuminated, glowing |
| `"diffuse"` | Planets, moons | Matte surface, standard shading |
| `"metallic"` | Asteroids | Shiny, reflective metallic surface |
| `"mirror"` | Satellites (future) | Highly reflective |

## Complete Example

```json
{
  "type": "planet",
  "name": "Earth",
  "orbit": {
    "semi_major_axis": 100,
    "eccentricity": 0.017,
    "inclination": 0.0,
    "longitude_ascending_node": 0.0,
    "argument_periapsis": 1.796158,
    "orbital_period": 365.256,
    "initial_mean_anomaly": "random"
  },
  "physical": {
    "radius": 0.5,
    "mass": 5.972e+24,
    "color": [100, 149, 237, 255]
  },
  "rendering": {
    "material": "diffuse"
  },
  "importance": 80
}
```

## Moon Example

Moons require a `parent` field and use local coordinates:

```json
{
  "type": "moon",
  "name": "Moon",
  "parent": "Earth",
  "orbit": {
    "semi_major_axis": 3.84,
    "eccentricity": 0.0549,
    "inclination": 0.089,
    "longitude_ascending_node": 0.0,
    "argument_periapsis": 0.0,
    "orbital_period": 27.322,
    "initial_mean_anomaly": "random"
  },
  "physical": {
    "radius": 0.14,
    "mass": 7.342e+22,
    "color": [200, 200, 200, 255]
  },
  "rendering": {
    "material": "diffuse"
  },
  "importance": 60
}
```

## Ring System Example

Rings orbit extremely close to their parent and have `inner_radius`:

```json
{
  "type": "ring",
  "name": "Saturn-Ring-B",
  "parent": "Saturn",
  "orbit": {
    "semi_major_axis": 0,
    "eccentricity": 0,
    "inclination": 0,
    "longitude_ascending_node": 0,
    "argument_periapsis": 0,
    "orbital_period": 0.42,
    "initial_mean_anomaly": "0"
  },
  "physical": {
    "radius": 12.0,
    "inner_radius": 9.2,
    "mass": 1e+19,
    "color": [210, 180, 140, 200]
  },
  "rendering": {
    "material": "diffuse"
  },
  "importance": 30
}
```

## Data Sources

Orbital periods and elements are sourced from:
- **NASA/JPL Horizons System**: https://ssd.jpl.nasa.gov/horizons/
- **IAU Minor Planet Center**: https://www.minorplanetcenter.net/
- **NASA Planetary Fact Sheets**: https://nssdc.gsfc.nasa.gov/planetary/factsheet/

## Validation Checklist

Before adding a new body, ensure:

- ✅ `orbital_period` is specified in **Earth days**
- ✅ `orbital_speed` is **NOT present** (deprecated)
- ✅ `inclination`, `longitude_ascending_node`, `argument_periapsis` are in **radians**
- ✅ `semi_major_axis` uses 100 units = 1 AU scale
- ✅ `importance` matches body category (see table above)
- ✅ `initial_mean_anomaly` is either `"random"` or a specific radian value
- ✅ Moons have `parent` field set to parent body name
- ✅ `color` is RGBA array with values 0-255

## Common Mistakes

### ❌ Using orbital_speed
```json
"orbit": {
  "semi_major_axis": 100,
  "orbital_speed": 1.0  // WRONG - causes ~6 second orbit!
}
```

### ✅ Using orbital_period
```json
"orbit": {
  "semi_major_axis": 100,
  "orbital_period": 365.256  // CORRECT - 1 Earth year
}
```

### ❌ Angles in degrees
```json
"orbit": {
  "inclination": 7.155  // WRONG - degrees
}
```

### ✅ Angles in radians
```json
"orbit": {
  "inclination": 0.1249  // CORRECT - 7.155° converted to radians
}
```

## Version History

- **v1.0** (2026-02-24): Initial schema with corrected orbital periods for all 304 major bodies and ring systems.

## See Also

- `internal/space/loader.go`: JSON loading implementation
- `internal/space/objects.go`: Object creation functions
- `internal/space/asteroids.go`: Dynamic asteroid generation
- `scripts/fix_all_orbital_periods.py`: Python script to correct orbital periods

## Dynamically Generated Objects

### Asteroid Belt

The **Asteroid Belt** is generated dynamically at runtime, not defined in the JSON. Asteroids are created by `CreateAsteroids()` in `internal/space/asteroids.go`.

**Properties:**
- **Distance**: 195-240 simulation units (1.95-2.4 AU)
- **Orbital Periods**: 3-4 years (calculated via Kepler's law)
- **Vertical Spread**: ±7.5 units (0.15 AU)
- **Count**: 200 (Small) → 1,200 (Medium) → 2,400 (Large) → 24,000 (Huge)
- **Toggle**: `+` / `-` keys cycle through dataset sizes

**Size Categories:**
- **Large Asteroids**: 0.15-0.25 radius (100+ km diameter)
- **Medium Asteroids**: 0.05-0.13 radius (30-100 km diameter)

### Kuiper Belt

The **Kuiper Belt** is also generated dynamically at runtime by `CreateKuiperBelt()` in `internal/space/kuiper.go`.

**Properties:**
- **Distance**: 3000-5000 simulation units (30-50 AU)
  - **Classical Belt**: 4200-4800 units (42-48 AU, 60% of objects)
- **Orbital Periods**: 164-353 years (calculated via Kepler's law)
- **Vertical Spread**: ±25 units (thicker distribution than asteroid belt)
- **Count**: 200 (Small) → 1,200 (Medium) → 2,400 (Large) → 24,000 (Huge)
- **Toggle**: Same as asteroid belt (`+` / `-` keys)
- **Color**: Reddish/brownish tones (RGB: 180-220, 140-180, 120-160)

**Size Categories:**
- **Large KBOs**: 0.15-0.30 radius (100-300 km diameter, dwarf planet candidates)
- **Medium KBOs**: 0.05-0.15 radius (30-100 km diameter)

**Eccentricity**: 0.01-0.16 (more eccentric than asteroids)
**Inclination**: ±8.6° typical

### Implementation Notes

Both asteroid and Kuiper Belt objects:
- Use realistic orbital periods calculated from their distance
- Are marked with a dataset level (Small/Medium/Large/Huge)
- Share the same dataset toggle system (both visible or both hidden)
- Have randomized colors and sizes within their category
- Use Keplerian orbital mechanics for accurate motion

To generate different distributions, modify:
- `internal/space/asteroids.go`: Asteroid generation logic
- `internal/space/kuiper.go`: Kuiper Belt generation logic

