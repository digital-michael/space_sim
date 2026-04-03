package eventloop

import (
	"testing"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/digital-michael/space_sim/internal/server/pool/group"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

func TestWorkerCreation(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	if worker.id != 0 {
		t.Fatalf("expected worker ID 0, got %d", worker.id)
	}
	if worker.manager != manager {
		t.Fatal("expected manager to be set")
	}
	if worker.runtime != runtime {
		t.Fatal("expected runtime to be set")
	}
	if worker.metrics == nil {
		t.Fatal("expected metrics to be initialized")
	}
}

func TestWorkerMetricsRetrieval(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(1, manager, runtime)

	stats := worker.GetMetrics()
	if stats.WorkerID != 1 {
		t.Fatalf("expected worker ID 1, got %d", stats.WorkerID)
	}
	if stats.EventsProcessed != 0 {
		t.Fatalf("expected 0 events, got %d", stats.EventsProcessed)
	}
}

func TestWorkerProcessEvent(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	guid := uuid.New()
	event := eventqueue.NewEvent(guid, eventqueue.EventTypeCustom, map[string]interface{}{"tag": "ok"}, eventqueue.TransactionTypeNone)

	err := worker.processEvent(event, guid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stats := worker.GetMetrics()
	if stats.EventsProcessed != 1 {
		t.Fatalf("expected 1 event processed, got %d", stats.EventsProcessed)
	}
}

func TestWorkerProcessEventNil(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	guid := uuid.New()

	err := worker.processEvent(nil, guid)
	if err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestWorkerStop(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	worker.Stop()

	select {
	case <-worker.stopCh:
	default:
		t.Fatal("expected stop channel to be closed")
	}
}

func TestWorkerProcessOneEventNoEvents(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	processed := worker.processOneEvent()
	if processed {
		t.Fatal("expected no event to be processed")
	}

	stats := worker.GetMetrics()
	if stats.EventsProcessed != 0 {
		t.Fatalf("expected 0 processed events, got %d", stats.EventsProcessed)
	}
}

func TestWorkerProcessOneEventWithEvent(t *testing.T) {
	definitions := group.NewPool()
	manager := eventqueue.NewQueueManager(10)
	runtime := runtimepkg.NewRuntimeEnvironment(definitions)

	worker := NewWorker(0, manager, runtime)

	guid := uuid.New()
	event := eventqueue.NewEvent(guid, eventqueue.EventTypeCustom, map[string]interface{}{"tag": "queue"}, eventqueue.TransactionTypeNone)
	if err := manager.Enqueue(event); err != nil {
		t.Fatalf("failed to enqueue event: %v", err)
	}

	processed := worker.processOneEvent()
	if !processed {
		t.Fatal("expected one event to be processed")
	}

	stats := worker.GetMetrics()
	if stats.EventsProcessed != 1 {
		t.Fatalf("expected 1 processed event, got %d", stats.EventsProcessed)
	}
}
