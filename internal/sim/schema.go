package sim

import (
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// Configuration structures for loading solar systems from JSON files.
// Supports template-based reusable definitions and system-specific overrides.

// SystemConfig defines a complete solar system or stellar system.
type SystemConfig struct {
	Name             string                 `json:"name"`
	Version          string                 `json:"version"`
	ScaleFactor      float32                `json:"scale_factor"`
	SecondsPerSecond float32                `json:"seconds_per_second"`
	Bodies           []BodyConfig           `json:"bodies"`
	Features         []engine.FeatureConfig `json:"features"`
	Templates        string                 `json:"templates,omitempty"`
	DefaultState     StateConfig            `json:"default_state,omitempty"`
}

// BodyConfig defines a celestial body (star, planet, moon, dwarf planet, rogue, artifact).
type BodyConfig struct {
	Type       string            `json:"type"`
	Name       string            `json:"name"`
	Subtype    string            `json:"subtype,omitempty"` // rogue/artifact subtype (e.g. "periodic", "probe")
	Parent     string            `json:"parent,omitempty"`
	Faction    string            `json:"faction,omitempty"` // artifact faction id (F-035, Phase 2)
	Template   string            `json:"template,omitempty"`
	Overrides  map[string]any    `json:"overrides,omitempty"`
	Orbit      OrbitConfig       `json:"orbit,omitempty"`
	Physical   PhysicalConfig    `json:"physical"`
	Rendering  RenderingConfig   `json:"rendering,omitempty"`
	Atmosphere *AtmosphereConfig `json:"atmosphere,omitempty"`
	Luminosity LuminosityConfig  `json:"luminosity,omitempty"`
	Importance int               `json:"importance"`
	Metadata   map[string]any    `json:"metadata,omitempty"` // artifact freeform metadata
}

// OrbitConfig defines orbital mechanics parameters.
type OrbitConfig struct {
	SemiMajorAxis          float32 `json:"semi_major_axis"`
	Eccentricity           float32 `json:"eccentricity"`
	Inclination            float32 `json:"inclination"`
	LongitudeAscendingNode float32 `json:"longitude_ascending_node,omitempty"`
	ArgumentPeriapsis      float32 `json:"argument_periapsis,omitempty"`
	OrbitalPeriod          float32 `json:"orbital_period"`
	InitialMeanAnomaly     string  `json:"initial_mean_anomaly,omitempty"`
	OrbitalSpeed           float32 `json:"orbital_speed,omitempty"`
	// Artifact-only: direct position/velocity placement bypassing Keplerian calculation.
	PositionOverride []float64 `json:"position_override,omitempty"` // [x, y, z] in sim units
	VelocityOverride []float64 `json:"velocity_override,omitempty"` // [vx, vy, vz] in sim units/s
}

// PhysicalConfig defines physical characteristics.
type PhysicalConfig struct {
	Radius           float32  `json:"radius"`
	RadiusKm         float32  `json:"radius_km,omitempty"`
	EquatorialRadius float32  `json:"equatorial_radius,omitempty"`
	PolarRadius      float32  `json:"polar_radius,omitempty"`
	InnerRadius      float32  `json:"inner_radius,omitempty"`
	Mass             float64  `json:"mass"`
	RotationPeriod   float32  `json:"rotation_period,omitempty"`
	AxialTilt        float32  `json:"axial_tilt,omitempty"`
	Color            [4]uint8 `json:"color"`
	Albedo           float32  `json:"albedo,omitempty"`
}

// RenderingConfig defines visual representation.
type RenderingConfig struct {
	Material          string     `json:"material,omitempty"`
	Texture           string     `json:"texture,omitempty"`
	TextureImage      string     `json:"texture_image,omitempty"`
	NightTextureImage string     `json:"night_texture_image,omitempty"`
	ColorMap          string     `json:"color_map,omitempty"`
	RingColorsMap     string     `json:"ring_colors_map,omitempty"`
	FallbackColor     [4]uint8   `json:"fallback_color,omitempty"`
	NormalMap         string     `json:"normal_map,omitempty"`
	SpecularMap       string     `json:"specular_map,omitempty"`
	BumpMap           string     `json:"bump_map,omitempty"`
	Shader            string     `json:"shader,omitempty"`
	LODLevels         []LODLevel `json:"lod_levels,omitempty"`
}

// LODLevel defines rendering detail at different distances.
type LODLevel struct {
	Distance  float32 `json:"distance"`
	Model     string  `json:"model,omitempty"`
	Rings     int32   `json:"rings,omitempty"`
	Slices    int32   `json:"slices,omitempty"`
	PointSize float32 `json:"point_size,omitempty"`
}

// AtmosphereConfig defines atmospheric properties for a body.
// Fields match the JSON schema used in data/systems/*.json.
type AtmosphereConfig struct {
	ThicknessKm        float32  `json:"thickness_km"`
	PrincipalChemicals []string `json:"principal_chemicals,omitempty"`
	CloudCoverage      float32  `json:"cloud_coverage,omitempty"`
	CloudComposition   []string `json:"cloud_composition,omitempty"`
	ColorHint          [4]uint8 `json:"color_hint,omitempty"`
}

// LuminosityConfig defines self-emission properties for stars and glowing bodies.
type LuminosityConfig struct {
	SelfLuminous        bool     `json:"self_luminous,omitempty"`
	SolarLuminosity     float32  `json:"solar_luminosity,omitempty"`
	SurfaceTemperatureK float32  `json:"surface_temperature_k,omitempty"`
	EmissionColor       [4]uint8 `json:"emission_color,omitempty"`
	EmissionIntensity   float32  `json:"emission_intensity,omitempty"`
}

// StateConfig defines default simulation state.
type StateConfig struct {
	SimulationHz   float64 `json:"simulation_hz,omitempty"`
	WorkerThreads  int     `json:"worker_threads,omitempty"`
	InitialDataset string  `json:"initial_dataset,omitempty"`
}

// TemplateLibrary contains reusable body templates.
type TemplateLibrary struct {
	Stars        map[string]BodyConfig           `json:"stars,omitempty"`
	Planets      map[string]BodyConfig           `json:"planets,omitempty"`
	DwarfPlanets map[string]BodyConfig           `json:"dwarf_planets,omitempty"`
	Moons        map[string]BodyConfig           `json:"moons,omitempty"`
	Asteroids    map[string]BodyConfig           `json:"asteroids,omitempty"`
	Features     map[string]engine.FeatureConfig `json:"features,omitempty"`
}

// BeltConfig encapsulates all parameters for belt generation.
type BeltConfig struct {
	Name                     string
	NamePrefix               string
	InnerRadius              float32
	OuterRadius              float32
	Thickness                float32
	ClassicalBeltMin         float32
	ClassicalBeltMax         float32
	ClassicalBeltProbability float32
	DistanceToAURatio        float32
	UseKeplerLaw             bool
	EccentricityMin          float32
	EccentricityMax          float32
	InclinationMin           float32
	InclinationMax           float32
	ObjectTypes              map[string]BeltObjectTypeConfig
	ColorPalette             [][4]uint8
	ColorVariation           float32
	Seed                     int64
}

// BeltObjectTypeConfig defines a specific object type within a belt.
type BeltObjectTypeConfig struct {
	Count      int
	SizeMin    float32
	SizeMax    float32
	Importance int
}

// ── F-034 System Directory Format ──────────────────────────────────────────

// SystemManifest is the top-level manifest parsed from a system directory's
// system.json file (schema_version "2.0").
type SystemManifest struct {
	Name          string                 `json:"name"`
	SystemVersion string                 `json:"system_version"`
	SchemaVersion string                 `json:"schema_version"`
	ScaleFactor   float32                `json:"scale_factor"`
	Simulation    SystemSimulationConfig `json:"simulation,omitempty"`
	Files         SystemFilesManifest    `json:"files,omitempty"`
}

// SystemSimulationConfig holds per-system simulation parameters from system.json.
type SystemSimulationConfig struct {
	DefaultTimeScale float32 `json:"default_time_scale,omitempty"`
	NBodyMode        string  `json:"nbody_mode,omitempty"` // "keplerian" (default) or "nbody"
}

// SystemFilesManifest maps type keys to sub-file names within a system directory.
type SystemFilesManifest struct {
	Stars        string `json:"stars,omitempty"`
	Planets      string `json:"planets,omitempty"`
	DwarfPlanets string `json:"dwarf_planets,omitempty"`
	Moons        string `json:"moons,omitempty"`
	Belts        string `json:"belts,omitempty"`
	Rogues       string `json:"rogues,omitempty"`
	Artifacts    string `json:"artifacts,omitempty"`
}

// TypedBodiesFile is the container for per-type body files (stars.json,
// planets.json, dwarf_planets.json, moons.json).
type TypedBodiesFile struct {
	SchemaVersion string       `json:"schema_version"`
	Bodies        []BodyConfig `json:"bodies"`
}

// TypedBeltsFile is the container for belts.json in a system directory.
type TypedBeltsFile struct {
	SchemaVersion string                 `json:"schema_version"`
	Belts         []engine.FeatureConfig `json:"belts"`
}

// RoguesFile is the container for rogues.json — comets, interstellar objects,
// and other high-eccentricity natural bodies.
type RoguesFile struct {
	SchemaVersion string       `json:"schema_version"`
	Rogues        []BodyConfig `json:"rogues"`
}

// ArtifactsFile is the container for artifacts.json — human-made or alien-made
// durable objects (absorbs F-008).
type ArtifactsFile struct {
	SchemaVersion string       `json:"schema_version"`
	Artifacts     []BodyConfig `json:"artifacts"`
}

// supportedSchemaVersions is the set of schema_version values accepted by the
// directory loader.  Add new versions here when the per-file schema changes.
var supportedSchemaVersions = map[string]bool{
	"2.0": true,
}
