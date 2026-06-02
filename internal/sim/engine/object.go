package engine

// MaterialType defines the rendering material.
type MaterialType int

const (
	MaterialDiffuse        MaterialType = iota // Matte surface
	MaterialEmissive                           // Glowing (sun)
	MaterialMetallic                           // Shiny metal (asteroids)
	MaterialMirror                             // Reflective (satellites)
	MaterialDiffuseThermal                     // Thermally self-luminous but not a star (gas giants, volcanic Io)
	MaterialBlackHole                          // Light absorber; body draws black; photon ring / accretion disk are separate features
	MaterialNeutronStar                        // Ultra-compact high-energy emitter (neutron star, pulsar, magnetar)
)

// ObjectCategory defines object grouping for UI navigation.
// Values are generic — the loader maps JSON "type" strings onto them.
type ObjectCategory int

const (
	// Stellar categories (0–9); slots 5–9 reserved for future stellar additions.
	CategoryStarPreMain      ObjectCategory = 0 // Protostar, T Tauri star
	CategoryStarMainSequence ObjectCategory = 1 // Red Dwarf (M) through Blue (O)
	CategoryStarEvolved      ObjectCategory = 2 // Giants, Supergiants, Wolf-Rayet
	CategorySubstellar       ObjectCategory = 3 // Brown Dwarf (L/T/Y); failed stars
	CategoryStellarRemnant   ObjectCategory = 4 // White Dwarf, Neutron Star, Black Hole

	// System body categories (10–19); slots 17–19 reserved for future additions.
	CategoryPlanet      ObjectCategory = 10
	CategoryDwarfPlanet ObjectCategory = 11
	CategoryMoon        ObjectCategory = 12
	CategoryAsteroid    ObjectCategory = 13
	CategoryBelt        ObjectCategory = 14 // Virtual: asteroid/Kuiper belts
	CategoryRogue       ObjectCategory = 15 // Comets, rogue planets, interstellar objects
	CategoryArtifact    ObjectCategory = 16 // Human-made or alien-made durable objects (F-008/F-034)

	// Internal sentinel — never shown as a UI tab.
	CategoryRing ObjectCategory = 99 // Ring systems; excluded from navigation order
)

// IsStarLike reports whether a category belongs to the stellar group
// (pre-main, main sequence, evolved, substellar, or stellar remnant).
func IsStarLike(cat ObjectCategory) bool {
	return cat >= CategoryStarPreMain && cat <= CategoryStellarRemnant
}

// ViewRadius returns the effective radius used for camera auto-zoom distance
// calculations. For black holes, using PhysicalRadius directly places the camera
// inside the accretion disk (outer edge = 7×radius). A multiplier of 1.5 targets
// a camera distance of ~9×radius — just outside the disk outer edge — giving a
// clear view of the disk structure without being embedded in it or too far away.
// For all other bodies it is PhysicalRadius unchanged.
func (m *ObjectMetadata) ViewRadius() float32 {
	if m.Material == MaterialBlackHole {
		return m.PhysicalRadius * 1.5
	}
	return m.PhysicalRadius
}

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
	Name             string         // Display name
	Category         ObjectCategory // Object category for UI grouping
	Mass             float64        // Mass in kilograms
	PhysicalRadius   float32        // Physical size (or outer radius for rings); equatorial radius when oblate
	PhysicalRadiusKm float32        // Real-world equatorial radius in km; 0 = derive from Earth-calibrated km/su
	EquatorialRadius float32        // Equatorial radius for oblate bodies; 0 means use PhysicalRadius uniformly
	PolarRadius      float32        // Polar radius for oblate bodies; 0 means use PhysicalRadius uniformly
	InnerRadius      float32        // Inner radius (rings only; 0 for spheres)
	Color            Color          // Display color (fallback when no texture)
	Material         MaterialType   // Rendering material
	Importance       int            // Rendering priority 0-100

	// Texture and surface
	TexturePath      string  // Path to diffuse texture image; empty means use Color
	NightTexturePath string  // Path to night-side emission texture (city lights); empty means no night glow
	Albedo           float32 // Surface albedo 0–1; used for lighting calculations

	// Luminosity (stars and thermally glowing bodies)
	SelfLuminous        bool    // True if the body emits its own light
	SolarLuminosity     float32 // Luminosity relative to Sol (1.0 = Sol)
	SurfaceTemperatureK float32 // Surface or effective temperature in Kelvin
	EmissionColor       Color   // Dominant emission color for light tinting

	// Atmosphere
	AtmosphereColorHint   Color   // Tint color for atmosphere overlay rendering
	AtmosphereThicknessKm float32 // Nominal atmosphere thickness in km (0 = no atmosphere)
	CloudCoverage         float32 // 0–1 fraction of surface covered by clouds

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

	// Stellar classification (optional). Used by the Advanced Info display toggle.
	// SpectralClass: spectral or luminosity class label shown in parentheses
	//   e.g. "M", "G", "K", "III", "L/T/Y", "Ia"
	// StellarVariant: named variant within a category shown in parentheses
	//   e.g. "Pulsar", "Magnetar", "White Dwarf", "Black Hole", "Quasar"
	SpectralClass  string
	StellarVariant string

	// Relativistic jet parameters (set by relativistic_jet feature; 0 = no jets)
	JetLength float32 // Total length of each jet arm in sim units
	JetRadius float32 // Cone base radius at body surface in sim units
	JetColor  Color   // Jet emission color

	// Accretion disk / visual parameters (MaterialBlackHole and MaterialNeutronStar).
	// DiskTilt: disk tilt in degrees; 0 = derive from AxialTilt, fallback 30°.
	// AccretionEnergy: 0 = default (1.0); scales disk brightness
	//   (e.g. 0.05 = quiescent, 1.0 = standard, 2.5 = quasar-level).
	DiskTilt        float32
	AccretionEnergy float32

	// PulsePeriod: rotation period for pulsar lighthouse effect in seconds.
	// 0 = no pulse. Applies to MaterialNeutronStar only.
	PulsePeriod float32

	// Hierarchy
	ParentName string // Empty for top-level bodies; parent name for moons/rings

	// N-body gravitational parameters (computed at load time; see F-013).
	GM         float64 // G_sim × Mass in sim³/s²
	HillRadius float64 // Hill sphere radius in sim units (0 = system default)
	LaplaceSOI float64 // Laplace SOI radius in sim units (0 = use HillRadius)
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

	// N-body integration state (float64 for long-term precision; see F-013).
	// Renderer reads float32 Position above; N-body writes here.
	NBodyPos [3]float64 // Inertial-frame position in sim units
	NBodyVel [3]float64 // Inertial-frame velocity in sim units/s
	NBodyAcc [3]float64 // Current gravitational acceleration in sim units/s²
}

// Object represents a simulated celestial body.
type Object struct {
	Meta    ObjectMetadata  // Immutable physical/design properties
	Anim    AnimationState  // Mutable per-frame animation data
	Visible bool            // Whether this object should be rendered
	Dataset AsteroidDataset // Which dataset this belongs to (-1 for non-asteroids)
	pooled  bool
}
