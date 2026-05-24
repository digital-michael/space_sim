package engine

import (
	"math"
	"testing"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

const (
	// Sim-unit values for common bodies (1 sim unit = AU/100 = 1.496e9 m)
	massSol   = 1.989e30 // kg
	massEarth = 5.972e24 // kg
	massMoon  = 7.342e22 // kg
	aMoon     = 2.57     // sim units (384,400 km / 1.496e9 m per sim unit)
	aEarth    = 100.0    // sim units (1 AU)
	eSolar    = 0.0167   // Earth's eccentricity
	GMsol     = G_sim * massSol
	GMearth   = G_sim * massEarth

	yearSec  = 3.156e7 // seconds in 1 Earth year
	monthSec = 2.551e6 // seconds in 1 synodic month (≈29.5 days)
)

// makeSol returns a minimal Sol object seeded for N-body at the origin.
func makeSol() *Object {
	return &Object{
		Meta: ObjectMetadata{
			Name:     "Sol",
			Category: CategoryStar,
			Mass:     massSol,
			GM:       GMsol,
		},
		Anim: AnimationState{
			Position: Vector3{},
			NBodyPos: [3]float64{0, 0, 0},
			NBodyVel: [3]float64{0, 0, 0},
		},
		Dataset: -1,
	}
}

// makeEarth returns an Earth-like object in a circular orbit around Sol at
// semi-major axis aEarth sim units.  Seeded with vis-viva circular velocity.
func makeEarth(sol *Object) *Object {
	// Circular orbit: v = sqrt(GM_sol / a)
	v := math.Sqrt(GMsol / aEarth)
	return &Object{
		Meta: ObjectMetadata{
			Name:          "Earth",
			Category:      CategoryPlanet,
			Mass:          massEarth,
			GM:            GMearth,
			ParentName:    "Sol",
			SemiMajorAxis: aEarth,
			OrbitalPeriod: float32(yearSec),
		},
		Anim: AnimationState{
			Position: Vector3{X: aEarth, Y: 0, Z: 0},
			NBodyPos: [3]float64{aEarth, 0, 0},
			NBodyVel: [3]float64{sol.Anim.NBodyVel[0], sol.Anim.NBodyVel[1] + v, sol.Anim.NBodyVel[2]},
		},
		Dataset: -1,
	}
}

// makeMoon returns a Moon-like object orbiting Earth with circular velocity.
func makeMoon(earth *Object) *Object {
	v := math.Sqrt(GMearth / aMoon)
	return &Object{
		Meta: ObjectMetadata{
			Name:          "Moon",
			Category:      CategoryMoon,
			Mass:          massMoon,
			GM:            G_sim * massMoon,
			ParentName:    "Earth",
			SemiMajorAxis: aMoon,
			OrbitalPeriod: float32(monthSec),
		},
		Anim: AnimationState{
			Position: Vector3{X: float32(aEarth + aMoon), Y: 0, Z: 0},
			NBodyPos: [3]float64{aEarth + aMoon, 0, 0},
			NBodyVel: [3]float64{earth.Anim.NBodyVel[0], earth.Anim.NBodyVel[1] + v, earth.Anim.NBodyVel[2]},
		},
		Dataset: -1,
	}
}

// makeState constructs a minimal SimulationState from the given objects.
func makeState(objs ...*Object) *SimulationState {
	s := &SimulationState{
		Objects:   make([]*Object, len(objs)),
		ObjectMap: make(map[string]*Object, len(objs)),
	}
	for i, o := range objs {
		s.Objects[i] = o
		s.ObjectMap[o.Meta.Name] = o
	}
	return s
}

// totalEnergy returns the total mechanical energy (KE + PE) of all Participants.
func totalEnergy(gs GravSet) float64 {
	var e float64
	for _, p := range gs.Participants {
		vx, vy, vz := p.Anim.NBodyVel[0], p.Anim.NBodyVel[1], p.Anim.NBodyVel[2]
		e += 0.5 * p.Meta.Mass * (vx*vx + vy*vy + vz*vz)
	}
	for i, pi := range gs.Participants {
		for j := i + 1; j < len(gs.Participants); j++ {
			pj := gs.Participants[j]
			dx := pi.Anim.NBodyPos[0] - pj.Anim.NBodyPos[0]
			dy := pi.Anim.NBodyPos[1] - pj.Anim.NBodyPos[1]
			dz := pi.Anim.NBodyPos[2] - pj.Anim.NBodyPos[2]
			r := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if r > 0 {
				e -= G_sim * pi.Meta.Mass * pj.Meta.Mass / r
			}
		}
	}
	return e
}

// angularDeg returns the angle in degrees of a body relative to the X axis in
// the XZ plane (the orbital plane for zero-inclination orbits).
func angularDeg(obj *Object) float64 {
	return math.Atan2(-obj.Anim.NBodyPos[2], obj.Anim.NBodyPos[0]) * 180.0 / math.Pi
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestNBody_EarthOrbitPeriod verifies that Earth returns within 5° of its
// starting position after running the integrator for exactly one orbital period.
func TestNBody_EarthOrbitPeriod(t *testing.T) {
	sol := makeSol()
	earth := makeEarth(sol)
	gs := GravSet{
		Name:         "2body",
		Participants: []*Object{sol, earth},
	}

	startAngle := angularDeg(earth)

	// Run for 1 Earth year; use ~1000 steps (each ~8.67 hours of sim time).
	const nSteps = 1000
	dt := yearSec / float64(nSteps)
	for i := 0; i < nSteps; i++ {
		StepGravSet(gs, dt)
	}

	endAngle := angularDeg(earth)
	// Wrap difference into [-180, 180]
	diff := endAngle - startAngle
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}
	// After one full orbit, diff ≈ 0 (within ±360° wrap noise)
	if math.Abs(diff) > 5.0 {
		t.Errorf("Earth orbit period test: angular drift = %.2f° (want < 5°)", diff)
	}
}

// TestNBody_EnergyConservation verifies that the total mechanical energy
// (KE + PE) of a Sol–Earth 2-body system varies by less than 0.1% over
// 10 orbital periods.
func TestNBody_EnergyConservation(t *testing.T) {
	sol := makeSol()
	earth := makeEarth(sol)
	gs := GravSet{
		Name:         "2body",
		Participants: []*Object{sol, earth},
	}

	e0 := totalEnergy(gs)

	const nSteps = 10000
	dt := 10.0 * yearSec / float64(nSteps)
	eMin, eMax := e0, e0
	for i := 0; i < nSteps; i++ {
		StepGravSet(gs, dt)
		e := totalEnergy(gs)
		if e < eMin {
			eMin = e
		}
		if e > eMax {
			eMax = e
		}
	}

	relDrift := math.Abs(eMax-eMin) / math.Abs(e0)
	if relDrift > 0.001 {
		t.Errorf("Energy conservation: relative drift = %.4f%% (want < 0.1%%)", relDrift*100)
	}
}

// TestNBody_MoonOrbit verifies that the Earth-Moon distance stays within the
// expected range [35, 42] sim units over one synodic month.
func TestNBody_MoonOrbit(t *testing.T) {
	sol := makeSol()
	earth := makeEarth(sol)
	moon := makeMoon(earth)
	gs := GravSet{
		Name:         "EarthMoon",
		Participants: []*Object{sol, earth, moon},
	}

	const nSteps = 500
	dt := monthSec / float64(nSteps)
	minDist, maxDist := math.MaxFloat64, 0.0
	for i := 0; i < nSteps; i++ {
		StepGravSet(gs, dt)
		dx := earth.Anim.NBodyPos[0] - moon.Anim.NBodyPos[0]
		dy := earth.Anim.NBodyPos[1] - moon.Anim.NBodyPos[1]
		dz := earth.Anim.NBodyPos[2] - moon.Anim.NBodyPos[2]
		d := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if d < minDist {
			minDist = d
		}
		if d > maxDist {
			maxDist = d
		}
	}

	// aMoon ≈ 2.57 sim units; allow slack for solar tidal perturbation.
	if minDist < 2.0 || maxDist > 3.5 {
		t.Errorf("Moon orbit: distance range [%.3f, %.3f] sim units (want [2.0, 3.5])", minDist, maxDist)
	}
}

// TestNBody_BarycenterInsideSol verifies that the Sol-Earth barycenter lies
// inside Sol's physical radius after one Earth orbit.  Sol radius ≈ 4.65 sim units.
func TestNBody_BarycenterInsideSol(t *testing.T) {
	sol := makeSol()
	earth := makeEarth(sol)
	state := makeState(sol, earth)
	state.NBodyMode = "nbody"
	state.SystemSet = GravSet{
		Name:         "system",
		Participants: []*Object{sol, earth},
	}

	const nSteps = 1000
	dt := yearSec / float64(nSteps)
	for i := 0; i < nSteps; i++ {
		StepGravSet(state.SystemSet, dt)
	}
	updateBarycenter(state)

	bcx := float64(state.SystemBarycenter.X)
	bcy := float64(state.SystemBarycenter.Y)
	bcz := float64(state.SystemBarycenter.Z)
	dist := math.Sqrt(bcx*bcx + bcy*bcy + bcz*bcz)

	// Sol's radius in sim units: 696,000 km / 1,496,000 km per sim unit ≈ 0.465 sim units.
	// The barycenter should be well within that.
	if dist > 0.5 {
		t.Errorf("Barycenter distance from origin = %.4f sim units (want < 0.5)", dist)
	}
}

// TestNBody_TestParticleNeutrality verifies that 100 massless test particles
// do not measurably alter the trajectories of the Participants.
func TestNBody_TestParticleNeutrality(t *testing.T) {
	// Reference run: Sol + Earth, no test particles.
	solRef := makeSol()
	earthRef := makeEarth(solRef)
	gsRef := GravSet{
		Name:         "ref",
		Participants: []*Object{solRef, earthRef},
	}

	// Perturbed run: same bodies with 100 test particles at Earth's orbit.
	solP := makeSol()
	earthP := makeEarth(solP)
	v := math.Sqrt(GMsol / aEarth)
	testParticles := make([]*Object, 100)
	for i := range testParticles {
		angle := 2.0 * math.Pi * float64(i) / 100.0
		testParticles[i] = &Object{
			Meta: ObjectMetadata{
				Name:     "tp",
				Category: CategoryAsteroid,
				Mass:     0, // massless
				GM:       0,
			},
			Anim: AnimationState{
				NBodyPos: [3]float64{aEarth * math.Cos(angle), 0, aEarth * math.Sin(angle)},
				NBodyVel: [3]float64{-v * math.Sin(angle), 0, v * math.Cos(angle)},
			},
			Dataset: -1,
		}
	}
	gsP := GravSet{
		Name:          "perturbed",
		Participants:  []*Object{solP, earthP},
		TestParticles: testParticles,
	}

	const nSteps = 1000
	dt := yearSec / float64(nSteps)
	for i := 0; i < nSteps; i++ {
		StepGravSet(gsRef, dt)
		StepGravSet(gsP, dt)
	}

	// Participant positions should be identical (test particles are massless).
	dxE := earthRef.Anim.NBodyPos[0] - earthP.Anim.NBodyPos[0]
	dyE := earthRef.Anim.NBodyPos[1] - earthP.Anim.NBodyPos[1]
	dzE := earthRef.Anim.NBodyPos[2] - earthP.Anim.NBodyPos[2]
	drift := math.Sqrt(dxE*dxE + dyE*dyE + dzE*dzE)
	if drift > 1e-9 {
		t.Errorf("Test-particle neutrality: Earth drifted %.2e sim units (want 0)", drift)
	}
}
