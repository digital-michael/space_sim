// Package engine - SimCommand is defined here so the simulation loop can
// accept typed commands without importing domain packages.
package engine

// SimCommand is a command dispatched to the simulation loop. Each concrete
// type carries the data needed for the loop to act without further coordination.
type SimCommand interface {
	simCommand() // unexported marker
}

// DatasetChangeCommand requests a switch to a new belt dataset level.
type DatasetChangeCommand struct {
	Dataset AsteroidDataset
}

func (DatasetChangeCommand) simCommand() {}
