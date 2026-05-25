package sim

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// resolveColor returns the first color with a non-zero alpha, converting [4]uint8 to engine.Color.
// Prefers rendering.fallback_color; falls back to physical.color for backward compatibility.
func resolveColor(primary, secondary [4]uint8) engine.Color {
	c := primary
	if c[3] == 0 {
		c = secondary
	}
	if c[3] == 0 {
		// neutral grey so the body is at least visible
		c = [4]uint8{200, 200, 200, 255}
	}
	return engine.Color{R: c[0], G: c[1], B: c[2], A: c[3]}
}

func resolveAtmosphereColor(a *AtmosphereConfig) engine.Color {
	if a == nil || a.ColorHint[3] == 0 {
		return engine.Color{}
	}
	return engine.Color{R: a.ColorHint[0], G: a.ColorHint[1], B: a.ColorHint[2], A: a.ColorHint[3]}
}

func resolveAtmosphereThickness(a *AtmosphereConfig) float32 {
	if a == nil {
		return 0
	}
	return a.ThicknessKm
}

func resolveCloudCoverage(a *AtmosphereConfig) float32 {
	if a == nil {
		return 0
	}
	return a.CloudCoverage
}

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
		} else if strings.ToLower(featureConfig.Type) == "ring_system" ||
			strings.ToLower(featureConfig.Type) == "photon_ring" ||
			strings.ToLower(featureConfig.Type) == "accretion_disk" {
			seenCategories[engine.CategoryRing] = true
		}

		if err := createFeatureFromConfig(state, featureConfig, templates, rng); err != nil {
			return nil, fmt.Errorf("failed to create feature %s: %w", featureConfig.Name, err)
		}
	}

	canonicalOrder := []engine.ObjectCategory{engine.CategoryBlackHole, engine.CategoryStar,
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

	computeSOIRadii(state)
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
		case "diffuse_thermal":
			material = engine.MaterialDiffuseThermal
		case "blackhole":
			material = engine.MaterialBlackHole
		case "neutron_star":
			material = engine.MaterialNeutronStar
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
	case "rogue", "comet":
		category = engine.CategoryRogue
	case "artifact":
		category = engine.CategoryArtifact
	case "blackhole":
		category = engine.CategoryBlackHole
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
			Name:             config.Name,
			Category:         category,
			Subtype:          config.Subtype,
			Mass:             config.Physical.Mass,
			GM:               engine.G_sim * config.Physical.Mass,
			PhysicalRadius:   config.Physical.Radius,
			PhysicalRadiusKm: config.Physical.RadiusKm,
			EquatorialRadius: config.Physical.EquatorialRadius,
			PolarRadius:      config.Physical.PolarRadius,
			InnerRadius:      config.Physical.InnerRadius,
			Color:            resolveColor(config.Rendering.FallbackColor, config.Physical.Color),
			Material:         material,
			Importance:       config.Importance,

			// Texture and surface
			TexturePath:      config.Rendering.TextureImage,
			NightTexturePath: config.Rendering.NightTextureImage,
			Albedo:           config.Physical.Albedo,

			// Luminosity
			SelfLuminous:        config.Luminosity.SelfLuminous,
			SolarLuminosity:     config.Luminosity.SolarLuminosity,
			SurfaceTemperatureK: config.Luminosity.SurfaceTemperatureK,
			EmissionColor:       resolveColor(config.Luminosity.EmissionColor, config.Luminosity.EmissionColor),

			// Atmosphere
			AtmosphereColorHint:   resolveAtmosphereColor(config.Atmosphere),
			AtmosphereThicknessKm: resolveAtmosphereThickness(config.Atmosphere),
			CloudCoverage:         resolveCloudCoverage(config.Atmosphere),

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

	// Artifact position_override: direct coordinate placement bypasses Keplerian calc.
	if len(config.Orbit.PositionOverride) == 3 {
		obj.Anim.Position = engine.Vector3{
			X: float32(config.Orbit.PositionOverride[0]),
			Y: float32(config.Orbit.PositionOverride[1]),
			Z: float32(config.Orbit.PositionOverride[2]),
		}
	}
	if len(config.Orbit.VelocityOverride) == 3 {
		obj.Anim.Velocity = engine.Vector3{
			X: float32(config.Orbit.VelocityOverride[0]),
			Y: float32(config.Orbit.VelocityOverride[1]),
			Z: float32(config.Orbit.VelocityOverride[2]),
		}
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
	if overrides.Physical.EquatorialRadius != 0 {
		result.Physical.EquatorialRadius = overrides.Physical.EquatorialRadius
	}
	if overrides.Physical.PolarRadius != 0 {
		result.Physical.PolarRadius = overrides.Physical.PolarRadius
	}
	if overrides.Physical.Mass != 0 {
		result.Physical.Mass = overrides.Physical.Mass
	}
	if overrides.Rendering.FallbackColor[3] != 0 {
		result.Rendering.FallbackColor = overrides.Rendering.FallbackColor
	} else if overrides.Physical.Color[3] != 0 {
		result.Physical.Color = overrides.Physical.Color
	}
	if overrides.Rendering.Material != "" {
		result.Rendering.Material = overrides.Rendering.Material
	}
	if overrides.Rendering.Texture != "" {
		result.Rendering.Texture = overrides.Rendering.Texture
	}
	if overrides.Rendering.TextureImage != "" {
		result.Rendering.TextureImage = overrides.Rendering.TextureImage
	}
	if overrides.Rendering.NightTextureImage != "" {
		result.Rendering.NightTextureImage = overrides.Rendering.NightTextureImage
	}
	if overrides.Physical.Albedo != 0 {
		result.Physical.Albedo = overrides.Physical.Albedo
	}
	if overrides.Atmosphere != nil {
		result.Atmosphere = overrides.Atmosphere
	}
	if overrides.Luminosity.SelfLuminous {
		result.Luminosity = overrides.Luminosity
	}

	return result
}

func createFeatureFromConfig(state *engine.SimulationState, config engine.FeatureConfig, templates *TemplateLibrary, rng *rand.Rand) error {
	switch strings.ToLower(config.Type) {
	case "asteroid_belt", "kuiper_belt":
		return createBeltFromConfig(state, config, rng)
	case "ring_system", "photon_ring", "accretion_disk":
		return createRingSystemFromConfig(state, config)
	case "relativistic_jet":
		return createRelativistcJetFromConfig(state, config)
	default:
		return fmt.Errorf("unsupported feature type: %s", config.Type)
	}
}

func parseFeatureMaterial(material string) (engine.MaterialType, error) {
	switch strings.ToLower(material) {
	case "", "diffuse":
		return engine.MaterialDiffuse, nil
	case "emissive":
		return engine.MaterialEmissive, nil
	case "metallic":
		return engine.MaterialMetallic, nil
	case "mirror":
		return engine.MaterialMirror, nil
	default:
		return engine.MaterialDiffuse, fmt.Errorf("unsupported ring material %q", material)
	}
}

// createRelativistcJetFromConfig attaches jet parameters to the named parent
// body. Jets are rendered by drawObject when JetLength > 0.
// JSON fields used:
//   - parent: name of the star or black hole body
//   - distribution.outer_radius: total jet arm length in sim units
//   - distribution.thickness:    jet cone base radius at the body surface
//   - physical.color:            jet emission color (RGBA)
func createRelativistcJetFromConfig(state *engine.SimulationState, config engine.FeatureConfig) error {
	if strings.TrimSpace(config.Parent) == "" {
		return fmt.Errorf("relativistic_jet %q is missing parent", config.Name)
	}
	parent := state.GetObject(config.Parent)
	if parent == nil {
		return fmt.Errorf("relativistic_jet %q parent %q not found", config.Name, config.Parent)
	}
	if config.Distribution.OuterRadius <= 0 {
		return fmt.Errorf("relativistic_jet %q must define distribution.outer_radius > 0 (jet length)", config.Name)
	}
	parent.Meta.JetLength = config.Distribution.OuterRadius
	parent.Meta.JetRadius = config.Distribution.Thickness
	if parent.Meta.JetRadius <= 0 {
		parent.Meta.JetRadius = parent.Meta.PhysicalRadius * 0.3
	}
	if config.Physical.Color[3] != 0 {
		parent.Meta.JetColor = engine.Color{
			R: config.Physical.Color[0],
			G: config.Physical.Color[1],
			B: config.Physical.Color[2],
			A: config.Physical.Color[3],
		}
	} else {
		parent.Meta.JetColor = engine.Color{R: 140, G: 200, B: 255, A: 180}
	}
	return nil
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
	if strings.TrimSpace(config.Parent) == "" {
		return fmt.Errorf("ring system %q is missing parent", config.Name)
	}
	if config.Distribution.OuterRadius <= 0 {
		return fmt.Errorf("ring system %q must define distribution.outer_radius > 0", config.Name)
	}
	if config.Distribution.InnerRadius <= 0 {
		return fmt.Errorf("ring system %q must define distribution.inner_radius > 0", config.Name)
	}
	if config.Distribution.InnerRadius >= config.Distribution.OuterRadius {
		return fmt.Errorf("ring system %q must define inner_radius < outer_radius", config.Name)
	}

	parent := state.GetObject(config.Parent)
	if parent == nil {
		return fmt.Errorf("ring system %q parent %q not found", config.Name, config.Parent)
	}

	material, err := parseFeatureMaterial(config.Rendering.Material)
	if err != nil {
		return err
	}

	importance := config.Importance
	if importance == 0 {
		importance = 30
	}

	mass := config.Physical.Mass
	if mass == 0 {
		mass = 1.0e19
	}

	color := parent.Meta.Color
	color.A = 180
	if len(config.Procedural.ColorPalette) > 0 {
		base := config.Procedural.ColorPalette[0]
		color = engine.Color{R: base[0], G: base[1], B: base[2], A: base[3]}
	}
	if config.Physical.Color[3] != 0 {
		color = engine.Color{
			R: config.Physical.Color[0],
			G: config.Physical.Color[1],
			B: config.Physical.Color[2],
			A: config.Physical.Color[3],
		}
	}
	if config.Rendering.FallbackColor[3] != 0 {
		c := config.Rendering.FallbackColor
		color = engine.Color{R: c[0], G: c[1], B: c[2], A: c[3]}
	}

	obj := &engine.Object{
		Meta: engine.ObjectMetadata{
			Name:           config.Name,
			Category:       engine.CategoryRing,
			Mass:           mass,
			PhysicalRadius: config.Distribution.OuterRadius,
			InnerRadius:    config.Distribution.InnerRadius,
			Color:          color,
			Material:       material,
			Importance:     importance,
			AxialTilt:      config.Physical.AxialTilt,
			OrbitRadius:    0,
			OrbitSpeed:     0,
			ParentName:     config.Parent,
			TexturePath:    config.Rendering.RingColorsMap,
		},
		Anim: engine.AnimationState{
			Position:    parent.Anim.Position,
			Velocity:    engine.Vector3{},
			OrbitCenter: parent.Anim.Position,
			OrbitAngle:  0,
			OrbitAxis:   engine.Vector3{X: 0, Y: 1, Z: 0},
		},
		Visible: true,
		Dataset: -1,
	}

	state.AddObject(obj)
	return nil
}

// LoadSystemFromDir loads a solar system from a versioned directory (F-034 format).
//
// The directory must contain a system.json manifest. Optional sub-files
// (stars.json, planets.json, etc.) are loaded when declared in the manifest.
// Returns the same *engine.SimulationState as LoadSystemFromFile for full
// backward compatibility with all downstream consumers.
func LoadSystemFromDir(dir string) (*engine.SimulationState, error) {
	manifestPath := filepath.Join(dir, "system.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("LoadSystemFromDir: failed to read system.json in %s: %w", dir, err)
	}

	var manifest SystemManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("LoadSystemFromDir: failed to parse system.json in %s: %w", dir, err)
	}

	if manifest.SchemaVersion == "" {
		return nil, fmt.Errorf("LoadSystemFromDir: system.json in %s is missing schema_version", dir)
	}
	if !supportedSchemaVersions[manifest.SchemaVersion] {
		return nil, fmt.Errorf("LoadSystemFromDir: unsupported schema_version %q in %s (supported: 2.0)", manifest.SchemaVersion, dir)
	}

	state := engine.NewSimulationState()
	rng := rand.New(rand.NewSource(42))
	seenCategories := make(map[engine.ObjectCategory]bool)

	// Helper: load a typed bodies file and add bodies to state.
	loadBodies := func(filename string) error {
		if filename == "" {
			return nil
		}
		path := filepath.Join(dir, filename)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		var f TypedBodiesFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}
		if f.SchemaVersion != "" && !supportedSchemaVersions[f.SchemaVersion] {
			return fmt.Errorf("unsupported schema_version %q in %s (supported: 2.0)", f.SchemaVersion, path)
		}
		for _, bc := range f.Bodies {
			obj, err := createBodyFromConfig(bc, nil, rng, manifest.ScaleFactor)
			if err != nil {
				return fmt.Errorf("failed to create body %q from %s: %w", bc.Name, path, err)
			}
			state.AddObject(obj)
			seenCategories[obj.Meta.Category] = true
		}
		return nil
	}

	// Load standard typed body files.
	for _, filename := range []string{
		manifest.Files.Stars,
		manifest.Files.Planets,
		manifest.Files.DwarfPlanets,
		manifest.Files.Moons,
	} {
		if err := loadBodies(filename); err != nil {
			return nil, err
		}
	}

	// Load rogues.json.
	if manifest.Files.Rogues != "" {
		path := filepath.Join(dir, manifest.Files.Rogues)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var f RoguesFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		if f.SchemaVersion != "" && !supportedSchemaVersions[f.SchemaVersion] {
			return nil, fmt.Errorf("unsupported schema_version %q in %s (supported: 2.0)", f.SchemaVersion, path)
		}
		for _, bc := range f.Rogues {
			if bc.Type == "" {
				bc.Type = "rogue"
			}
			obj, err := createBodyFromConfig(bc, nil, rng, manifest.ScaleFactor)
			if err != nil {
				return nil, fmt.Errorf("failed to create rogue %q from %s: %w", bc.Name, path, err)
			}
			state.AddObject(obj)
			seenCategories[obj.Meta.Category] = true
		}
	}

	// Load artifacts.json.
	if manifest.Files.Artifacts != "" {
		path := filepath.Join(dir, manifest.Files.Artifacts)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var f ArtifactsFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		if f.SchemaVersion != "" && !supportedSchemaVersions[f.SchemaVersion] {
			return nil, fmt.Errorf("unsupported schema_version %q in %s (supported: 2.0)", f.SchemaVersion, path)
		}
		for _, bc := range f.Artifacts {
			if bc.Type == "" {
				bc.Type = "artifact"
			}
			obj, err := createBodyFromConfig(bc, nil, rng, manifest.ScaleFactor)
			if err != nil {
				return nil, fmt.Errorf("failed to create artifact %q from %s: %w", bc.Name, path, err)
			}
			state.AddObject(obj)
			seenCategories[obj.Meta.Category] = true
		}
	}

	// Load belts.json.
	if manifest.Files.Belts != "" {
		path := filepath.Join(dir, manifest.Files.Belts)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var f TypedBeltsFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		if f.SchemaVersion != "" && !supportedSchemaVersions[f.SchemaVersion] {
			return nil, fmt.Errorf("unsupported schema_version %q in %s (supported: 2.0)", f.SchemaVersion, path)
		}
		for _, featureConfig := range f.Belts {
			if strings.ToLower(featureConfig.Type) == "asteroid_belt" {
				configCopy := featureConfig
				state.AsteroidBeltConfig = &configCopy
				seenCategories[engine.CategoryBelt] = true
			} else if strings.ToLower(featureConfig.Type) == "kuiper_belt" {
				configCopy := featureConfig
				state.KuiperBeltConfig = &configCopy
				seenCategories[engine.CategoryBelt] = true
			} else if strings.ToLower(featureConfig.Type) == "ring_system" ||
				strings.ToLower(featureConfig.Type) == "photon_ring" ||
				strings.ToLower(featureConfig.Type) == "accretion_disk" {
				seenCategories[engine.CategoryRing] = true
			}
			if err := createFeatureFromConfig(state, featureConfig, nil, rng); err != nil {
				return nil, fmt.Errorf("failed to create feature %q from %s: %w", featureConfig.Name, path, err)
			}
		}
	}

	// Load features.json (photon rings, accretion disks, relativistic jets, ring systems).
	if manifest.Files.Features != "" {
		path := filepath.Join(dir, manifest.Files.Features)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var f TypedFeaturesFile
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		if f.SchemaVersion != "" && !supportedSchemaVersions[f.SchemaVersion] {
			return nil, fmt.Errorf("unsupported schema_version %q in %s (supported: 2.0)", f.SchemaVersion, path)
		}
		for _, fc := range f.Features {
			ft := strings.ToLower(fc.Type)
			if ft == "ring_system" || ft == "photon_ring" || ft == "accretion_disk" {
				seenCategories[engine.CategoryRing] = true
			}
			if err := createFeatureFromConfig(state, fc, nil, rng); err != nil {
				return nil, fmt.Errorf("failed to create feature %q from %s: %w", fc.Name, path, err)
			}
		}
	}

	// Validate cross-file parent references: every body with a non-empty ParentName
	// must resolve to a loaded object.
	for _, obj := range state.Objects {
		if obj.Meta.ParentName != "" {
			if state.GetObject(obj.Meta.ParentName) == nil {
				return nil, fmt.Errorf("LoadSystemFromDir: body %q references unknown parent %q in %s",
					obj.Meta.Name, obj.Meta.ParentName, dir)
			}
		}
	}

	// Build NavigationOrder from seen categories in canonical order.
	canonicalOrder := []engine.ObjectCategory{
		engine.CategoryBlackHole,
		engine.CategoryStar,
		engine.CategoryPlanet,
		engine.CategoryDwarfPlanet,
		engine.CategoryMoon,
		engine.CategoryAsteroid,
		engine.CategoryRing,
		engine.CategoryBelt,
		engine.CategoryRogue,
		engine.CategoryArtifact,
	}
	for _, cat := range canonicalOrder {
		if seenCategories[cat] {
			state.NavigationOrder = append(state.NavigationOrder, cat)
		}
	}

	// Propagate simulation mode and compute N-body parameters.
	state.NBodyMode = manifest.Simulation.NBodyMode
	computeSOIRadii(state)

	return state, nil
}

// computeSOIRadii populates HillRadius and LaplaceSOI on all loaded objects.
// Called after all bodies are loaded so parent references are resolved.
func computeSOIRadii(state *engine.SimulationState) {
	for _, obj := range state.Objects {
		if obj.Meta.Mass == 0 {
			continue
		}
		if obj.Meta.ParentName == "" {
			// Top-level body (star): Hill sphere spans the system edge.
			obj.Meta.HillRadius = 1e5
			continue
		}
		parent := state.ObjectMap[obj.Meta.ParentName]
		if parent == nil || parent.Meta.Mass == 0 {
			continue
		}
		a := float64(obj.Meta.SemiMajorAxis)
		if a == 0 {
			continue
		}
		m := obj.Meta.Mass
		M := parent.Meta.Mass
		obj.Meta.HillRadius = a * math.Cbrt(m/(3*M))
		obj.Meta.LaplaceSOI = a * math.Pow(m/M, 0.4)
	}
}
