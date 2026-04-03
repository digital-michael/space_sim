package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim/engine"
)

// snapshotFile is the on-disk JSON representation of a WorldSnapshot.
type snapshotFile struct {
	Speed              float64                 `json:"speed"`
	SimTime            float64                 `json:"sim_time"`
	DeltaTime          float64                 `json:"delta_time"`
	SecondsPerSecond   float32                 `json:"seconds_per_second"`
	NumWorkers         int                     `json:"num_workers"`
	CurrentDataset     engine.AsteroidDataset  `json:"current_dataset"`
	AllocatedDatasets  map[string]bool         `json:"allocated_datasets"`
	AsteroidBeltConfig *engine.FeatureConfig   `json:"asteroid_belt_config,omitempty"`
	KuiperBeltConfig   *engine.FeatureConfig   `json:"kuiper_belt_config,omitempty"`
	NavigationOrder    []engine.ObjectCategory `json:"navigation_order"`
	Objects            []snapshotObject        `json:"objects"`
}

type snapshotObject struct {
	Meta    engine.ObjectMetadata  `json:"meta"`
	Anim    engine.AnimationState  `json:"anim"`
	Visible bool                   `json:"visible"`
	Dataset engine.AsteroidDataset `json:"dataset"`
}

// SaveSnapshot writes a full WorldSnapshot (Meta + AnimationState + timing) to
// path atomically.
func SaveSnapshot(path string, snap protocol.WorldSnapshot) error {
	if snap.State == nil {
		return fmt.Errorf("persist: snapshot state is nil")
	}

	objs := make([]snapshotObject, len(snap.State.Objects))
	for i, o := range snap.State.Objects {
		objs[i] = snapshotObject{
			Meta:    o.Meta,
			Anim:    o.Anim,
			Visible: o.Visible,
			Dataset: o.Dataset,
		}
	}

	allocated := make(map[string]bool, len(snap.State.AllocatedDatasets))
	for k, v := range snap.State.AllocatedDatasets {
		allocated[strconv.Itoa(int(k))] = v
	}

	f := snapshotFile{
		Speed:              snap.Speed,
		SimTime:            snap.State.Time,
		DeltaTime:          snap.State.DeltaTime,
		SecondsPerSecond:   snap.State.SecondsPerSecond,
		NumWorkers:         snap.State.NumWorkers,
		CurrentDataset:     snap.State.CurrentDataset,
		AllocatedDatasets:  allocated,
		AsteroidBeltConfig: snap.State.AsteroidBeltConfig,
		KuiperBeltConfig:   snap.State.KuiperBeltConfig,
		NavigationOrder:    snap.State.NavigationOrder,
		Objects:            objs,
	}

	data, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("persist: marshal snapshot: %w", err)
	}
	return atomicWrite(path, data)
}

// LoadSnapshot reads a snapshot file and returns a WorldSnapshot with both
// Meta and AnimationState restored.
func LoadSnapshot(path string) (protocol.WorldSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return protocol.WorldSnapshot{}, fmt.Errorf("persist: read snapshot %s: %w", path, err)
	}

	var f snapshotFile
	if err := json.Unmarshal(data, &f); err != nil {
		return protocol.WorldSnapshot{}, fmt.Errorf("persist: unmarshal snapshot: %w", err)
	}

	state := engine.NewSimulationState()
	state.Time = f.SimTime
	state.DeltaTime = f.DeltaTime
	state.SecondsPerSecond = f.SecondsPerSecond
	state.NumWorkers = f.NumWorkers
	state.CurrentDataset = f.CurrentDataset
	state.AsteroidBeltConfig = f.AsteroidBeltConfig
	state.KuiperBeltConfig = f.KuiperBeltConfig
	state.NavigationOrder = f.NavigationOrder

	for k, v := range f.AllocatedDatasets {
		n, err := strconv.Atoi(k)
		if err != nil {
			return protocol.WorldSnapshot{}, fmt.Errorf("persist: invalid dataset key %q: %w", k, err)
		}
		state.AllocatedDatasets[engine.AsteroidDataset(n)] = v
	}

	for _, so := range f.Objects {
		obj := &engine.Object{
			Meta:    so.Meta,
			Anim:    so.Anim,
			Visible: so.Visible,
			Dataset: so.Dataset,
		}
		state.AddObject(obj)
	}

	return protocol.WorldSnapshot{
		State: state,
		Speed: f.Speed,
	}, nil
}
