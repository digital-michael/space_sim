package eventqueue

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

var ErrNoEvents = errors.New("no events available")

// QueueManager routes events to per-GUID FIFO queues and tracks metrics.
type QueueManager struct {
	mu       sync.RWMutex
	queues   map[uuid.UUID]*GUIDQueue
	queueIDs []uuid.UUID
	rrIndex  int
	capacity int
	metrics  ManagerMetrics
}

// ManagerMetrics tracks queue depth and event throughput.
type ManagerMetrics struct {
	TotalEnqueued uint64
	TotalDequeued uint64
	TotalRejected uint64
	PeakQueueSize int
	ActiveQueues  int
}

// NewQueueManager creates a new queue manager with per-GUID capacity.
func NewQueueManager(capacity int) *QueueManager {
	if capacity <= 0 {
		capacity = 1000
	}
	return &QueueManager{
		queues:   make(map[uuid.UUID]*GUIDQueue),
		queueIDs: make([]uuid.UUID, 0),
		rrIndex:  0,
		capacity: capacity,
		metrics:  ManagerMetrics{},
	}
}

// Enqueue routes an event to its GUID's queue.
// Returns ErrQueueFull if the GUID's queue is at capacity.
func (m *QueueManager) Enqueue(event *Event) error {
	if event == nil {
		return fmt.Errorf("cannot enqueue nil event")
	}

	m.mu.Lock()
	queue, exists := m.queues[event.GUID]
	if !exists {
		queue = NewGUIDQueue(event.GUID, m.capacity)
		m.queues[event.GUID] = queue
		m.queueIDs = append(m.queueIDs, event.GUID)
	}
	m.mu.Unlock()

	if err := queue.Enqueue(event); err != nil {
		m.mu.Lock()
		m.metrics.TotalRejected++
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	m.metrics.TotalEnqueued++
	currentSize := len(m.queues)
	if currentSize > m.metrics.PeakQueueSize {
		m.metrics.PeakQueueSize = currentSize
	}
	m.mu.Unlock()

	return nil
}

// Dequeue retrieves the next event from a specific GUID's queue.
// Returns an error if the queue doesn't exist or is empty.
func (m *QueueManager) Dequeue(guid uuid.UUID) (*Event, error) {
	m.mu.RLock()
	queue, exists := m.queues[guid]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no queue for GUID %s", guid)
	}

	event, err := queue.Dequeue()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.metrics.TotalDequeued++
	m.mu.Unlock()

	return event, nil
}

// DequeueNext retrieves one event from any non-empty GUID queue.
// Returns ErrNoEvents when all queues are empty.
func (m *QueueManager) DequeueNext() (*Event, uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.queueIDs) == 0 {
		return nil, uuid.Nil, ErrNoEvents
	}

	start := m.rrIndex
	for offset := 0; offset < len(m.queueIDs); offset++ {
		idx := (start + offset) % len(m.queueIDs)
		guid := m.queueIDs[idx]
		queue, exists := m.queues[guid]
		if !exists {
			continue
		}

		event, err := queue.Dequeue()
		if err != nil {
			continue
		}

		m.rrIndex = (idx + 1) % len(m.queueIDs)
		m.metrics.TotalDequeued++
		return event, guid, nil
	}

	return nil, uuid.Nil, ErrNoEvents
}

// DequeueAllForGUID retrieves all pending events from a GUID's queue in order.
func (m *QueueManager) DequeueAllForGUID(guid uuid.UUID) ([]*Event, error) {
	m.mu.RLock()
	queue, exists := m.queues[guid]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no queue for GUID %s", guid)
	}

	events := queue.DequeueAll()
	m.mu.Lock()
	m.metrics.TotalDequeued += uint64(len(events))
	m.mu.Unlock()

	return events, nil
}

// PeekGUID returns the next event for a GUID without removing it.
func (m *QueueManager) PeekGUID(guid uuid.UUID) (*Event, error) {
	m.mu.RLock()
	queue, exists := m.queues[guid]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no queue for GUID %s", guid)
	}

	return queue.Peek()
}

// LenGUID returns the number of pending events for a GUID.
func (m *QueueManager) LenGUID(guid uuid.UUID) (int, error) {
	m.mu.RLock()
	queue, exists := m.queues[guid]
	m.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("no queue for GUID %s", guid)
	}

	return queue.Len(), nil
}

// FullGUID checks if a GUID's queue is at capacity.
func (m *QueueManager) FullGUID(guid uuid.UUID) (bool, error) {
	m.mu.RLock()
	queue, exists := m.queues[guid]
	m.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("no queue for GUID %s", guid)
	}

	return queue.Full(), nil
}

// GUIDExists checks if a queue exists for the given GUID.
func (m *QueueManager) GUIDExists(guid uuid.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.queues[guid]
	return exists
}

// Metrics returns a copy of current manager metrics.
func (m *QueueManager) Metrics() ManagerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return ManagerMetrics{
		TotalEnqueued: m.metrics.TotalEnqueued,
		TotalDequeued: m.metrics.TotalDequeued,
		TotalRejected: m.metrics.TotalRejected,
		PeakQueueSize: m.metrics.PeakQueueSize,
		ActiveQueues:  len(m.queues),
	}
}
