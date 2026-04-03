// Package simple provides a flat, map-backed implementation of the ObjectPool
// interface. It stores objects by UUID with no grouping or hierarchy.
// Use this pool when group membership and DAG traversal are not required.
package simple

import (
	"errors"
	"fmt"
	"sync"
	"time"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/google/uuid"
)

var (
	// ErrObjectNotFound is returned when a requested UUID is not in the pool.
	ErrObjectNotFound = errors.New("simple pool: object not found")
	// ErrObjectExists is returned when Create is called with a UUID already in use.
	ErrObjectExists = errors.New("simple pool: object already exists")
)

// ObjectDefinition is the value type stored in SimplePool. It mirrors the
// shape used by the group pool so consumers can switch pool types without
// changing their definition handling code.
type ObjectDefinition struct {
	ID         uuid.UUID              `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// Clone returns a deep copy. Properties values are shallow-cloned; do not
// store pointer types in Properties if mutation isolation is required.
func (d *ObjectDefinition) Clone() *ObjectDefinition {
	props := make(map[string]interface{}, len(d.Properties))
	for k, v := range d.Properties {
		props[k] = v
	}
	return &ObjectDefinition{
		ID:         d.ID,
		Type:       d.Type,
		Properties: props,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// Pool is a thread-safe, flat object pool that implements basepool.ObjectPool.
type Pool struct {
	mu      sync.RWMutex
	objects map[uuid.UUID]*ObjectDefinition
}

// NewPool returns an empty SimplePool.
func NewPool() *Pool {
	return &Pool{
		objects: make(map[uuid.UUID]*ObjectDefinition),
	}
}

// GetType returns PoolTypeSimple.
func (p *Pool) GetType() basepool.PoolType {
	return basepool.PoolTypeSimple
}

// Create inserts a new object. Returns ErrObjectExists if id is already present.
// Returns an error if id is uuid.Nil or objType is empty.
func (p *Pool) Create(id uuid.UUID, objType string, properties map[string]interface{}) error {
	if id == uuid.Nil {
		return fmt.Errorf("simple pool: object ID cannot be nil")
	}
	if objType == "" {
		return fmt.Errorf("simple pool: object type cannot be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.objects[id]; exists {
		return ErrObjectExists
	}

	now := time.Now()
	props := make(map[string]interface{}, len(properties))
	for k, v := range properties {
		props[k] = v
	}

	p.objects[id] = &ObjectDefinition{
		ID:         id,
		Type:       objType,
		Properties: props,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return nil
}

// Get returns a Clone of the stored object so callers cannot mutate pool state
// through the returned pointer. Returns ErrObjectNotFound if id is absent.
func (p *Pool) Get(id uuid.UUID) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	def, exists := p.objects[id]
	if !exists {
		return nil, ErrObjectNotFound
	}
	return def.Clone(), nil
}

// Update merges the supplied properties into the stored object's property map.
// Returns ErrObjectNotFound if id is absent.
func (p *Pool) Update(id uuid.UUID, properties map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	def, exists := p.objects[id]
	if !exists {
		return ErrObjectNotFound
	}

	if def.Properties == nil {
		def.Properties = make(map[string]interface{}, len(properties))
	}
	for k, v := range properties {
		def.Properties[k] = v
	}
	def.UpdatedAt = time.Now()
	return nil
}

// Delete removes an object by UUID. Returns ErrObjectNotFound if absent.
func (p *Pool) Delete(id uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.objects[id]; !exists {
		return ErrObjectNotFound
	}
	delete(p.objects, id)
	return nil
}

// List returns the UUIDs of all objects currently in the pool.
// Order is not guaranteed.
func (p *Pool) List() []uuid.UUID {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(p.objects))
	for id := range p.objects {
		ids = append(ids, id)
	}
	return ids
}
