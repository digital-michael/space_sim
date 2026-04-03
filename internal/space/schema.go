package space

import (
	"github.com/digital-michael/space_sim/internal/space/engine"
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

// BodyConfig defines a celestial body (star, planet, moon, dwarf planet).
type BodyConfig struct {
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Parent     string          `json:"parent,omitempty"`
	Template   string          `json:"template,omitempty"`
	Overrides  map[string]any  `json:"overrides,omitempty"`
	Orbit      OrbitConfig     `json:"orbit,omitempty"`
	Physical   PhysicalConfig  `json:"physical"`
	Rendering  RenderingConfig `json:"rendering,omitempty"`
	Importance int             `json:"importance"`
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
}

// PhysicalConfig defines physical characteristics.
type PhysicalConfig struct {
	Radius         float32  `json:"radius"`
	InnerRadius    float32  `json:"inner_radius,omitempty"`
	Mass           float64  `json:"mass"`
	RotationPeriod float32  `json:"rotation_period,omitempty"`
	AxialTilt      float32  `json:"axial_tilt,omitempty"`
	Color          [4]uint8 `json:"color"`
	Albedo         float32  `json:"albedo,omitempty"`
}

// RenderingConfig defines visual representation.
type RenderingConfig struct {
	Material    string            `json:"material,omitempty"`
	Texture     string            `json:"texture,omitempty"`
	NormalMap   string            `json:"normal_map,omitempty"`
	SpecularMap string            `json:"specular_map,omitempty"`
	BumpMap     string            `json:"bump_map,omitempty"`
	Shader      string            `json:"shader,omitempty"`
	LODLevels   []LODLevel        `json:"lod_levels,omitempty"`
	Atmosphere  *AtmosphereConfig `json:"atmosphere,omitempty"`
}

// LODLevel defines rendering detail at different distances.
type LODLevel struct {
	Distance  float32 `json:"distance"`
	Model     string  `json:"model,omitempty"`
	Rings     int32   `json:"rings,omitempty"`
	Slices    int32   `json:"slices,omitempty"`
	PointSize float32 `json:"point_size,omitempty"`
}

// AtmosphereConfig defines atmospheric rendering.
type AtmosphereConfig struct {
	Enabled    bool     `json:"enabled"`
	Color      [4]uint8 `json:"color"`
	Thickness  float32  `json:"thickness"`
	Density    float32  `json:"density"`
	Scattering bool     `json:"scattering,omitempty"`
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
