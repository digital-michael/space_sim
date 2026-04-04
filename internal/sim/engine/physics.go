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

	db := NewDoubleBuffer(state)
	db.EnableInPlaceSwap()
	fmt.Printf("✓ Enabled in-place swap optimization (zero-allocation mode)\n")

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
	return sim
}

// GetState returns the double buffer for renderer access.
func (s *Simulation) GetState() *DoubleBuffer {
	return s.state
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

	return Vector3{X: x3, Y: z3, Z: y3} // Swap Y/Z to match Y-up coordinate system
}
