package engine

import (
	"fmt"
	"math"
)

// ─── Core types ──────────────────────────────────────────────────────────────

// GravSet is a configurable set of bodies for leapfrog N-body gravitational
// integration.  Participants exert gravity on each other and on TestParticles.
// TestParticles receive forces but do not exert them back (test-particle
// approximation — valid when their mass is negligible compared to Participants).
type GravSet struct {
	Name          string    // human label for debug/logging
	Participants  []*Object // full mutual interaction (stars, planets, moons, artifacts)
	TestParticles []*Object // receive forces only (ships, asteroids near an SOI)
}

// ShipParticle is the physics representation of a client ship in the N-body
// layer.  It bridges the session registry with GravSet TestParticles.
// Full lifecycle management is deferred to F-022 Phase 2.
type ShipParticle struct {
	SessionID string
	NBodyPos  [3]float64
	NBodyVel  [3]float64
	NBodyAcc  [3]float64
}

// ─── Collectors ──────────────────────────────────────────────────────────────

// CollectByCategory returns all named non-ring objects matching any of the given
// categories.  "Named" means Dataset == -1.  CategoryRing is always excluded.
// If CategoryAsteroid is included in cats, named asteroids (Dataset == -1)
// are also included.
func CollectByCategory(state *SimulationState, cats ...ObjectCategory) []*Object {
	want := make(map[ObjectCategory]bool, len(cats))
	for _, c := range cats {
		want[c] = true
	}
	var out []*Object
	for _, obj := range state.Objects {
		if obj.Meta.Category == CategoryRing {
			continue
		}
		if !want[obj.Meta.Category] {
			continue
		}
		if obj.Dataset >= 0 && !want[CategoryAsteroid] {
			continue // skip belt particles unless explicitly requested
		}
		out = append(out, obj)
	}
	return out
}

// CollectInSOI returns all objects whose current float32 Position is within
// radius sim units of center.  Named bodies are candidates for Participants;
// belt members are candidates for TestParticles.
func CollectInSOI(state *SimulationState, center Vector3, radius float64) []*Object {
	r2 := radius * radius
	var out []*Object
	for _, obj := range state.Objects {
		dx := float64(obj.Anim.Position.X - center.X)
		dy := float64(obj.Anim.Position.Y - center.Y)
		dz := float64(obj.Anim.Position.Z - center.Z)
		if dx*dx+dy*dy+dz*dz <= r2 {
			out = append(out, obj)
		}
	}
	return out
}

// CollectInHillSphere returns all objects inside body.Meta.HillRadius of body.
// Convenience wrapper around CollectInSOI using body.Anim.Position as center.
func CollectInHillSphere(state *SimulationState, body *Object) []*Object {
	return CollectInSOI(state, body.Anim.Position, body.Meta.HillRadius)
}

// CollectChildren returns all objects whose Meta.ParentName equals
// parent.Meta.Name.  Includes direct children only.
func CollectChildren(state *SimulationState, parent *Object) []*Object {
	var out []*Object
	for _, obj := range state.Objects {
		if obj.Meta.ParentName == parent.Meta.Name {
			out = append(out, obj)
		}
	}
	return out
}

// CollectByName returns the named objects from state.ObjectMap.
// Missing names are silently skipped.
func CollectByName(state *SimulationState, names ...string) []*Object {
	var out []*Object
	for _, n := range names {
		if obj := state.ObjectMap[n]; obj != nil {
			out = append(out, obj)
		}
	}
	return out
}

// ─── Set builders ────────────────────────────────────────────────────────────

// BuildSystemSet builds the default system-wide GravSet:
//   - Participants = all named non-ring bodies (stars, planets, dwarf planets,
//     moons, artifacts).
//   - TestParticles = provided ship objects.
func BuildSystemSet(state *SimulationState, ships ...*Object) GravSet {
	return GravSet{
		Name: "system",
		Participants: CollectByCategory(state,
			CategoryStar, CategoryPlanet, CategoryDwarfPlanet,
			CategoryMoon, CategoryArtifact,
		),
		TestParticles: ships,
	}
}

// BuildLocalSet builds a planet-centric GravSet:
//   - Participants = planet + its direct moons.
//   - TestParticles = ships whose Position is inside the planet's Hill sphere.
func BuildLocalSet(state *SimulationState, planet *Object, ships ...*Object) GravSet {
	children := CollectChildren(state, planet)
	participants := make([]*Object, 0, 1+len(children))
	participants = append(participants, planet)
	participants = append(participants, children...)

	var testParticles []*Object
	for _, s := range ships {
		if isInHillSphere(s.Anim.Position, planet) {
			testParticles = append(testParticles, s)
		}
	}
	return GravSet{
		Name:          planet.Meta.Name + "/local",
		Participants:  participants,
		TestParticles: testParticles,
	}
}

// BuildSOISet builds a spatial GravSet from an explicit center and radius.
// Named bodies inside the sphere become Participants; belt members and ships
// inside the sphere become TestParticles.
func BuildSOISet(state *SimulationState, center Vector3, radius float64, ships ...*Object) GravSet {
	inside := CollectInSOI(state, center, radius)
	r2 := radius * radius
	var parts, beltParts []*Object
	for _, obj := range inside {
		if obj.Meta.Category == CategoryRing {
			continue
		}
		if obj.Dataset >= 0 {
			beltParts = append(beltParts, obj)
		} else {
			parts = append(parts, obj)
		}
	}
	var testParticles []*Object
	testParticles = append(testParticles, beltParts...)
	for _, s := range ships {
		dx := float64(s.Anim.Position.X - center.X)
		dy := float64(s.Anim.Position.Y - center.Y)
		dz := float64(s.Anim.Position.Z - center.Z)
		if dx*dx+dy*dy+dz*dz <= r2 {
			testParticles = append(testParticles, s)
		}
	}
	return GravSet{
		Name:          fmt.Sprintf("soi(r=%.1f)", radius),
		Participants:  parts,
		TestParticles: testParticles,
	}
}

// ─── Leapfrog DKD integrator ─────────────────────────────────────────────────

// StepGravSet advances all bodies in gs by one leapfrog Drift-Kick-Drift step
// of size dt seconds.  NBodyPos and NBodyVel are updated for all bodies.
// For Participants, float32 Position and Velocity are also updated so the
// renderer sees the new positions immediately.
// TestParticle float32 copies are left for the F-022 session-registry caller.
func StepGravSet(gs GravSet, dt float64) {
	h := dt / 2.0
	all := make([]*Object, 0, len(gs.Participants)+len(gs.TestParticles))
	all = append(all, gs.Participants...)
	all = append(all, gs.TestParticles...)

	// Drift ½ — all bodies advance half a step at current velocity.
	for _, obj := range all {
		obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
		obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
		obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
	}

	// Kick — compute accelerations at half-drifted positions, then update velocities.
	accumForces(gs)
	accumTestParticleForces(gs)
	for _, obj := range all {
		obj.Anim.NBodyVel[0] += obj.Anim.NBodyAcc[0] * dt
		obj.Anim.NBodyVel[1] += obj.Anim.NBodyAcc[1] * dt
		obj.Anim.NBodyVel[2] += obj.Anim.NBodyAcc[2] * dt
	}

	// Drift ½ — all bodies advance the remaining half step.
	for _, obj := range all {
		obj.Anim.NBodyPos[0] += obj.Anim.NBodyVel[0] * h
		obj.Anim.NBodyPos[1] += obj.Anim.NBodyVel[1] * h
		obj.Anim.NBodyPos[2] += obj.Anim.NBodyVel[2] * h
	}

	// Copy float64 precision state → float32 for the renderer (Participants only).
	for _, obj := range gs.Participants {
		obj.Anim.Position.X = float32(obj.Anim.NBodyPos[0])
		obj.Anim.Position.Y = float32(obj.Anim.NBodyPos[1])
		obj.Anim.Position.Z = float32(obj.Anim.NBodyPos[2])
		obj.Anim.Velocity.X = float32(obj.Anim.NBodyVel[0])
		obj.Anim.Velocity.Y = float32(obj.Anim.NBodyVel[1])
		obj.Anim.Velocity.Z = float32(obj.Anim.NBodyVel[2])
	}
}

// accumForces zero-initialises NBodyAcc on all Participants and then
// accumulates the O(N²) pairwise gravitational accelerations via Newton's law.
func accumForces(gs GravSet) {
	n := len(gs.Participants)
	for _, obj := range gs.Participants {
		obj.Anim.NBodyAcc = [3]float64{}
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pi := gs.Participants[i]
			pj := gs.Participants[j]
			dx := pj.Anim.NBodyPos[0] - pi.Anim.NBodyPos[0]
			dy := pj.Anim.NBodyPos[1] - pi.Anim.NBodyPos[1]
			dz := pj.Anim.NBodyPos[2] - pi.Anim.NBodyPos[2]
			r2 := dx*dx + dy*dy + dz*dz
			if r2 < 1e-12 {
				continue
			}
			r3 := r2 * math.Sqrt(r2)
			// Acceleration on pi from pj
			f := pj.Meta.GM / r3
			pi.Anim.NBodyAcc[0] += f * dx
			pi.Anim.NBodyAcc[1] += f * dy
			pi.Anim.NBodyAcc[2] += f * dz
			// Reaction on pj from pi (Newton's third law)
			g := pi.Meta.GM / r3
			pj.Anim.NBodyAcc[0] -= g * dx
			pj.Anim.NBodyAcc[1] -= g * dy
			pj.Anim.NBodyAcc[2] -= g * dz
		}
	}
}

// accumTestParticleForces accumulates gravity on each TestParticle from all
// Participants.  Called after accumForces.
func accumTestParticleForces(gs GravSet) {
	for _, tp := range gs.TestParticles {
		tp.Anim.NBodyAcc = [3]float64{}
		for _, p := range gs.Participants {
			dx := p.Anim.NBodyPos[0] - tp.Anim.NBodyPos[0]
			dy := p.Anim.NBodyPos[1] - tp.Anim.NBodyPos[1]
			dz := p.Anim.NBodyPos[2] - tp.Anim.NBodyPos[2]
			r2 := dx*dx + dy*dy + dz*dz
			if r2 < 1e-12 {
				continue
			}
			r3 := r2 * math.Sqrt(r2)
			tp.Anim.NBodyAcc[0] += p.Meta.GM / r3 * dx
			tp.Anim.NBodyAcc[1] += p.Meta.GM / r3 * dy
			tp.Anim.NBodyAcc[2] += p.Meta.GM / r3 * dz
		}
	}
}

// ─── SOI Tracker (Phase 5 stub) ───────────────────────────────────────────────

// SOITracker checks whether test particles have crossed sphere-of-influence
// boundaries and updates LocalSet membership accordingly.
// The Tick method is a no-op until ship physics (F-022 Phase 2) is implemented.
type SOITracker struct {
	tickInterval int
	tickCount    int
	// localSets will map planet name → active LocalSet once F-022 Phase 2 is done.
}

// NewSOITracker returns a SOITracker that checks every tickInterval physics ticks.
// If tickInterval <= 0 it defaults to 60 (≈1 real second at 60 Hz).
func NewSOITracker(tickInterval int) *SOITracker {
	if tickInterval <= 0 {
		tickInterval = 60
	}
	return &SOITracker{tickInterval: tickInterval}
}

// Tick advances the tracker counter.  Currently a no-op stub; full
// promotion/demotion logic is deferred to F-022 Phase 2.
func (t *SOITracker) Tick(_ *SimulationState) {
	t.tickCount++
	if t.tickCount < t.tickInterval {
		return
	}
	t.tickCount = 0
	// TODO F-022 Phase 2: promote/demote test particles crossing SOI boundaries.
}

// ─── Internal helpers ────────────────────────────────────────────────────────

func isInHillSphere(pos Vector3, body *Object) bool {
	dx := float64(pos.X - body.Anim.Position.X)
	dy := float64(pos.Y - body.Anim.Position.Y)
	dz := float64(pos.Z - body.Anim.Position.Z)
	return dx*dx+dy*dy+dz*dz <= body.Meta.HillRadius*body.Meta.HillRadius
}
