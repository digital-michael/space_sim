package eventloop

import (
"context"
"testing"

"github.com/digital-michael/space_sim/internal/server/eventqueue"
"github.com/digital-michael/space_sim/internal/server/pool/group"
runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
)

func TestWorkerPoolCreation(t *testing.T) {
definitions := group.NewPool()
manager := eventqueue.NewQueueManager(10)
runtime := runtimepkg.NewRuntimeEnvironment(definitions)

pool := NewWorkerPool(2, manager, runtime)

if pool.WorkerCount() != 2 {
t.Fatalf("expected 2 workers, got %d", pool.WorkerCount())
}
if pool.IsRunning() {
t.Fatal("expected pool to not be running")
}
}

func TestWorkerPoolDefaults(t *testing.T) {
definitions := group.NewPool()
manager := eventqueue.NewQueueManager(10)
runtime := runtimepkg.NewRuntimeEnvironment(definitions)

pool := NewWorkerPool(0, manager, runtime)

if pool.WorkerCount() != 1 {
t.Fatalf("expected default 1 worker, got %d", pool.WorkerCount())
}
}

func TestWorkerPoolStartStop(t *testing.T) {
definitions := group.NewPool()
manager := eventqueue.NewQueueManager(10)
runtime := runtimepkg.NewRuntimeEnvironment(definitions)

pool := NewWorkerPool(1, manager, runtime)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := pool.Start(ctx); err != nil {
t.Fatalf("start failed: %v", err)
}

if !pool.IsRunning() {
t.Fatal("expected pool to be running")
}

pool.Stop()

if pool.IsRunning() {
t.Fatal("expected pool to be stopped")
}
}

func TestWorkerPoolDoubleStart(t *testing.T) {
definitions := group.NewPool()
manager := eventqueue.NewQueueManager(10)
runtime := runtimepkg.NewRuntimeEnvironment(definitions)

pool := NewWorkerPool(1, manager, runtime)

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := pool.Start(ctx); err != nil {
t.Fatalf("first start failed: %v", err)
}

if err := pool.Start(ctx); err != nil {
t.Fatalf("second start failed: %v", err)
}

pool.Stop()
}

func TestWorkerPoolGetMetrics(t *testing.T) {
definitions := group.NewPool()
manager := eventqueue.NewQueueManager(10)
runtime := runtimepkg.NewRuntimeEnvironment(definitions)

pool := NewWorkerPool(2, manager, runtime)

metrics := pool.GetMetrics()

if len(metrics) != 2 {
t.Fatalf("expected 2 worker metrics, got %d", len(metrics))
}
if metrics[0].WorkerID != 0 {
t.Fatalf("expected worker 0, got %d", metrics[0].WorkerID)
}
if metrics[1].WorkerID != 1 {
t.Fatalf("expected worker 1, got %d", metrics[1].WorkerID)
}
}
