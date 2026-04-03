// Package factory constructs ObjectPool implementations by PoolType.
// It is the single place that knows about all concrete pool packages so
// callers depend only on the basepool.ObjectPool interface and the PoolType
// enum — neither of which imports back into this package.
//
// Usage:
//
//	p, err := factory.New(basepool.PoolTypeSimple)
//	if err != nil { ... }
//	p.Create(id, "planet", nil)
package factory

import (
	"fmt"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/digital-michael/space_sim/internal/server/pool/distributed"
	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/digital-michael/space_sim/internal/server/pool/simple"
)

// New returns a ready-to-use ObjectPool for the requested PoolType.
// Returns an error for PoolTypeHybrid (not yet implemented) and any unknown
// PoolType value.
func New(pt basepool.PoolType) (basepool.ObjectPool, error) {
	switch pt {
	case basepool.PoolTypeSimple:
		return simple.NewPool(), nil
	case basepool.PoolTypeGroup:
		return group.NewPool(), nil
	case basepool.PoolTypeDistributed:
		return distributed.NewPool(), nil
	default:
		return nil, fmt.Errorf("factory: no implementation for pool type %v", pt)
	}
}
