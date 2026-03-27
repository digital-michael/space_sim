package engine

// FeatureConfig defines a system-wide feature (asteroid belt, Kuiper belt, ring
// system, etc.) loaded from JSON. Lives in engine so SimulationState can hold
// typed pointers without an import cycle.
type FeatureConfig struct {
	Type             string                    `json:"type"`
	Name             string                    `json:"name"`
	Parent           string                    `json:"parent"`
	Template         string                    `json:"template,omitempty"`
	Overrides        map[string]any            `json:"overrides,omitempty"`
	Distribution     FeatureDistribution       `json:"distribution"`
	Procedural       FeatureProcedural         `json:"procedural,omitempty"`
	CountLevels      []int                     `json:"count_levels,omitempty"`
	ObjectTypes      map[string]FeatureObjSpec `json:"object_types,omitempty"`
	OrbitalMechanics FeatureOrbitalMechanics   `json:"orbital_mechanics,omitempty"`
}

// FeatureDistribution defines spatial distribution of a feature.
type FeatureDistribution struct {
	InnerRadius              float32    `json:"inner_radius"`
	OuterRadius              float32    `json:"outer_radius"`
	Thickness                float32    `json:"thickness,omitempty"`
	DensityProfile           string     `json:"density_profile,omitempty"`
	Inclination              float32    `json:"inclination,omitempty"`
	ClassicalBeltRange       [2]float32 `json:"classical_belt_range,omitempty"`
	ClassicalBeltProbability float32    `json:"classical_belt_probability,omitempty"`
}

// FeatureProcedural defines procedural generation parameters for a feature.
type FeatureProcedural struct {
	Seed           int64      `json:"seed"`
	SizeRange      [2]float32 `json:"size_range"`
	ColorPalette   [][4]uint8 `json:"color_palette,omitempty"`
	ColorVariation float32    `json:"color_variation,omitempty"`
	ShapeModel     string     `json:"shape_model,omitempty"`
	ShapeSeed      int64      `json:"shape_seed,omitempty"`
}

// FeatureObjSpec defines a class of objects within a belt.
type FeatureObjSpec struct {
	CountByLevel      []int      `json:"count_by_level"`
	SizeRange         [2]float32 `json:"size_range"`
	ImportanceByLevel []int      `json:"importance_by_level"`
	Description       string     `json:"description"`
}

// FeatureOrbitalMechanics defines orbital parameters for belt objects.
type FeatureOrbitalMechanics struct {
	UseKeplerLaw      bool       `json:"use_kepler_law"`
	DistanceToAURatio float32    `json:"distance_to_au_ratio"`
	EccentricityRange [2]float32 `json:"eccentricity_range"`
	InclinationRange  [2]float32 `json:"inclination_range"`
}
