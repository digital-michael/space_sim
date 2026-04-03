package engine

import (
	"sync"
	"time"
)

// SimulationState holds all objects and timing information for one simulation frame.
type SimulationState struct {
	Objects          []*Object          // All objects in the simulation
	ObjectMap        map[string]*Object // Fast lookup by name
	Time             float64            // Simulation time in seconds since J2000.0
	DeltaTime        float64            // Time since last update (seconds)
	SecondsPerSecond float32            // Simulation seconds per real second
	NumWorkers       int                // Number of parallel physics worker threads

	// Asteroid dataset management
	CurrentDataset    AsteroidDataset          // Currently active dataset
	AllocatedDatasets map[AsteroidDataset]bool // Which datasets have been allocated

	// Belt configurations from JSON (for dynamic allocation)
	AsteroidBeltConfig *FeatureConfig // Asteroid belt config (set by loader)
	KuiperBeltConfig   *FeatureConfig // Kuiper belt config (set by loader)

	// NavigationOrder is the ordered list of categories available as UI tabs.
	// Populated by the loader from the types it encounters in the dataset.
	NavigationOrder []ObjectCategory

	// Cached arrays for performance (avoid repeated categorisation loop)
	parents  []*Object
	children []*Object
	dirty    bool

	// Strategy 6: Object Pooling (future)
	UseObjectPool bool
}

// NewSimulationState creates an empty simulation state initialised to now.
func NewSimulationState() *SimulationState {
	j2000Epoch := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	secondsSinceEpoch := time.Since(j2000Epoch).Seconds()

	return &SimulationState{
		Objects:           make([]*Object, 0, 32),
		ObjectMap:         make(map[string]*Object),
		Time:              secondsSinceEpoch,
		DeltaTime:         0.0,
		SecondsPerSecond:  1.0,
		NumWorkers:        4,
		CurrentDataset:    AsteroidDatasetSmall,
		AllocatedDatasets: make(map[AsteroidDataset]bool),
		NavigationOrder:   make([]ObjectCategory, 0, 8),
	}
}

// AddObject adds an object to the simulation.
func (s *SimulationState) AddObject(obj *Object) {
	s.Objects = append(s.Objects, obj)
	s.ObjectMap[obj.Meta.Name] = obj
	s.dirty = true
}

// GetObject retrieves an object by name (returns nil if not found).
func (s *SimulationState) GetObject(name string) *Object {
	return s.ObjectMap[name]
}

// rebuildCategories separates objects into parents and children.
func (s *SimulationState) rebuildCategories() {
	if !s.dirty {
		return
	}
	s.parents = s.parents[:0]
	s.children = s.children[:0]
	for _, obj := range s.Objects {
		if obj.Meta.ParentName == "" {
			s.parents = append(s.parents, obj)
		} else {
			s.children = append(s.children, obj)
		}
	}
	s.dirty = false
}

// GetParents returns cached parent objects (rebuilds if necessary).
func (s *SimulationState) GetParents() []*Object {
	s.rebuildCategories()
	return s.parents
}

// GetChildren returns cached child objects (rebuilds if necessary).
func (s *SimulationState) GetChildren() []*Object {
	s.rebuildCategories()
	return s.children
}

// Clone creates a deep copy of the state for rendering.
// Strategy 5: lazy clone — only deep-clone visible objects.
func (s *SimulationState) Clone() *SimulationState {
	return s.clone(nil)
}

// CloneWithPool creates a deep copy of the state using pooled objects for
// transient front-buffer clones.
func (s *SimulationState) CloneWithPool(pool *ObjectPool) *SimulationState {
	return s.clone(pool)
}

func (s *SimulationState) clone(pool *ObjectPool) *SimulationState {
	cloned := &SimulationState{
		Objects:            make([]*Object, len(s.Objects)),
		ObjectMap:          make(map[string]*Object, len(s.Objects)),
		Time:               s.Time,
		DeltaTime:          s.DeltaTime,
		SecondsPerSecond:   s.SecondsPerSecond,
		NumWorkers:         s.NumWorkers,
		CurrentDataset:     s.CurrentDataset,
		AllocatedDatasets:  make(map[AsteroidDataset]bool, len(s.AllocatedDatasets)),
		AsteroidBeltConfig: s.AsteroidBeltConfig, // immutable after load
		KuiperBeltConfig:   s.KuiperBeltConfig,   // immutable after load
	}

	for k, v := range s.AllocatedDatasets {
		cloned.AllocatedDatasets[k] = v
	}
	cloned.dirty = true

	for i, obj := range s.Objects {
		if obj.Dataset == -1 || obj.Visible {
			var newObj *Object
			if pool != nil {
				newObj = pool.Borrow()
				newObj.Meta = obj.Meta
				newObj.Anim = obj.Anim
				newObj.Visible = obj.Visible
				newObj.Dataset = obj.Dataset
			} else {
				newObj = &Object{
					Meta:    obj.Meta,
					Anim:    obj.Anim,
					Visible: obj.Visible,
					Dataset: obj.Dataset,
				}
			}
			cloned.Objects[i] = newObj
			cloned.ObjectMap[newObj.Meta.Name] = newObj
		} else {
			cloned.Objects[i] = obj
			cloned.ObjectMap[obj.Meta.Name] = obj
		}
	}

	return cloned
}

// ReleasePooledObjects returns any pooled clone objects held by the state.
func (s *SimulationState) ReleasePooledObjects(pool *ObjectPool) {
	if pool == nil {
		return
	}
	for i, obj := range s.Objects {
		if obj != nil && obj.pooled {
			pool.Return(obj)
			s.Objects[i] = nil
		}
	}
	s.ObjectMap = nil
}

// DoubleBuffer provides lock-safe front/back state access.
// The simulation writes to the back buffer; the renderer reads from the front.
type DoubleBuffer struct {
	front           *SimulationState
	back            *SimulationState
	objectPool      *ObjectPool
	mu              sync.RWMutex
	lockingDisabled bool
	useInPlaceSwap  bool
}

// NewDoubleBuffer creates a double buffer seeded with the given state.
func NewDoubleBuffer(initial *SimulationState) *DoubleBuffer {
	return newDoubleBufferWithPool(initial, globalObjectPool)
}

func newDoubleBufferWithPool(initial *SimulationState, pool *ObjectPool) *DoubleBuffer {
	return &DoubleBuffer{
		front:      initial,
		back:       initial.Clone(),
		objectPool: pool,
	}
}

// GetFront returns the front buffer (renderer use).
func (db *DoubleBuffer) GetFront() *SimulationState {
	if !db.lockingDisabled {
		db.mu.RLock()
		defer db.mu.RUnlock()
	}
	return db.front
}

// LockFront acquires a read lock and returns the front buffer.
// Caller MUST call UnlockFront when done.
func (db *DoubleBuffer) LockFront() *SimulationState {
	if !db.lockingDisabled {
		db.mu.RLock()
	}
	return db.front
}

// UnlockFront releases the read lock acquired by LockFront.
func (db *DoubleBuffer) UnlockFront() {
	if !db.lockingDisabled {
		db.mu.RUnlock()
	}
}

// GetBack returns the back buffer for writing (simulation use only).
func (db *DoubleBuffer) GetBack() *SimulationState {
	return db.back
}

// Swap makes the back buffer visible to the renderer.
func (db *DoubleBuffer) Swap() {
	if db.useInPlaceSwap {
		db.SwapInPlace()
		return
	}
	if !db.lockingDisabled {
		db.mu.Lock()
		defer db.mu.Unlock()
	}
	oldFront := db.front
	db.front = db.back.CloneWithPool(db.objectPool)
	if oldFront != nil {
		oldFront.ReleasePooledObjects(db.objectPool)
	}
}

// SwapInPlace performs a zero-allocation swap by reusing buffers.
// Only safe when no objects are added or removed after initialisation.
func (db *DoubleBuffer) SwapInPlace() {
	if !db.lockingDisabled {
		db.mu.Lock()
		defer db.mu.Unlock()
	}

	savedSPS := db.back.SecondsPerSecond
	savedWorkers := db.back.NumWorkers

	db.front, db.back = db.back, db.front

	db.back.SecondsPerSecond = savedSPS
	db.back.NumWorkers = savedWorkers
	db.back.Time = db.front.Time
	db.back.DeltaTime = db.front.DeltaTime
	db.back.CurrentDataset = db.front.CurrentDataset

	for i := range db.back.Objects {
		db.back.Objects[i].Anim = db.front.Objects[i].Anim
		db.back.Objects[i].Visible = db.front.Objects[i].Visible
	}
}

// DisableLocking disables all mutex operations (benchmarking only — unsafe).
func (db *DoubleBuffer) DisableLocking() {
	db.lockingDisabled = true
}

// EnableInPlaceSwap enables zero-allocation swapping.
func (db *DoubleBuffer) EnableInPlaceSwap() {
	db.useInPlaceSwap = true
}

// DisableInPlaceSwap re-enables Clone-based swapping (supports dynamic object add/remove).
func (db *DoubleBuffer) DisableInPlaceSwap() {
	db.useInPlaceSwap = false
}

// LockFrontWrite acquires a write lock and returns the front buffer.
// Use this when the simulation goroutine must mutate front buffer visibility
// (e.g. during a dataset change). Caller MUST call UnlockFrontWrite when done.
func (db *DoubleBuffer) LockFrontWrite() *SimulationState {
	if !db.lockingDisabled {
		db.mu.Lock()
	}
	return db.front
}

// UnlockFrontWrite releases the write lock acquired by LockFrontWrite.
func (db *DoubleBuffer) UnlockFrontWrite() {
	if !db.lockingDisabled {
		db.mu.Unlock()
	}
}
