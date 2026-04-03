package api

// SimulationControl is the contract for server-side commands that affect the
// running simulation — speed, pause, dataset, and world selection.
//
// TODO: flesh out when the Phase 6 gRPC command surface is defined. Expected
// additions: SetTimeScale, Pause/Resume, LoadWorld, SaveWorld.
type SimulationControl interface {
	// SetSpeed sets the simulation time multiplier (seconds-of-sim per
	// second of wall-clock time). A value of 1.0 is real-time; 3600.0 is
	// one simulation hour per wall-clock second.
	SetSpeed(secondsPerSecond float32)

	// Speed returns the current time multiplier.
	Speed() float32

	// SetDataset selects the asteroid population LOD tier.
	// TODO: replace int with a typed constant once api owns the enum.
	SetDataset(level int)

	// Dataset returns the currently active LOD tier.
	Dataset() int

	// TODO: Pause()
	// TODO: Resume()
	// TODO: LoadWorld(configPath string) error
	// TODO: SaveWorld(path string) error
}

// AnimationControl is the contract for server-side commands that drive or
// inspect per-body animation state — primarily for tooling, replay, and
// scripted sequences.
//
// TODO: flesh out when replay and scripting requirements are defined.
// Expected additions: SeekToTime, SetBodyPosition, ResetToEpoch.
type AnimationControl interface {
	// SimulationTime returns the current simulation time in seconds since
	// J2000.0.
	SimulationTime() float64

	// TODO: SeekToTime(secondsSinceJ2000 float64) error
	// TODO: SetBodyPosition(name string, pos [3]float64) error
	// TODO: ResetToEpoch()
}
