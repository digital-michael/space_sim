// Package sim provides the simulation process layer: it takes an environment
// built by the space package and drives it at runtime, including asteroid belt
// dataset management and double-buffer lifecycle.
package world

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// Simulation wraps engine.Simulation with dataset management driven by the
// loaded system configuration.
type World struct {
	*engine.Simulation
	beltConfigs []*engine.FeatureConfig // belt feature configs for count queries
}

// NewSimulation loads an environment from configPath and starts the simulation
// process. If configPath is empty, defaults to "data/systems/solar_system.json".
func NewWorld(hz float64, configPath string) (*World, error) {
	if configPath == "" {
		configPath = "data/systems/solar_system.json"
	}

	state, err := sim.LoadSystemFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("load system %s: %w", configPath, err)
	}

	fmt.Printf("\u2713 Loaded system from %s\n", configPath)

	// Derive belt prefixes and configs from the loaded system.
	// These are stable after construction and safe to capture in the closure.
	var beltConfigs []*engine.FeatureConfig
	var beltPrefixes []string
	asteroidPrefix, kuiperPrefix := "", ""
	if state.AsteroidBeltConfig != nil {
		asteroidPrefix = beltNamePrefix(state.AsteroidBeltConfig.Type)
		beltPrefixes = append(beltPrefixes, asteroidPrefix)
		beltConfigs = append(beltConfigs, state.AsteroidBeltConfig)
	}
	if state.KuiperBeltConfig != nil {
		kuiperPrefix = beltNamePrefix(state.KuiperBeltConfig.Type)
		beltPrefixes = append(beltPrefixes, kuiperPrefix)
		beltConfigs = append(beltConfigs, state.KuiperBeltConfig)
	}

	// We need a reference to the DoubleBuffer that engine.NewSimulation will
	// create internally. Use a pointer-to-pointer: the apply closure reads
	// *dbPtr, which we fill in after engine.NewSimulation returns (before
	// Start is ever called).
	var dbPtr *engine.DoubleBuffer

	applyFn := func(cmd engine.SimCommand) {
		dc, ok := cmd.(engine.DatasetChangeCommand)
		if !ok {
			return
		}
		dataset := dc.Dataset

		db := dbPtr // safe: Start() has not been called yet when we set dbPtr
		if db == nil {
			return
		}
		back := db.GetBack()

		if back.CurrentDataset == dataset {
			return
		}

		asteroidAllocated := asteroidPrefix != "" && hasBeltDataset(back, asteroidPrefix, dataset)
		kuiperAllocated := kuiperPrefix != "" && hasBeltDataset(back, kuiperPrefix, dataset)

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
			if isBeltObject(obj, beltPrefixes) {
				obj.Visible = obj.Dataset <= dataset
			}
		}

		// Update visibility in front buffer (must lock).
		front := db.LockFrontWrite()
		front.CurrentDataset = dataset
		for _, obj := range front.Objects {
			if isBeltObject(obj, beltPrefixes) {
				obj.Visible = obj.Dataset <= dataset
			}
		}
		db.UnlockFrontWrite()
	}

	inner := engine.NewSimulation(state, hz, applyFn)

	// Inject the DoubleBuffer reference now that engine.NewSimulation has
	// created it. This happens before Start() so there is no data race.
	dbPtr = inner.GetState()

	return &World{Simulation: inner, beltConfigs: beltConfigs}, nil
}

// GetAsteroidCount returns the total number of belt objects for the given
// dataset level, summed across all belt feature configs in the loaded system.
func (s *World) GetAsteroidCount(dataset engine.AsteroidDataset) int {
	total := 0
	for _, cfg := range s.beltConfigs {
		for _, spec := range cfg.ObjectTypes {
			if int(dataset) < len(spec.CountByLevel) {
				total += spec.CountByLevel[int(dataset)]
			}
		}
	}
	return total
}

// Snapshot returns a lock-free point-in-time copy of the simulation state.
// Safe to read after this call returns without holding any lock.
func (w *World) Snapshot() protocol.WorldSnapshot {
	db := w.GetState()
	state := db.LockFront()
	cloned := state.Clone()
	db.UnlockFront()
	return protocol.WorldSnapshot{
		State: cloned,
		Speed: w.GetSpeed(),
	}
}

// applyBeltConfig translates a FeatureConfig into a BeltConfig and populates
// state with the belt objects for the given dataset level.
func applyBeltConfig(state *engine.SimulationState, config *engine.FeatureConfig, dataset engine.AsteroidDataset, rng *rand.Rand) {
	if config == nil {
		fmt.Println("Warning: belt config missing; skipping belt allocation")
		return
	}

	objectTypes := make(map[string]sim.BeltObjectTypeConfig)
	for typeName, typeSpec := range config.ObjectTypes {
		if int(dataset) < len(typeSpec.CountByLevel) {
			count := typeSpec.CountByLevel[int(dataset)]
			importance := 10
			if int(dataset) < len(typeSpec.ImportanceByLevel) {
				importance = typeSpec.ImportanceByLevel[int(dataset)]
			}
			objectTypes[typeName] = sim.BeltObjectTypeConfig{
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

	beltConfig := sim.BeltConfig{
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

	sim.CreateBelt(state, beltConfig, dataset, rng)
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

func isBeltObject(obj *engine.Object, prefixes []string) bool {
	if obj.Dataset < 0 {
		return false
	}
	for _, p := range prefixes {
		if strings.HasPrefix(obj.Meta.Name, p) {
			return true
		}
	}
	return false
}

func beltDatasetRNG(config *engine.FeatureConfig, dataset engine.AsteroidDataset) *rand.Rand {
	baseSeed := int64(42)
	if config != nil && config.Procedural.Seed != 0 {
		baseSeed = config.Procedural.Seed
	}
	return rand.New(rand.NewSource(baseSeed + int64(dataset)*1000003))
}
