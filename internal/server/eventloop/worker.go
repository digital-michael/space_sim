package eventloop

import (
	"context"
	"fmt"
	"time"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	runtimepkg "github.com/digital-michael/space_sim/internal/server/runtime"
	"github.com/google/uuid"
)

// Worker processes events from the event queue manager.
type Worker struct {
	id      int
	manager *eventqueue.QueueManager
	runtime *runtimepkg.RuntimeEnvironment
	metrics *WorkerMetrics
	stopCh  chan struct{}
}

// NewWorker creates a new worker instance.
func NewWorker(
	id int,
	manager *eventqueue.QueueManager,
	runtime *runtimepkg.RuntimeEnvironment,
) *Worker {
	return &Worker{
		id:      id,
		manager: manager,
		runtime: runtime,
		metrics: NewWorkerMetrics(id),
		stopCh:  make(chan struct{}),
	}
}

// Run is the worker's main loop - processes events from the queue manager.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			// Try to pull and process one event
			if !w.processOneEvent() {
				// No events available, continue waiting
			}
		}
	}
}

// processOneEvent pulls one event from any queue and processes it.
// Returns true if an event was processed, false if no events available.
func (w *Worker) processOneEvent() bool {
	event, guid, err := w.manager.DequeueNext()
	if err != nil {
		if err == eventqueue.ErrNoEvents {
			return false
		}
		w.metrics.RecordError(err)
		return false
	}

	if err := w.processEvent(event, guid); err != nil {
		w.metrics.RecordError(err)
	}

	return true
}

// processEvent executes an event on the runtime environment.
func (w *Worker) processEvent(event *eventqueue.Event, guid uuid.UUID) error {
	start := time.Now()
	if event == nil {
		return fmt.Errorf("nil event")
	}
	if event.GUID != guid {
		return fmt.Errorf("event GUID %s does not match target GUID %s", event.GUID, guid)
	}

	transactionContext := eventqueue.NewTransactionContext(event.TransactionType)
	transactionContext.AddEvent(event)
	if err := transactionContext.Execute(w.runtime); err != nil {
		return err
	}

	duration := time.Since(start)
	w.metrics.RecordEvent(duration)

	return nil
}

// GetMetrics returns the worker's metrics snapshot.
func (w *Worker) GetMetrics() WorkerStats {
	return w.metrics.GetStats()
}

// Stop gracefully stops the worker.
func (w *Worker) Stop() {
	close(w.stopCh)
}
