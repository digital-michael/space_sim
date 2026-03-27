package engine

import "sync"

// ObjectPool manages a pool of reusable Object instances to reduce GC pressure.
// Status: infrastructure ready; Strategy 1+5 reduce allocations by ~98%,
// so pooling is not yet wired in.
type ObjectPool struct {
	pool sync.Pool
}

// globalObjectPool is the package-level singleton.
var globalObjectPool = &ObjectPool{
	pool: sync.Pool{
		New: func() interface{} { return &Object{} },
	},
}

// Borrow gets an Object from the pool (allocates if pool is empty).
func (p *ObjectPool) Borrow() *Object {
	return p.pool.Get().(*Object)
}

// Return puts an Object back into the pool for reuse.
func (p *ObjectPool) Return(obj *Object) {
	p.pool.Put(obj)
}

// Stats returns pool metrics (sync.Pool does not expose internal counts).
func (p *ObjectPool) Stats() PoolStats {
	return PoolStats{Available: -1, InUse: -1}
}

// PoolStats contains object pool metrics.
type PoolStats struct {
	Available int // -1 if unknown
	InUse     int // -1 if unknown
}
