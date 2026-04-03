package eventloop

import (
"context"
"sync"

"github.com/digital-michael/space_sim/internal/server/eventqueue"
runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
)

// WorkerPool manages a pool of workers for parallel event processing.
type WorkerPool struct {
mu          sync.RWMutex
workers     []*Worker
workerCount int
manager     *eventqueue.QueueManager
runtime     *runtimepkg.RuntimeEnvironment
running     bool
stopCh      chan struct{}
wg          sync.WaitGroup
}

// NewWorkerPool creates a new worker pool.
// If workerCount is 0, it defaults to the number of runtime CPUs.
func NewWorkerPool(
workerCount int,
manager *eventqueue.QueueManager,
runtime *runtimepkg.RuntimeEnvironment,
) *WorkerPool {
if workerCount <= 0 {
workerCount = 1 // Default to 1 worker for single-threaded start
}

pool := &WorkerPool{
workerCount: workerCount,
manager:     manager,
runtime:     runtime,
stopCh:      make(chan struct{}),
}

// Create workers
pool.workers = make([]*Worker, workerCount)
for i := 0; i < workerCount; i++ {
pool.workers[i] = NewWorker(i, manager, runtime)
}

return pool
}

// Start launches all worker goroutines.
func (p *WorkerPool) Start(ctx context.Context) error {
p.mu.Lock()
defer p.mu.Unlock()

if p.running {
return nil // Already running
}

p.running = true

for _, worker := range p.workers {
p.wg.Add(1)
go func(w *Worker) {
defer p.wg.Done()
w.Run(ctx)
}(worker)
}

return nil
}

// Stop gracefully shuts down all workers and waits for them to finish.
func (p *WorkerPool) Stop() {
p.mu.Lock()
if !p.running {
p.mu.Unlock()
return
}
p.running = false
p.mu.Unlock()

// Stop all workers
for _, worker := range p.workers {
worker.Stop()
}

// Wait for all workers to finish
p.wg.Wait()
}

// ProcessEvents triggers event processing in the worker pool.
// In the current design, workers continuously process events.
// This method can be used to force processing or coordinate timing.
func (p *WorkerPool) ProcessEvents(ctx context.Context) error {
// TODO: Implement cross-GUID work-stealing and coordination
// Current workers are always processing in the background
return nil
}

// GetMetrics returns metrics for all workers.
func (p *WorkerPool) GetMetrics() []WorkerStats {
p.mu.RLock()
defer p.mu.RUnlock()

stats := make([]WorkerStats, len(p.workers))
for i, worker := range p.workers {
stats[i] = worker.GetMetrics()
}
return stats
}

// IsRunning checks if the pool is currently running.
func (p *WorkerPool) IsRunning() bool {
p.mu.RLock()
defer p.mu.RUnlock()
return p.running
}

// WorkerCount returns the number of workers in the pool.
func (p *WorkerPool) WorkerCount() int {
p.mu.RLock()
defer p.mu.RUnlock()
return p.workerCount
}
