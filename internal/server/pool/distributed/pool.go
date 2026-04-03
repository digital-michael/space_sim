// Package distributed reserves the DistributedPool type for a future
// sharded, multi-node pool implementation. All methods return ErrNotImplemented
// so callers can register the type without enabling it prematurely.
//
// When the implementation is ready, replace the stub method bodies and update
// the factory. No interface changes are required.
package distributed

import (
	"errors"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/google/uuid"
)

// ErrNotImplemented is returned by all Pool methods until this pool type is
// fully implemented.
var ErrNotImplemented = errors.New("distributed pool: not implemented")

// Pool is a stub that satisfies basepool.ObjectPool. It is safe to construct
// but all mutations and queries return ErrNotImplemented.
type Pool struct{}

// NewPool returns the DistributedPool stub.
func NewPool() *Pool {
	return &Pool{}
}

// GetType returns PoolTypeDistributed.
func (p *Pool) GetType() basepool.PoolType {
	return basepool.PoolTypeDistributed
}

// Create is not yet implemented.
func (p *Pool) Create(_ uuid.UUID, _ string, _ map[string]interface{}) error {
	return ErrNotImplemented
}

// Get is not yet implemented.
func (p *Pool) Get(_ uuid.UUID) (interface{}, error) {
	return nil, ErrNotImplemented
}

// Update is not yet implemented.
func (p *Pool) Update(_ uuid.UUID, _ map[string]interface{}) error {
	return ErrNotImplemented
}

// Delete is not yet implemented.
func (p *Pool) Delete(_ uuid.UUID) error {
	return ErrNotImplemented
}

// List returns nil; no objects are tracked by the stub.
func (p *Pool) List() []uuid.UUID {
	return nil
}
