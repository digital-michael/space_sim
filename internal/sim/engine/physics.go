package engine

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// Simulation runs the physics simulation in a background goroutine.
type Simulation struct {
	state         *DoubleBuffer
	hz            float64
	stopCh        chan struct{}
	cmdCh         chan SimCommand
	speedChangeCh chan float64
	speedMu       sync.RWMutex
	speed         float64
	updateCounter int

	// applyCommand is called by the simulation loop when a SimCommand is
	// dequeued. Injected by the caller so the engine stays decoupled from
	// domain logic (belt allocation, physics model switching, etc.).
	applyCommand func(SimCommand)

	// postTickHook is called once per ticker interval after all accumulated
	// physics steps complete. Runs on the sim goroutine. Used by world.World
	// to build and atomically publish a ready-to-render snapshot so the main
	// thread never has to lock or clone.
	postTickHook func()
}

// NewSimulation creates a simulation from an already-loaded state.
// applyCommandFn is called inside the simulation loop whenever a SimCommand
// is dequeued; pass nil if no command handling is needed.
func NewSimulation(state *SimulationState, hz float64, applyCommandFn func(SimCommand)) *Simulation {
	// Prime mean anomalies to the current epoch so orbits start at today's
	// positions rather than at the J2000 reference.
	for _, obj := range state.Objects {
		if obj.Meta.OrbitalPeriod > 0 {
			n := float32(2.0 * math.Pi / float64(obj.Meta.OrbitalPeriod))
			obj.Anim.MeanAnomaly = obj.Meta.MeanAnomalyAtEpoch + n*float32(state.Time)
			twoPi := float32(2.0 * math.Pi)
			obj.Anim.MeanAnomaly = float32(math.Mod(float64(obj.Anim.MeanAnomaly), float64(twoPi)))
			if obj.Anim.MeanAnomaly < 0 {
				obj.Anim.MeanAnomaly += twoPi
			}
		}
	}

	// N-body systems use Clone-based swap so the back buffer pointer stays
	// stable (SystemSet holds raw *Object pointers into the back buffer).
	// Keplerian systems use zero-allocation InPlaceSwap.
	db := NewDoubleBuffer(state)
	if state.NBodyMode != "nbody" {
		db.EnableInPlaceSwap()
		fmt.Printf("✓ Enabled in-place swap optimization (zero-allocation mode)\n")
	} else {
		fmt.Printf("✓ N-body mode: using clone-based swap for pointer stability\n")
	}

	sim := &Simulation{
		state:         db,
		hz:            hz,
		stopCh:        make(chan struct{}),
		cmdCh:         make(chan SimCommand, 1),
		speedChangeCh: make(chan float64, 1),
		speed:         1.0,
	}
	if applyCommandFn != nil {
		sim.applyCommand = applyCommandFn
	} else {
		sim.applyCommand = func(SimCommand) {}
	}
	sim.postTickHook = func() {}

	// Seed N-body state on the back buffer when N-body mode is active.
	// Must happen after NewDoubleBuffer so we initialise the correct buffer.
	if state.NBodyMode == "nbody" {
		back := db.GetBack()
		back.SystemSet = BuildSystemSet(back)
		initNBody(back)
	}

	return sim
}

// GetState returns the double buffer for renderer access.
func (s *Simulation) GetState() *DoubleBuffer {
	return s.state
}

// SetPostTickHook registers a function that is called once per ticker
// interval, after all accumulated physics steps complete. It runs on the
// simulation goroutine. Only one hook is supported; calling SetPostTickHook
// again replaces the previous hook.
func (s *Simulation) SetPostTickHook(fn func()) {
	if fn == nil {
		s.postTickHook = func() {}
		return
	}
	s.postTickHook = fn
}

// Start begins the simulation loop; blocks until ctx is cancelled or Stop is called.
func (s *Simulation) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(s.hz))
	defer ticker.Stop()

	dt := 1.0 / s.hz
	var accumulatedTime float64

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			select {
			case newSpeed := <-s.speedChangeCh:
				s.speedMu.Lock()
				s.speed = newSpeed
				s.speedMu.Unlock()
			default:
			}

			s.speedMu.RLock()
			currentSpeed := s.speed
			s.speedMu.RUnlock()

			if currentSpeed == 0.0 {
				continue
			}

			accumulatedTime += dt * currentSpeed
			for accumulatedTime >= dt {
				accumulatedTime -= dt

				select {
				case cmd := <-s.cmdCh:
					s.applyCommand(cmd)
				default:
				}

				s.updateCounter++
				s.update(dt)
			}
			// Publish a ready-to-render snapshot once per ticker interval,
			// after all accumulated steps. Runs on the sim goroutine so the
			// main thread never needs to lock or clone.
			s.postTickHook()
		}
	}
}

// Stop signals the simulation loop to exit.
func (s *Simulation) Stop() {
	close(s.stopCh)
}

// SetSpeed sets the simulation speed multiplier (0.0 = paused, 1.0 = real time).
func (s *Simulation) SetSpeed(speed float64) {
	select {
	case s.speedChangeCh <- speed:
	default:
	}
}

// GetSpeed returns the current simulation speed multiplier.
func (s *Simulation) GetSpeed() float64 {
	s.speedMu.RLock()
	defer s.speedMu.RUnlock()
	return s.speed
}

// SetWorkerCount sets the number of physics worker threads.
func (s *Simulation) SetWorkerCount(count int) {
	back := s.state.GetBack()
	back.NumWorkers = count
	s.state.Swap()
}

// DisableLocking disables double-buffer mutex operations (benchmarking only — unsafe).
func (s *Simulation) DisableLocking() {
	s.state.DisableLocking()
}

// SetAsteroidDataset queues an async dataset change request.
func (s *Simulation) SetAsteroidDataset(dataset AsteroidDataset) {
	select {
	case s.cmdCh <- DatasetChangeCommand{Dataset: dataset}:
	default:
	}
}

// update performs one simulation step.
func (s *Simulation) update(dt float64) {
	back := s.state.GetBack()
	scaledDt := dt * float64(back.SecondsPerSecond)

	back.Time += scaledDt
	back.DeltaTime = scaledDt

	if back.NBodyMode == "nbody" {
		// ── N-body path ──────────────────────────────────────────────────────
		// Leapfrog DKD step for all named bodies (stars, planets, moons, …).
		StepGravSet(back.SystemSet, scaledDt)

		// Belt children (Dataset >= 0) still follow Keplerian orbits whose
		// center tracks the N-body updated parent position.
		children := back.GetChildren()
		for _, obj := range children {
			// Named bodies (Dataset<0) are integrated inside SystemSet, except
			// rings which orbit via Keplerian and need their center updated.
			if obj.Dataset < 0 && obj.Meta.Category != CategoryRing {
				continue
			}
			if !obj.Visible {
				continue
			}
			if parent := back.ObjectMap[obj.Meta.ParentName]; parent != nil {
				obj.Anim.OrbitCenter = parent.Anim.Position
			}
		}
		for _, obj := range children {
			if obj.Dataset < 0 && obj.Meta.Category != CategoryRing {
				continue
			}
			s.updateObject(obj, float32(scaledDt))
		}

		updateBarycenter(back)
		s.state.Swap()
		return
	}

	// ── Keplerian path (default) ──────────────────────────────────────────────
	parents := back.GetParents()
	children := back.GetChildren()

	numWorkers := back.NumWorkers
	parentsPerWorker := (len(parents) + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		start := i * parentsPerWorker
		end := start + parentsPerWorker
		if end > len(parents) {
			end = len(parents)
		}
		if start >= len(parents) {
			break
		}
		wg.Add(1)
		go func(objs []*Object) {
			defer wg.Done()
			for _, obj := range objs {
				s.updateObject(obj, float32(scaledDt))
			}
		}(parents[start:end])
	}
	wg.Wait()

	for _, obj := range children {
		if !obj.Visible && obj.Dataset >= 0 {
			continue
		}
		if parent := back.ObjectMap[obj.Meta.ParentName]; parent != nil {
			obj.Anim.OrbitCenter = parent.Anim.Position
		}
	}

	childrenPerWorker := (len(children) + numWorkers - 1) / numWorkers
	for i := 0; i < numWorkers; i++ {
		start := i * childrenPerWorker
		end := start + childrenPerWorker
		if end > len(children) {
			end = len(children)
		}
		if start >= len(children) {
			break
		}
		wg.Add(1)
		go func(objs []*Object) {
			defer wg.Done()
			for _, obj := range objs {
				s.updateObject(obj, float32(scaledDt))
			}
		}(children[start:end])
	}
	wg.Wait()

	s.state.Swap()
}

// updateObject updates a single object's position using Keplerian mechanics.
func (s *Simulation) updateObject(obj *Object, dt float32) {
	if !obj.Visible {
		return
	}
	if obj.Meta.ParentName != "" {
		if obj.Meta.OrbitRadius == 0 {
			obj.Anim.Position = obj.Anim.OrbitCenter
			return
		}
	}
	if obj.Meta.OrbitalPeriod == 0 || obj.Meta.SemiMajorAxis == 0 {
		return
	}

	periodSeconds := float64(obj.Meta.OrbitalPeriod)
	meanMotion := (2.0 * math.Pi) / periodSeconds
	obj.Anim.MeanAnomaly += float32(meanMotion * float64(dt))
	for obj.Anim.MeanAnomaly >= 2*math.Pi {
		obj.Anim.MeanAnomaly -= 2 * math.Pi
	}

	eccentricAnomaly := solveKeplersEquation(obj.Anim.MeanAnomaly, obj.Meta.Eccentricity)
	obj.Anim.TrueAnomaly = calculateTrueAnomaly(eccentricAnomaly, obj.Meta.Eccentricity)
	obj.Anim.OrbitAngle = obj.Anim.TrueAnomaly

	e := float64(obj.Meta.Eccentricity)
	nu := float64(obj.Anim.TrueAnomaly)
	a := float64(obj.Meta.SemiMajorAxis)
	radius := float32(a * (1 - e*e) / (1 + e*math.Cos(nu)))

	cosNu := float32(math.Cos(nu))
	sinNu := float32(math.Sin(nu))
	xOrbit := radius * cosNu
	yOrbit := radius * sinNu

	pos3D := rotateOrbit(xOrbit, yOrbit, obj.Meta.ArgPeriapsis, obj.Meta.Inclination, obj.Meta.LongAscendingNode)
	obj.Anim.Position = Vector3{
		X: obj.Anim.OrbitCenter.X + pos3D.X,
		Y: obj.Anim.OrbitCenter.Y + pos3D.Y + obj.Anim.OrbitYOffset,
		Z: obj.Anim.OrbitCenter.Z + pos3D.Z,
	}

	GM := 1.0
	speed := float32(math.Sqrt(GM * (2.0/float64(radius) - 1.0/a)))
	velOrbit := Vector3{X: -sinNu * speed, Y: cosNu * speed, Z: 0}
	obj.Anim.Velocity = rotateOrbit(velOrbit.X, velOrbit.Y, obj.Meta.ArgPeriapsis, obj.Meta.Inclination, obj.Meta.LongAscendingNode)
}

// solveKeplersEquation solves M = E - e*sin(E) for E using Newton-Raphson.
func solveKeplersEquation(M, e float32) float32 {
	E := M
	tolerance := float32(1e-6)
	for i := 0; i < 10; i++ {
		sinE := float32(math.Sin(float64(E)))
		f := E - e*sinE - M
		fPrime := 1 - e*float32(math.Cos(float64(E)))
		delta := f / fPrime
		E -= delta
		if math.Abs(float64(delta)) < float64(tolerance) {
			break
		}
	}
	return E
}

// calculateTrueAnomaly converts eccentric anomaly E to true anomaly ν.
func calculateTrueAnomaly(E, e float32) float32 {
	halfE := float64(E) / 2.0
	factor := math.Sqrt((1 + float64(e)) / (1 - float64(e)))
	halfNu := math.Atan(factor * math.Tan(halfE))
	return float32(2.0 * halfNu)
}

// rotateOrbit applies the three Keplerian rotation matrices:
// R_z(Ω) * R_x(i) * R_z(ω) * [x, y, 0]
func rotateOrbit(x, y float32, argPeri, incl, longNode float32) Vector3 {
	cosW := float32(math.Cos(float64(argPeri)))
	sinW := float32(math.Sin(float64(argPeri)))
	x1 := x*cosW - y*sinW
	y1 := x*sinW + y*cosW
	z1 := float32(0.0)

	cosI := float32(math.Cos(float64(incl)))
	sinI := float32(math.Sin(float64(incl)))
	x2 := x1
	y2 := y1*cosI - z1*sinI
	z2 := y1*sinI + z1*cosI

	cosO := float32(math.Cos(float64(longNode)))
	sinO := float32(math.Sin(float64(longNode)))
	x3 := x2*cosO - y2*sinO
	y3 := x2*sinO + y2*cosO
	z3 := z2

	return Vector3{X: x3, Y: z3, Z: -y3} // Swap Y/Z to match Y-up; negate Z for CCW-from-north orbits
}

// ─── N-body initialisation ────────────────────────────────────────────────────

// initNBody seeds NBodyPos and NBodyVel for all bodies in state.SystemSet.
// Positions are computed from the current (primed) MeanAnomaly via Kepler's
// equation.  Velocities are derived from the vis-viva relation and accumulated
// inertially (child's velocity = orbital velocity + parent's inertial velocity).
// Bodies are processed in three topological passes: level-0 (no parent),
// level-1 (parent is level-0), level-2 (parent is level-1), covering stars →
// planets → moons without requiring the Participants slice to be ordered.
func initNBody(state *SimulationState) {
	processed := make(map[string]bool, len(state.SystemSet.Participants))

	for pass := 0; pass < 3; pass++ {
		for _, obj := range state.SystemSet.Participants {
			pName := obj.Meta.ParentName

			if pass == 0 && pName != "" {
				continue
			}
			if pass > 0 && (pName == "" || !processed[pName]) {
				continue
			}

			// ── Position ────────────────────────────────────────────────────
			if obj.Meta.SemiMajorAxis > 0 && obj.Meta.OrbitalPeriod > 0 {
				E := solveKeplersEquation(obj.Anim.MeanAnomaly, obj.Meta.Eccentricity)
				nu := calculateTrueAnomaly(E, obj.Meta.Eccentricity)
				obj.Anim.TrueAnomaly = nu

				e := float64(obj.Meta.Eccentricity)
				a := float64(obj.Meta.SemiMajorAxis)
				nu64 := float64(nu)
				r := float32(a * (1 - e*e) / (1 + e*math.Cos(nu64)))

				pos3D := rotateOrbit(
					r*float32(math.Cos(nu64)),
					r*float32(math.Sin(nu64)),
					obj.Meta.ArgPeriapsis,
					obj.Meta.Inclination,
					obj.Meta.LongAscendingNode,
				)
				center := Vector3{}
				if pName != "" {
					if parent := state.ObjectMap[pName]; parent != nil {
						center = parent.Anim.Position
					}
				}
				obj.Anim.Position = Vector3{
					X: center.X + pos3D.X,
					Y: center.Y + pos3D.Y,
					Z: center.Z + pos3D.Z,
				}
			}

			obj.Anim.NBodyPos = [3]float64{
				float64(obj.Anim.Position.X),
				float64(obj.Anim.Position.Y),
				float64(obj.Anim.Position.Z),
			}

			// ── Velocity ────────────────────────────────────────────────────
			if pName != "" && obj.Meta.SemiMajorAxis > 0 {
				if parent := state.ObjectMap[pName]; parent != nil && parent.Meta.GM > 0 {
					a := float64(obj.Meta.SemiMajorAxis)
					e := float64(obj.Meta.Eccentricity)
					nu := float64(obj.Anim.TrueAnomaly)
					p := a * (1 - e*e)
					sqGMp := math.Sqrt(parent.Meta.GM / p)
					// In-plane velocity components (perifocal frame)
					vx := -sqGMp * math.Sin(nu)
					vy := sqGMp * (e + math.Cos(nu))
					// Rotate to 3D inertial frame
					rotV := rotateOrbit(
						float32(vx), float32(vy),
						obj.Meta.ArgPeriapsis,
						obj.Meta.Inclination,
						obj.Meta.LongAscendingNode,
					)
					// Add parent's inertial velocity for correct heliocentric frame
					obj.Anim.NBodyVel = [3]float64{
						float64(rotV.X) + parent.Anim.NBodyVel[0],
						float64(rotV.Y) + parent.Anim.NBodyVel[1],
						float64(rotV.Z) + parent.Anim.NBodyVel[2],
					}
				}
			} else {
				// Top-level body: seed from Anim.Velocity (e.g. velocity_override or zero).
				obj.Anim.NBodyVel = [3]float64{
					float64(obj.Anim.Velocity.X),
					float64(obj.Anim.Velocity.Y),
					float64(obj.Anim.Velocity.Z),
				}
			}

			processed[obj.Meta.Name] = true
		}
	}
}

// updateBarycenter computes the mass-weighted center of all Participants and
// stores it in state.SystemBarycenter (float32 Vector3 for renderer access).
func updateBarycenter(state *SimulationState) {
	var cx, cy, cz, totalMass float64
	for _, obj := range state.SystemSet.Participants {
		m := obj.Meta.Mass
		cx += m * obj.Anim.NBodyPos[0]
		cy += m * obj.Anim.NBodyPos[1]
		cz += m * obj.Anim.NBodyPos[2]
		totalMass += m
	}
	if totalMass > 0 {
		state.SystemBarycenter = Vector3{
			X: float32(cx / totalMass),
			Y: float32(cy / totalMass),
			Z: float32(cz / totalMass),
		}
	}
}
