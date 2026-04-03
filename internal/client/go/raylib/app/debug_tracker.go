package app

import engine "github.com/digital-michael/space_sim/internal/sim/engine"

// DebugTracker tracks Earth and Moon visibility for debugging disappearance issues.
type DebugTracker struct {
	lastEarthVisible bool
	lastMoonVisible  bool
	frameCount       int
}

// NewDebugTracker creates a new debug tracker.
func NewDebugTracker() *DebugTracker {
	return &DebugTracker{
		lastEarthVisible: true,
		lastMoonVisible:  true,
	}
}

// CheckVisibility logs whenever Earth or Moon visibility changes.
func (d *DebugTracker) CheckVisibility(objects []*engine.Object, reason string) {
	d.frameCount++

	var earth *engine.Object
	var moon *engine.Object

	for _, obj := range objects {
		if obj.Meta.Name == "Earth" {
			earth = obj
		} else if obj.Meta.Name == "Moon" {
			moon = obj
		}
	}

	if earth != nil {
		if earth.Visible != d.lastEarthVisible {
			d.lastEarthVisible = earth.Visible
		}
	} else if d.lastEarthVisible {
		d.lastEarthVisible = false
	}

	if moon != nil {
		if moon.Visible != d.lastMoonVisible {
			d.lastMoonVisible = moon.Visible
		}
	} else if d.lastMoonVisible {
		d.lastMoonVisible = false
	}
}

// LogRenderDecision logs why Earth/Moon was not rendered in a specific frame.
func (d *DebugTracker) LogRenderDecision(obj *engine.Object, culled bool, cullingReason string) {
	if obj.Meta.Name == "Earth" || obj.Meta.Name == "Moon" {
		if !obj.Visible {
			return
		}
		if culled {
			return
		}
	}
}
