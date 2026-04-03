package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// definitionsFile is the on-disk JSON representation of the Meta-only
// (static) layer of a SimulationState.
type definitionsFile struct {
	Objects            []objectDef             `json:"objects"`
	CurrentDataset     engine.AsteroidDataset  `json:"current_dataset"`
	AllocatedDatasets  map[string]bool         `json:"allocated_datasets"`
	AsteroidBeltConfig *engine.FeatureConfig   `json:"asteroid_belt_config,omitempty"`
	KuiperBeltConfig   *engine.FeatureConfig   `json:"kuiper_belt_config,omitempty"`
	NavigationOrder    []engine.ObjectCategory `json:"navigation_order"`
	SecondsPerSecond   float32                 `json:"seconds_per_second"`
	NumWorkers         int                     `json:"num_workers"`
}

// objectDef captures the static properties of one object (Meta + display flags).
// AnimationState is intentionally absent; it is zero-valued on load and
// recalculated by the physics engine on its first tick.
type objectDef struct {
	Meta    engine.ObjectMetadata  `json:"meta"`
	Visible bool                   `json:"visible"`
	Dataset engine.AsteroidDataset `json:"dataset"`
}

// SaveDefinitions writes the static layer of state to path atomically.
// Only Meta, Visible, Dataset, and configuration fields are persisted;
// per-frame AnimationState is excluded.
func SaveDefinitions(path string, state *engine.SimulationState) error {
	defs := make([]objectDef, len(state.Objects))
	for i, o := range state.Objects {
		defs[i] = objectDef{
			Meta:    o.Meta,
			Visible: o.Visible,
			Dataset: o.Dataset,
		}
	}

	// Encode AllocatedDatasets with string keys for portable JSON round-trips.
	allocated := make(map[string]bool, len(state.AllocatedDatasets))
	for k, v := range state.AllocatedDatasets {
		allocated[strconv.Itoa(int(k))] = v
	}

	f := definitionsFile{
		Objects:            defs,
		CurrentDataset:     state.CurrentDataset,
		AllocatedDatasets:  allocated,
		AsteroidBeltConfig: state.AsteroidBeltConfig,
		KuiperBeltConfig:   state.KuiperBeltConfig,
		NavigationOrder:    state.NavigationOrder,
		SecondsPerSecond:   state.SecondsPerSecond,
		NumWorkers:         state.NumWorkers,
	}

	data, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("persist: marshal definitions: %w", err)
	}
	return atomicWrite(path, data)
}

// LoadDefinitions reads a definitions file and returns a SimulationState with
// Meta populated on each object. AnimationState fields are zero-valued until
// the physics engine runs its first tick.
func LoadDefinitions(path string) (*engine.SimulationState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persist: read definitions %s: %w", path, err)
	}

	var f definitionsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("persist: unmarshal definitions: %w", err)
	}

	state := engine.NewSimulationState()
	state.CurrentDataset = f.CurrentDataset
	state.AsteroidBeltConfig = f.AsteroidBeltConfig
	state.KuiperBeltConfig = f.KuiperBeltConfig
	state.NavigationOrder = f.NavigationOrder
	state.SecondsPerSecond = f.SecondsPerSecond
	state.NumWorkers = f.NumWorkers

	for k, v := range f.AllocatedDatasets {
		n, err := strconv.Atoi(k)
		if err != nil {
			return nil, fmt.Errorf("persist: invalid dataset key %q: %w", k, err)
		}
		state.AllocatedDatasets[engine.AsteroidDataset(n)] = v
	}

	for _, d := range f.Objects {
		obj := &engine.Object{
			Meta:    d.Meta,
			Visible: d.Visible,
			Dataset: d.Dataset,
		}
		state.AddObject(obj)
	}

	return state, nil
}
