package engine

import (
	"sync"
	"sync/atomic"
)

// ObjectPool manages a pool of reusable Object instances to reduce GC pressure
// during clone-based front-buffer swaps.
type ObjectPool struct {
	pool           sync.Pool
	borrows        atomic.Int64
	returns        atomic.Int64
	newAllocations atomic.Int64
}

// NewObjectPool creates a new reusable object pool.
func NewObjectPool() *ObjectPool {
	p := &ObjectPool{}
	p.pool.New = func() interface{} {
		p.newAllocations.Add(1)
		return &Object{}
	}
	return p
}

// globalObjectPool is the package-level singleton.
var globalObjectPool = NewObjectPool()

// Borrow gets an Object from the pool (allocates if pool is empty).
func (p *ObjectPool) Borrow() *Object {
	p.borrows.Add(1)
	obj := p.pool.Get().(*Object)
	*obj = Object{}
	obj.pooled = true
	return obj
}

// Return puts an Object back into the pool for reuse.
func (p *ObjectPool) Return(obj *Object) {
	if obj == nil || !obj.pooled {
		return
	}
	p.returns.Add(1)
	*obj = Object{}
	p.pool.Put(obj)
}

// Stats returns pool usage metrics.
func (p *ObjectPool) Stats() PoolStats {
	borrows := p.borrows.Load()
	returns := p.returns.Load()
	return PoolStats{
		Borrows:        borrows,
		Returns:        returns,
		NewAllocations: p.newAllocations.Load(),
		InUse:          borrows - returns,
	}
}

// PoolStats contains object pool metrics.
type PoolStats struct {
	Borrows        int64
	Returns        int64
	NewAllocations int64
	InUse          int64
}
