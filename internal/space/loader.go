package space

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/digital-michael/space_sim/internal/space/engine"
)

// LoadSystemFromFile loads a solar system configuration from a JSON file.
func LoadSystemFromFile(path string) (*engine.SimulationState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read system config: %w", err)
	}

	var config SystemConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse system config: %w", err)
	}

	var templates *TemplateLibrary
	if config.Templates != "" {
		templates, err = LoadTemplateLibrary(config.Templates)
		if err != nil {
			return nil, fmt.Errorf("failed to load templates: %w", err)
		}
	}

	state := engine.NewSimulationState()

	if config.DefaultState.WorkerThreads > 0 {
		state.NumWorkers = config.DefaultState.WorkerThreads
	}

	rng := rand.New(rand.NewSource(42))

	seenCategories := make(map[engine.ObjectCategory]bool)

	for _, bodyConfig := range config.Bodies {
		obj, err := createBodyFromConfig(bodyConfig, templates, rng, config.ScaleFactor)
		if err != nil {
			return nil, fmt.Errorf("failed to create body %s: %w", bodyConfig.Name, err)
		}
		state.AddObject(obj)
		seenCategories[obj.Meta.Category] = true
	}

	for _, featureConfig := range config.Features {
		if strings.ToLower(featureConfig.Type) == "asteroid_belt" {
			configCopy := featureConfig
			state.AsteroidBeltConfig = &configCopy
			seenCategories[engine.CategoryBelt] = true
		} else if strings.ToLower(featureConfig.Type) == "kuiper_belt" {
			configCopy := featureConfig
			state.KuiperBeltConfig = &configCopy
			seenCategories[engine.CategoryBelt] = true
		}

		if err := createFeatureFromConfig(state, featureConfig, templates, rng); err != nil {
			return nil, fmt.Errorf("failed to create feature %s: %w", featureConfig.Name, err)
		}
	}

	canonicalOrder := []engine.ObjectCategory{
		engine.CategoryStar,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryRing,
		engine.CategoryBelt,
	}
	for _, cat := range canonicalOrder {
		if seenCategories[cat] {
			state.NavigationOrder = append(state.NavigationOrder, cat)
		}
	}

	return state, nil
}

// LoadTemplateLibrary loads body templates from a JSON file or directory.
func LoadTemplateLibrary(path string) (*TemplateLibrary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	library := &TemplateLibrary{
		Stars:        make(map[string]BodyConfig),
		Planets:      make(map[string]BodyConfig),
		DwarfPlanets: make(map[string]BodyConfig),
		Moons:        make(map[string]BodyConfig),
		Asteroids:    make(map[string]BodyConfig),
		Features:     make(map[string]engine.FeatureConfig),
	}

	if info.IsDir() {
		files, err := filepath.Glob(filepath.Join(path, "*.json"))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if err := loadTemplateFile(file, library); err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", file, err)
			}
		}
	} else {
		if err := loadTemplateFile(path, library); err != nil {
			return nil, err
		}
	}

	return library, nil
}

func loadTemplateFile(path string, library *TemplateLibrary) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var temp TemplateLibrary
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	for k, v := range temp.Stars {
		library.Stars[k] = v
	}
	for k, v := range temp.Planets {
		library.Planets[k] = v
	}
	for k, v := range temp.DwarfPlanets {
		library.DwarfPlanets[k] = v
	}
	for k, v := range temp.Moons {
		library.Moons[k] = v
	}
	for k, v := range temp.Asteroids {
		library.Asteroids[k] = v
	}
	for k, v := range temp.Features {
		library.Features[k] = v
	}

	return nil
}

func createBodyFromConfig(config BodyConfig, templates *TemplateLibrary, rng *rand.Rand, scaleFactor float32) (*engine.Object, error) {
	if config.Template != "" && templates != nil {
		var templateConfig BodyConfig
		var found bool

		switch strings.ToLower(config.Type) {
		case "star":
			templateConfig, found = templates.Stars[config.Template]
		case "planet":
			templateConfig, found = templates.Planets[config.Template]
		case "dwarf_planet":
			templateConfig, found = templates.DwarfPlanets[config.Template]
		case "moon":
			templateConfig, found = templates.Moons[config.Template]
		case "asteroid":
			templateConfig, found = templates.Asteroids[config.Template]
		}

		if !found {
			return nil, fmt.Errorf("template %s not found for type %s", config.Template, config.Type)
		}
		config = applyOverrides(templateConfig, config)
	}

	material := engine.MaterialDiffuse
	if config.Rendering.Material != "" {
		switch strings.ToLower(config.Rendering.Material) {
		case "emissive":
			material = engine.MaterialEmissive
		case "metallic":
			material = engine.MaterialMetallic
		case "mirror":
			material = engine.MaterialMirror
		}
	}

	category := engine.CategoryPlanet
	switch strings.ToLower(config.Type) {
	case "star":
		category = engine.CategoryStar
	case "dwarf_planet":
		category = engine.CategoryDwarfPlanet
	case "moon":
		category = engine.CategoryMoon
	case "asteroid":
		category = engine.CategoryAsteroid
	case "ring":
		category = engine.CategoryRing
	}

	initialAngle := float32(0)
	if config.Orbit.InitialMeanAnomaly == "random" {
		initialAngle = float32(rng.Float64()) * 2 * math.Pi
	} else if config.Orbit.InitialMeanAnomaly != "" {
		fmt.Sscanf(config.Orbit.InitialMeanAnomaly, "%f", &initialAngle)
	}

	orbitalSpeed := config.Orbit.OrbitalSpeed
	orbitalPeriod := float32(0)
	if config.Orbit.OrbitalPeriod > 0 {
		orbitalPeriod = config.Orbit.OrbitalPeriod * 86400
		if orbitalSpeed == 0 {
			orbitalSpeed = float32(2 * math.Pi / float64(orbitalPeriod))
		}
	} else if orbitalSpeed > 0 {
		orbitalPeriod = float32(2 * math.Pi / float64(orbitalSpeed))
	}

	initialX := config.Orbit.SemiMajorAxis
	initialZ := float32(0)
	if config.Orbit.Eccentricity > 0 {
		r := config.Orbit.SemiMajorAxis * (1 - config.Orbit.Eccentricity)
		initialX = r * float32(math.Cos(float64(initialAngle)))
		initialZ = r * float32(math.Sin(float64(initialAngle)))
	} else {
		initialX = config.Orbit.SemiMajorAxis * float32(math.Cos(float64(initialAngle)))
		initialZ = config.Orbit.SemiMajorAxis * float32(math.Sin(float64(initialAngle)))
	}

	obj := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           config.Name,
			Category:       category,
			Mass:           config.Physical.Mass,
			PhysicalRadius: config.Physical.Radius,
			InnerRadius:    config.Physical.InnerRadius,
			Color: engine.Color{
				R: config.Physical.Color[0],
				G: config.Physical.Color[1],
				B: config.Physical.Color[2],
				A: config.Physical.Color[3],
			},
			Material:   material,
			Importance: config.Importance,

			RotationPeriod: config.Physical.RotationPeriod,
			AxialTilt:      config.Physical.AxialTilt,

			SemiMajorAxis:      config.Orbit.SemiMajorAxis,
			Eccentricity:       config.Orbit.Eccentricity,
			Inclination:        config.Orbit.Inclination,
			LongAscendingNode:  config.Orbit.LongitudeAscendingNode,
			ArgPeriapsis:       config.Orbit.ArgumentPeriapsis,
			MeanAnomalyAtEpoch: initialAngle,
			OrbitalPeriod:      orbitalPeriod,

			OrbitRadius: config.Orbit.SemiMajorAxis,
			OrbitSpeed:  orbitalSpeed,
			ParentName:  config.Parent,
		},
		Anim: engine.AnimationState{
			Position:     engine.Vector3{X: initialX, Y: 0, Z: initialZ},
			Velocity:     engine.Vector3{},
			OrbitCenter:  engine.Vector3{X: 0, Y: 0, Z: 0},
			MeanAnomaly:  initialAngle,
			TrueAnomaly:  initialAngle,
			OrbitAngle:   initialAngle,
			OrbitAxis:    engine.Vector3{X: 0, Y: 1, Z: 0},
			OrbitYOffset: 0,
		},
		Visible: true,
		Dataset: -1,
	}

	return obj, nil
}

func applyOverrides(template BodyConfig, overrides BodyConfig) BodyConfig {
	result := template

	if overrides.Name != "" {
		result.Name = overrides.Name
	}
	if overrides.Parent != "" {
		result.Parent = overrides.Parent
	}
	if overrides.Importance != 0 {
		result.Importance = overrides.Importance
	}
	if overrides.Orbit.SemiMajorAxis != 0 {
		result.Orbit.SemiMajorAxis = overrides.Orbit.SemiMajorAxis
	}
	if overrides.Orbit.Eccentricity != 0 {
		result.Orbit.Eccentricity = overrides.Orbit.Eccentricity
	}
	if overrides.Orbit.OrbitalPeriod != 0 {
		result.Orbit.OrbitalPeriod = overrides.Orbit.OrbitalPeriod
	}
	if overrides.Physical.Radius != 0 {
		result.Physical.Radius = overrides.Physical.Radius
	}
	if overrides.Physical.Mass != 0 {
		result.Physical.Mass = overrides.Physical.Mass
	}
	if overrides.Physical.Color[3] != 0 {
		result.Physical.Color = overrides.Physical.Color
	}
	if overrides.Rendering.Material != "" {
		result.Rendering.Material = overrides.Rendering.Material
	}
	if overrides.Rendering.Texture != "" {
		result.Rendering.Texture = overrides.Rendering.Texture
	}

	return result
}

func createFeatureFromConfig(state *engine.SimulationState, config engine.FeatureConfig, templates *TemplateLibrary, rng *rand.Rand) error {
	switch strings.ToLower(config.Type) {
	case "asteroid_belt", "kuiper_belt":
		return createBeltFromConfig(state, config, rng)
	case "ring_system":
		return createRingSystemFromConfig(state, config)
	default:
		return fmt.Errorf("unsupported feature type: %s", config.Type)
	}
}

func createBeltFromConfig(state *engine.SimulationState, config engine.FeatureConfig, rng *rand.Rand) error {
	datasetLevel := engine.AsteroidDatasetSmall

	classicalMin := float32(0)
	classicalMax := float32(0)
	classicalProb := float32(0)
	if len(config.Distribution.ClassicalBeltRange) == 2 {
		classicalMin = config.Distribution.ClassicalBeltRange[0]
		classicalMax = config.Distribution.ClassicalBeltRange[1]
		classicalProb = config.Distribution.ClassicalBeltProbability
	}

	namePrefix := "Belt-"
	if strings.ToLower(config.Type) == "asteroid_belt" {
		namePrefix = "Asteroid-"
	} else if strings.ToLower(config.Type) == "kuiper_belt" {
		namePrefix = "KBO-"
	}

	objectTypes := make(map[string]BeltObjectTypeConfig)
	for typeName, typeSpec := range config.ObjectTypes {
		if int(datasetLevel) < len(typeSpec.CountByLevel) {
			count := typeSpec.CountByLevel[int(datasetLevel)]
			importance := 10
			if int(datasetLevel) < len(typeSpec.ImportanceByLevel) {
				importance = typeSpec.ImportanceByLevel[int(datasetLevel)]
			}
			objectTypes[typeName] = BeltObjectTypeConfig{
				Count:      count,
				SizeMin:    typeSpec.SizeRange[0],
				SizeMax:    typeSpec.SizeRange[1],
				Importance: importance,
			}
		}
	}

	distanceToAU := float32(100.0)
	if config.OrbitalMechanics.DistanceToAURatio > 0 {
		distanceToAU = config.OrbitalMechanics.DistanceToAURatio
	}

	beltConfig := BeltConfig{
		Name:                     config.Name,
		NamePrefix:               namePrefix,
		InnerRadius:              config.Distribution.InnerRadius,
		OuterRadius:              config.Distribution.OuterRadius,
		Thickness:                config.Distribution.Thickness,
		ClassicalBeltMin:         classicalMin,
		ClassicalBeltMax:         classicalMax,
		ClassicalBeltProbability: classicalProb,
		DistanceToAURatio:        distanceToAU,
		UseKeplerLaw:             config.OrbitalMechanics.UseKeplerLaw,
		EccentricityMin:          config.OrbitalMechanics.EccentricityRange[0],
		EccentricityMax:          config.OrbitalMechanics.EccentricityRange[1],
		InclinationMin:           config.OrbitalMechanics.InclinationRange[0],
		InclinationMax:           config.OrbitalMechanics.InclinationRange[1],
		ObjectTypes:              objectTypes,
		ColorPalette:             config.Procedural.ColorPalette,
		ColorVariation:           config.Procedural.ColorVariation,
		Seed:                     config.Procedural.Seed,
	}

	CreateBelt(state, beltConfig, datasetLevel, rng)

	state.CurrentDataset = datasetLevel
	state.AllocatedDatasets[datasetLevel] = true

	return nil
}

func createRingSystemFromConfig(state *engine.SimulationState, config engine.FeatureConfig) error {
	return nil
}
