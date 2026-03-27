package space

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/digital-michael/space_sim/internal/space/engine"
)

// Simulation wraps engine.Simulation with SOL-specific dataset management.
type Simulation struct {
	*engine.Simulation
}

// NewSimulation creates a new simulation by loading from a JSON configuration.
// If configPath is empty, defaults to "data/systems/solar_system.json".
func NewSimulation(hz float64, configPath string) *Simulation {
	if configPath == "" {
		configPath = "data/systems/solar_system.json"
	}

	state, err := LoadSystemFromFile(configPath)
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to load system from %s: %v\nPlease ensure the JSON configuration file exists.",
			configPath, err,
		))
	}

	fmt.Printf("\u2713 Loaded system from %s\n", configPath)

	// We need a reference to the DoubleBuffer that engine.NewSimulation will
	// create internally.  Use a pointer-to-pointer: the apply closure reads
	// *dbPtr, which we fill in after engine.NewSimulation returns (before
	// Start is ever called).
	var dbPtr *engine.DoubleBuffer

	applyFn := func(dataset engine.AsteroidDataset) {
		db := dbPtr // safe: Start() has not been called yet when we set dbPtr
		if db == nil {
			return
		}
		back := db.GetBack()

		if back.CurrentDataset == dataset {
			return
		}

		asteroidAllocated := hasBeltDataset(back, "Asteroid-", dataset)
		kuiperAllocated := hasBeltDataset(back, "KBO-", dataset)

		needsAsteroid := back.AsteroidBeltConfig != nil && !asteroidAllocated
		needsKuiper := back.KuiperBeltConfig != nil && !kuiperAllocated

		if needsAsteroid || needsKuiper {
			// Disable in-place swap while mutating object slice lengths.
			db.DisableInPlaceSwap()

			if needsAsteroid {
				rngAstBack := beltDatasetRNG(back.AsteroidBeltConfig, dataset)
				applyBeltConfig(back, back.AsteroidBeltConfig, dataset, rngAstBack)
			}
			if needsKuiper {
				rngKupBack := beltDatasetRNG(back.KuiperBeltConfig, dataset)
				applyBeltConfig(back, back.KuiperBeltConfig, dataset, rngKupBack)
			}
			back.AllocatedDatasets[dataset] = true

			// Mirror allocations into front buffer with identical seeds/layout.
			front := db.LockFrontWrite()
			if front.AsteroidBeltConfig == nil {
				front.AsteroidBeltConfig = back.AsteroidBeltConfig
			}
			if front.KuiperBeltConfig == nil {
				front.KuiperBeltConfig = back.KuiperBeltConfig
			}

			if needsAsteroid && front.AsteroidBeltConfig != nil {
				rngAstFront := beltDatasetRNG(front.AsteroidBeltConfig, dataset)
				applyBeltConfig(front, front.AsteroidBeltConfig, dataset, rngAstFront)
			}
			if needsKuiper && front.KuiperBeltConfig != nil {
				rngKupFront := beltDatasetRNG(front.KuiperBeltConfig, dataset)
				applyBeltConfig(front, front.KuiperBeltConfig, dataset, rngKupFront)
			}
			front.AllocatedDatasets[dataset] = true
			db.UnlockFrontWrite()

			// Both buffers equal length — safe to re-enable in-place swap.
			db.EnableInPlaceSwap()
		}

		// Update visibility in back buffer.
		back.CurrentDataset = dataset
		for _, obj := range back.Objects {
			if isBeltRuntimeObject(obj) {
				obj.Visible = obj.Dataset <= dataset
			}
		}

		// Update visibility in front buffer (must lock).
		front := db.LockFrontWrite()
		front.CurrentDataset = dataset
		for _, obj := range front.Objects {
			if isBeltRuntimeObject(obj) {
				obj.Visible = obj.Dataset <= dataset
			}
		}
		db.UnlockFrontWrite()
	}

	inner := engine.NewSimulation(state, hz, applyFn)

	// Inject the DoubleBuffer reference now that engine.NewSimulation has
	// created it.  This happens before Start() so there is no data race.
	dbPtr = inner.GetState()

	return &Simulation{Simulation: inner}
}

// GetAsteroidCount returns the total number of asteroids for a given dataset.
func GetAsteroidCount(dataset engine.AsteroidDataset) int {
	return asteroidCount(dataset)
}

// GetDatasetName returns a human-readable name for a dataset level.
func GetDatasetName(dataset engine.AsteroidDataset) string {
	return datasetName(dataset)
}

// applyBeltConfig translates a FeatureConfig into a BeltConfig and populates
// state with the belt objects for the given dataset level.
func applyBeltConfig(state *engine.SimulationState, config *engine.FeatureConfig, dataset engine.AsteroidDataset, rng *rand.Rand) {
	if config == nil {
		fmt.Println("Warning: belt config missing; skipping belt allocation")
		return
	}

	objectTypes := make(map[string]BeltObjectTypeConfig)
	for typeName, typeSpec := range config.ObjectTypes {
		if int(dataset) < len(typeSpec.CountByLevel) {
			count := typeSpec.CountByLevel[int(dataset)]
			importance := 10
			if int(dataset) < len(typeSpec.ImportanceByLevel) {
				importance = typeSpec.ImportanceByLevel[int(dataset)]
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
		NamePrefix:               beltNamePrefix(config.Type),
		InnerRadius:              config.Distribution.InnerRadius,
		OuterRadius:              config.Distribution.OuterRadius,
		Thickness:                config.Distribution.Thickness,
		ClassicalBeltMin:         config.Distribution.ClassicalBeltRange[0],
		ClassicalBeltMax:         config.Distribution.ClassicalBeltRange[1],
		ClassicalBeltProbability: config.Distribution.ClassicalBeltProbability,
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

	CreateBelt(state, beltConfig, dataset, rng)
}

func beltNamePrefix(featureType string) string {
	switch strings.ToLower(featureType) {
	case "asteroid_belt":
		return "Asteroid-"
	case "kuiper_belt":
		return "KBO-"
	default:
		return "Belt-"
	}
}

func hasBeltDataset(state *engine.SimulationState, prefix string, dataset engine.AsteroidDataset) bool {
	for _, obj := range state.Objects {
		if obj.Dataset == dataset && strings.HasPrefix(obj.Meta.Name, prefix) {
			return true
		}
	}
	return false
}

func isBeltRuntimeObject(obj *engine.Object) bool {
	if obj.Dataset < 0 {
		return false
	}
	return strings.HasPrefix(obj.Meta.Name, "Asteroid-") || strings.HasPrefix(obj.Meta.Name, "KBO-")
}

func beltDatasetRNG(config *engine.FeatureConfig, dataset engine.AsteroidDataset) *rand.Rand {
	baseSeed := int64(42)
	if config != nil && config.Procedural.Seed != 0 {
		baseSeed = config.Procedural.Seed
	}
	return rand.New(rand.NewSource(baseSeed + int64(dataset)*1000003))
}

// asteroidCount returns the total number of asteroids for a dataset.
func asteroidCount(dataset engine.AsteroidDataset) int {
	switch dataset {
	case engine.AsteroidDatasetSmall:
		return 200
	case engine.AsteroidDatasetMedium:
		return 1200
	case engine.AsteroidDatasetLarge:
		return 2400
	case engine.AsteroidDatasetHuge:
		return 24000
	default:
		return 200
	}
}

// datasetName returns a human-readable name for the dataset.
func datasetName(dataset engine.AsteroidDataset) string {
	switch dataset {
	case engine.AsteroidDatasetSmall:
		return "Small (200)"
	case engine.AsteroidDatasetMedium:
		return "Medium (1.2K)"
	case engine.AsteroidDatasetLarge:
		return "Large (2.4K)"
	case engine.AsteroidDatasetHuge:
		return "Huge (24K)"
	default:
		return "Unknown"
	}
}

// anomalyNormalize keeps mean anomaly in [0, 2pi).
func anomalyNormalize(v float32) float32 {
	twoPi := float32(2.0 * math.Pi)
	v = float32(math.Mod(float64(v), float64(twoPi)))
	if v < 0 {
		v += twoPi
	}
	return v
}
