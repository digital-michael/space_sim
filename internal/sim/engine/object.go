package engine

// MaterialType defines the rendering material.
type MaterialType int

const (
	MaterialDiffuse  MaterialType = iota // Matte surface
	MaterialEmissive                     // Glowing (sun)
	MaterialMetallic                     // Shiny metal (asteroids)
	MaterialMirror                       // Reflective (satellites)
)

// ObjectCategory defines object grouping for UI navigation.
// Values are generic — the loader maps JSON "type" strings onto them.
type ObjectCategory int

const (
	CategoryPlanet      ObjectCategory = iota // 0
	CategoryDwarfPlanet                       // 1
	CategoryMoon                              // 2
	CategoryAsteroid                          // 3
	CategoryRing                              // 4
	CategoryStar                              // 5
	CategoryBelt        ObjectCategory = 6    // Virtual: asteroid/Kuiper belts
)

// AsteroidDataset represents a LOD level for asteroid populations.
type AsteroidDataset int

const (
	AsteroidDatasetSmall  AsteroidDataset = 0 // 200 objects
	AsteroidDatasetMedium AsteroidDataset = 1 // 1,200 objects
	AsteroidDatasetLarge  AsteroidDataset = 2 // 2,400 objects
	AsteroidDatasetHuge   AsteroidDataset = 3 // 24,000 objects
)

// Name returns a human-readable label for the dataset tier.
func (d AsteroidDataset) Name() string {
	switch d {
	case AsteroidDatasetSmall:
		return "Small"
	case AsteroidDatasetMedium:
		return "Medium"
	case AsteroidDatasetLarge:
		return "Large"
	case AsteroidDatasetHuge:
		return "Huge"
	default:
		return "Unknown"
	}
}

// ObjectMetadata contains immutable physical and rendering properties.
type ObjectMetadata struct {
	Name           string         // Display name
	Category       ObjectCategory // Object category for UI grouping
	Mass           float64        // Mass in kilograms
	PhysicalRadius float32        // Physical size (or outer radius for rings)
	InnerRadius    float32        // Inner radius (rings only; 0 for spheres)
	Color          Color          // Display color
	Material       MaterialType   // Rendering material
	Importance     int            // Rendering priority 0-100

	// Physical rotation
	RotationPeriod float32 // Rotation period in hours
	AxialTilt      float32 // Axial tilt in degrees from orbital plane

	// Keplerian orbital elements
	SemiMajorAxis      float32 // Semi-major axis (AU or local units)
	Eccentricity       float32 // Orbital eccentricity (0=circle)
	Inclination        float32 // Orbital inclination in radians
	LongAscendingNode  float32 // Longitude of ascending node (Ω) in radians
	ArgPeriapsis       float32 // Argument of periapsis (ω) in radians
	MeanAnomalyAtEpoch float32 // Mean anomaly at epoch (M₀) in radians
	OrbitalPeriod      float32 // Orbital period in seconds

	// Legacy simplified orbital parameters
	OrbitRadius float32 // Circular orbital distance
	OrbitSpeed  float32 // Angular velocity in radians/second

	// Hierarchy
	ParentName string // Empty for top-level bodies; parent name for moons/rings
}

// AnimationState contains mutable per-frame 3D state.
type AnimationState struct {
	Position     Vector3 // Current position in 3D space
	Velocity     Vector3 // Current velocity vector
	OrbitCenter  Vector3 // Current orbit center (updated each frame for moons)
	MeanAnomaly  float32 // Current mean anomaly (M) in radians
	TrueAnomaly  float32 // Current true anomaly (ν) in radians
	OrbitAngle   float32 // Legacy: equals TrueAnomaly for circular orbits
	OrbitAxis    Vector3 // Axis of rotation (typically Y-up: 0,1,0)
	OrbitYOffset float32 // Vertical offset from orbital plane (asteroid belt)
}

// Object represents a simulated celestial body.
type Object struct {
	Meta    ObjectMetadata  // Immutable physical/design properties
	Anim    AnimationState  // Mutable per-frame animation data
	Visible bool            // Whether this object should be rendered
	Dataset AsteroidDataset // Which dataset this belongs to (-1 for non-asteroids)
	pooled  bool
}
