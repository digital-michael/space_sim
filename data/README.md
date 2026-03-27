# Solar System Data Configuration

This directory contains JSON configuration files for the data-driven solar system simulation. The JSON system allows easy customization of celestial bodies, orbital mechanics, rendering properties, and procedural features without modifying code.

## Directory Structure

```
data/
├── systems/           # Complete solar system configurations
│   └── solar_system.json
├── bodies/            # Reusable body templates
│   ├── stars.json
│   ├── planets.json
│   ├── dwarf_planets.json
│   └── moons.json
└── assets/
    ├── textures/      # Texture image files
    └── textures.json  # Texture manifest
```

## Importance Scale (0-100)

The importance field controls rendering priority and performance culling. Objects below the importance threshold are not rendered.

| Value | Category | Examples | Usage |
|-------|----------|----------|-------|
| 100 | Stars | Sun | Always render - primary light sources |
| 90 | Gas Giants | Jupiter, Saturn | Major planets, high visual impact |
| 80 | Rocky Planets | Mercury, Venus, Earth, Mars | Primary terrestrial bodies |
| 70 | Ice Giants | Uranus, Neptune | Outer planets |
| 60 | Large Moons | Moon, Io, Europa, Ganymede, Callisto, Titan | Major satellites with geological interest |
| 50 | Dwarf Planets | Pluto, Eris, Haumea, Makemake, Ceres | Trans-Neptunian objects |
| 40 | Small Moons | Most minor satellites | Tiny moons, irregular shapes |
| 30 | Ring Systems | Saturn's rings, Jupiter's rings | Particle belts around planets |
| 10 | Asteroids | Main Belt asteroids | Procedurally generated debris |
| 5 | Comets | Kuiper Belt objects | Distant icy bodies |

### Performance Culling Levels

Press **P** in-game to access performance menu, then use **LEFT/RIGHT** arrows on "Importance Threshold":

- **0** - Render all objects (default, ~24,000 asteroids visible)
- **10** - Hide asteroids/comets (~500 objects visible)
- **30** - Major bodies only (~300 objects: planets, moons, dwarf planets)
- **50** - Planets and stars only (~20 objects)

## JSON Schema Reference

### SystemConfig

Top-level configuration for a complete solar system.

```json
{
  "name": "Solar System",
  "version": "1.0",
  "scale_factor": 50,
  "time_scale": 1.0,
  "bodies": [...],
  "features": [...],
  "templates": "data/bodies",
  "default_state": {
    "simulation_hz": 60,
    "worker_threads": 4,
    "initial_dataset": "small"
  }
}
```

**Fields:**
- `name` - Display name for the system
- `version` - Configuration version (for migration)
- `scale_factor` - Distance scaling (1 = realistic AU distances)
- `time_scale` - Orbital speed multiplier (1 = real-time)
- `bodies` - Array of celestial bodies (planets, moons, etc.)
- `features` - Array of procedural features (asteroid belts, rings) stored in the system file
- `templates` - Path to template library directory
- `default_state` - Initial simulation parameters

### BodyConfig

Configuration for a single celestial body (star, planet, moon, dwarf planet).

```json
{
  "type": "planet",
  "name": "Earth",
  "parent": "Sun",
  "template": "terrestrial_planet",
  "orbit": {
    "semi_major_axis": 100.0,
    "eccentricity": 0.0167,
    "inclination": 0.0,
    "orbital_period": 365.25,
    "initial_mean_anomaly": "random",
    "orbital_speed": 1.0
  },
  "physical": {
    "radius": 0.5,
    "mass": 5.972e24,
    "rotation_period": 24.0,
    "axial_tilt": 23.5,
    "color": [100, 149, 237, 255]
  },
  "rendering": {
    "material": "diffuse",
    "texture": "data/assets/textures/earth_daymap_8k.jpg",
    "normal_map": "data/assets/textures/earth_normal_8k.jpg",
    "lod_levels": [
      {"distance": 0, "rings": 32, "slices": 32},
      {"distance": 100, "rings": 16, "slices": 16},
      {"distance": 500, "rings": 8, "slices": 8}
    ],
    "atmosphere": {
      "enabled": true,
      "color": [135, 206, 235, 128],
      "thickness": 0.05
    }
  },
  "importance": 80
}
```

**Type values:** `star`, `planet`, `moon`, `dwarf_planet`, `ring`

**Material values:** `diffuse`, `emissive`, `metallic`, `mirror`

### OrbitConfig

Keplerian orbital elements.

- `semi_major_axis` - Average orbital distance (scaled units)
- `eccentricity` - Orbit ellipse shape (0 = circle, <1 = ellipse)
- `inclination` - Angle from orbital plane (degrees)
- `longitude_ascending_node` - Where orbit crosses reference plane (degrees)
- `argument_of_periapsis` - Orientation of ellipse (degrees)
- `orbital_period` - Time for one orbit (Earth days)
- `initial_mean_anomaly` - Starting position ("random" or angle in degrees)
- `orbital_speed` - Angular velocity (radians/sec, calculated from period)

### FeatureConfig

Procedural features like asteroid belts or ring systems.

```json
{
  "type": "asteroid_belt",
  "name": "Main Belt",
  "parent": "Sun",
  "distribution": {
    "inner_radius": 150.0,
    "outer_radius": 230.0,
    "thickness": 10.0,
    "density_profile": "gaussian"
  },
  "procedural": {
    "seed": 42,
    "size_range": [0.05, 0.15],
    "color_variation": 0.2,
    "shape_model": "sphere"
  },
  "count_levels": [200, 1200, 6000, 24000]
}
```

**Distribution profiles:** `uniform`, `gaussian`, `exponential`

**Count levels:** Asteroids per dataset [Small, Medium, Large, Huge]

## Template System

Templates provide reusable configurations for similar bodies. Use `template` field to inherit base properties, then `overrides` to customize.

### Creating a Custom Planet

```json
{
  "type": "planet",
  "name": "MyPlanet",
  "template": "terrestrial_planet",
  "orbit": {
    "semi_major_axis": 120.0,
    "orbital_speed": 0.8
  },
  "physical": {
    "radius": 0.6,
    "color": [255, 100, 50, 255]
  },
  "importance": 80
}
```

This inherits all properties from `terrestrial_planet` template but overrides orbit, size, and color.

### Available Templates

**Stars:** `g_type_star` (Sun-like)

**Planets:** 
- `terrestrial_planet` (Mercury/Venus/Earth/Mars-like)
- `gas_giant` (Jupiter/Saturn-like)
- `ice_giant` (Uranus/Neptune-like)

**Dwarf Planets:** `dwarf_planet_base` (Pluto-like)

**Moons:**
- `large_moon` (Moon/Titan/Ganymede-like)
- `small_moon` (Minor satellites)

**Features:**
- `main_belt` (Asteroid belt between Mars and Jupiter)
- `kuiper_belt` (Trans-Neptunian region)

## Orbital Mechanics Examples

### Circular Orbit
```json
"orbit": {
  "semi_major_axis": 100.0,
  "eccentricity": 0.0,
  "orbital_speed": 1.0
}
```

### Elliptical Orbit (Comet-like)
```json
"orbit": {
  "semi_major_axis": 500.0,
  "eccentricity": 0.8,
  "inclination": 45.0,
  "orbital_speed": 0.1
}
```

### Retrograde Orbit
```json
"orbit": {
  "semi_major_axis": 80.0,
  "inclination": 165.0,
  "orbital_speed": 1.5
}
```

## Texture Organization

Textures are organized by body name and resolution. See `data/assets/textures.json` for the complete manifest.

### Loading Strategy by Importance

| Importance | Resolution | Example Bodies |
|------------|-----------|----------------|
| 100 | 8K | Sun |
| 90 | 4K | Jupiter, Saturn |
| 80 | 4K | Earth, Mars, Venus, Mercury |
| 70 | 2K | Uranus, Neptune |
| 60 | 2K | Moon, Galilean moons, Titan |
| 50 | 1K | Pluto, Eris |
| 40 | 1K | Small moons |
| 30 | 512 | Ring particles |
| 10 | 512 | Asteroids (procedural) |

### Downloading Textures

Run the provided script to download NASA/JPL textures:

```bash
./scripts/download_textures.sh
```

This downloads ~500MB of high-resolution texture maps from Solar System Scope (CC BY 4.0).

## Usage Examples

### Loading a Custom System

```bash
./bin/space-sim --system-config=data/systems/my_system.json
```

### JSON-Only System Workflow

The legacy export tool has been removed. Create or update system definitions by
editing JSON files directly under `data/systems/` and the reusable templates
under `data/bodies/`.

### Creating a Minimal System

```json
{
  "name": "Binary Star System",
  "bodies": [
    {
      "type": "star",
      "name": "Star A",
      "template": "g_type_star",
      "importance": 100
    },
    {
      "type": "star",
      "name": "Star B",
      "template": "g_type_star",
      "orbit": {
        "semi_major_axis": 200.0,
        "orbital_speed": 0.5
      },
      "physical": {
        "radius": 20.0,
        "color": [255, 200, 150, 255]
      },
      "importance": 100
    },
    {
      "type": "planet",
      "name": "Planet",
      "parent": "Star A",
      "template": "terrestrial_planet",
      "orbit": {
        "semi_major_axis": 150.0,
        "orbital_speed": 1.0
      },
      "importance": 80
    }
  ]
}
```

## Adding New Metrics

The JSON schema is extensible. To add new properties:

1. Add field to appropriate config struct in `internal/space/config.go`
2. Update loader in `internal/space/loader.go` to read the field
3. Update the relevant JSON templates or system files to carry the field
4. Use the field in simulation/rendering code

Example - adding magnetic field strength:

```json
"physical": {
  "radius": 0.5,
  "mass": 5.972e24,
  "magnetic_field_strength": 50.0  // New field
}
```

## License & Attribution

Configuration files are MIT licensed (same as project).

Textures from Solar System Scope require attribution:
```
Textures courtesy of Solar System Scope (solarsystemscope.com)
Licensed under CC BY 4.0
```

NASA textures are public domain (no attribution required).

## Troubleshooting

**JSON fails to load:**
- Check JSON syntax with `jq data/systems/solar_system.json`
- Verify all required fields are present
- Check file paths are absolute or relative to working directory

**Objects not visible:**
- Check `importance` value vs performance threshold (Press P, check threshold)
- Verify `visible` field is true (default)
- Check orbital parameters place object within view distance

**Performance issues:**
- Increase importance threshold (P menu, set to 10 or 30)
- Reduce asteroid count_levels in features
- Enable frustum culling and spatial partitioning (P menu)
- Use lower resolution textures

**Template not found:**
- Verify template name matches file in `data/bodies/`
- Check `templates` path in system config
- Template files must be valid JSON

## Further Reading

- [Keplerian Orbital Elements](https://en.wikipedia.org/wiki/Orbital_elements)
- [NASA 3D Resources](https://nasa3d.arc.nasa.gov/)
- [Solar System Scope Textures](https://www.solarsystemscope.com/textures/)
- [Raylib Documentation](https://www.raylib.com/)
