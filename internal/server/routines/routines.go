// Package routines provides the routine library and execution system
package routines

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/digital-michael/space_sim/internal/server/pool"
)

var (
	ErrRoutineNotFound = errors.New("routine not found")
	ErrExecutionFailed = errors.New("routine execution failed")
)

// Routine defines the interface for executable behaviors
type Routine interface {
	// Name returns the routine identifier
	Name() string

	// Execute runs the routine on the given object with delta time
	Execute(obj *pool.Object, dt float32) error
}

// Library manages registered routines
type Library struct {
	routines map[string]Routine
	mu       sync.RWMutex
}

// NewLibrary creates a new routine library
func NewLibrary() *Library {
	lib := &Library{
		routines: make(map[string]Routine),
	}

	// Register built-in routines
	lib.Register(&RotateRoutine{})
	lib.Register(&MoveRoutine{})
	lib.Register(&ScaleRoutine{})

	return lib
}

// Register adds a routine to the library
func (l *Library) Register(routine Routine) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	name := routine.Name()
	if _, exists := l.routines[name]; exists {
		return fmt.Errorf("routine %s already registered", name)
	}

	l.routines[name] = routine
	return nil
}

// Get retrieves a routine by name
func (l *Library) Get(name string) (Routine, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	routine, exists := l.routines[name]
	if !exists {
		return nil, ErrRoutineNotFound
	}

	return routine, nil
}

// Execute runs a routine on an object
func (l *Library) Execute(name string, obj *pool.Object, dt float32) error {
	routine, err := l.Get(name)
	if err != nil {
		return err
	}

	return routine.Execute(obj, dt)
}

// List returns all registered routine names
func (l *Library) List() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.routines))
	for name := range l.routines {
		names = append(names, name)
	}

	return names
}

// Built-in routines

// RotateRoutine rotates an object continuously
type RotateRoutine struct{}

func (r *RotateRoutine) Name() string {
	return "rotate"
}

func (r *RotateRoutine) Execute(obj *pool.Object, dt float32) error {
	// Get rotation speed from properties, default to 90 deg/sec
	speed := float32(90.0)
	if val, ok := obj.Properties["rotationSpeed"].(float64); ok {
		speed = float32(val)
	}

	obj.Rotation.Z += speed * dt
	// Normalize to 0-360
	if obj.Rotation.Z >= 360 {
		obj.Rotation.Z -= 360
	}

	return nil
}

// MoveRoutine moves an object based on velocity
type MoveRoutine struct{}

func (m *MoveRoutine) Name() string {
	return "move"
}

func (m *MoveRoutine) Execute(obj *pool.Object, dt float32) error {
	// MoveRoutine defines velocity; physics integration owns position updates.
	vx := propertyFloat64(obj.Properties, "velocityX", 0)
	vy := propertyFloat64(obj.Properties, "velocityY", 0)
	vz := propertyFloat64(obj.Properties, "velocityZ", 0)

	obj.Properties["velocityX"] = vx
	obj.Properties["velocityY"] = vy
	obj.Properties["velocityZ"] = vz

	return nil
}

func propertyFloat64(properties map[string]interface{}, key string, defaultValue float64) float64 {
	value, exists := properties[key]
	if !exists {
		return defaultValue
	}

	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	default:
		return defaultValue
	}
}

// ScaleRoutine pulses object scale
type ScaleRoutine struct{}

func (s *ScaleRoutine) Name() string {
	return "scale"
}

func (s *ScaleRoutine) Execute(obj *pool.Object, dt float32) error {
	// Get pulse parameters from properties
	min := float32(0.5)
	max := float32(1.5)
	speed := float32(1.0)

	if val, ok := obj.Properties["scaleMin"].(float64); ok {
		min = float32(val)
	}
	if val, ok := obj.Properties["scaleMax"].(float64); ok {
		max = float32(val)
	}
	if val, ok := obj.Properties["scaleSpeed"].(float64); ok {
		speed = float32(val)
	}

	// Simple sine wave pulse
	phase, _ := obj.Properties["scalePhase"].(float64)
	phase += float64(speed * dt)
	if phase >= 360 {
		phase -= 360
	}
	obj.Properties["scalePhase"] = phase

	// Calculate scale based on sine wave
	scale := min + (max-min)*(float32(math.Sin(phase*math.Pi/180))+1)/2
	obj.Scale.X = scale
	obj.Scale.Y = scale
	obj.Scale.Z = scale

	return nil
}
