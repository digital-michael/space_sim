// Package pool manages the object pool for the simulation
package pool

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrObjectNotFound = errors.New("object not found")
	ErrPoolFull       = errors.New("object pool is full")
)

// Object represents a simulation object in the pool
type Object struct {
	GUID       string
	Type       string
	Position   Position
	Rotation   Rotation
	Scale      Scale
	Properties map[string]interface{}
	Routines   []string
	mu         sync.RWMutex
}

// Position represents 3D position
type Position struct {
	X, Y, Z float32
}

// Rotation represents 3D rotation (euler angles)
type Rotation struct {
	X, Y, Z float32
}

// Scale represents 3D scale
type Scale struct {
	X, Y, Z float32
}

// Pool manages a collection of simulation objects
type Pool struct {
	objects    map[string]*Object
	maxObjects int
	mu         sync.RWMutex
}

// New creates a new object pool
func New(initialCapacity, maxObjects int) *Pool {
	return &Pool{
		objects:    make(map[string]*Object, initialCapacity),
		maxObjects: maxObjects,
	}
}

// Add adds an object to the pool
func (p *Pool) Add(obj *Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.objects) >= p.maxObjects {
		return ErrPoolFull
	}

	// Generate GUID if not provided
	if obj.GUID == "" {
		obj.GUID = uuid.New().String()
	}

	p.objects[obj.GUID] = obj
	return nil
}

// Get retrieves an object by GUID
func (p *Pool) Get(guid string) (*Object, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	obj, exists := p.objects[guid]
	if !exists {
		return nil, ErrObjectNotFound
	}

	return obj, nil
}

// Update updates an existing object
func (p *Pool) Update(guid string, obj *Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.objects[guid]; !exists {
		return ErrObjectNotFound
	}

	obj.GUID = guid
	p.objects[guid] = obj
	return nil
}

// Delete removes an object from the pool
func (p *Pool) Delete(guid string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.objects[guid]; !exists {
		return ErrObjectNotFound
	}

	delete(p.objects, guid)
	return nil
}

// List returns all objects, optionally filtered by type
func (p *Pool) List(typeFilter string) []*Object {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Object
	for _, obj := range p.objects {
		if typeFilter == "" || obj.Type == typeFilter {
			result = append(result, obj)
		}
	}

	return result
}

// Count returns the number of objects in the pool
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.objects)
}

// Clear removes all objects from the pool
func (p *Pool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.objects = make(map[string]*Object)
}

// NewObject creates a new object with default values
func NewObject(objType string) *Object {
	return &Object{
		GUID:       uuid.New().String(),
		Type:       objType,
		Position:   Position{X: 0, Y: 0, Z: 0},
		Rotation:   Rotation{X: 0, Y: 0, Z: 0},
		Scale:      Scale{X: 1, Y: 1, Z: 1},
		Properties: make(map[string]interface{}),
		Routines:   []string{},
	}
}
