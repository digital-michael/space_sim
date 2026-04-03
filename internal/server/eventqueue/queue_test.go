package eventqueue

import (
	"testing"

	"github.com/google/uuid"
)

func TestGUIDQueueBasicOps(t *testing.T) {
	guid := uuid.New()
	queue := NewGUIDQueue(guid, 10)

	if queue.Len() != 0 {
		t.Fatal("expected empty queue")
	}

	event := NewEvent(guid, EventTypeCreate, map[string]interface{}{"id": 1}, TransactionTypeNone)
	if err := queue.Enqueue(event); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if queue.Len() != 1 {
		t.Fatalf("expected length 1, got %d", queue.Len())
	}

	peek, err := queue.Peek()
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if peek.ID != event.ID {
		t.Fatal("expected peek to return same event")
	}
	if queue.Len() != 1 {
		t.Fatal("peek should not remove event")
	}

	dequeued, err := queue.Dequeue()
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if dequeued.ID != event.ID {
		t.Fatal("expected dequeued event to match")
	}
	if queue.Len() != 0 {
		t.Fatal("expected queue to be empty after dequeue")
	}
}

func TestGUIDQueueFIFOOrder(t *testing.T) {
	guid := uuid.New()
	queue := NewGUIDQueue(guid, 10)

	events := make([]*Event, 5)
	for i := 0; i < 5; i++ {
		e := NewEvent(guid, EventTypeUpdate, map[string]interface{}{"index": i}, TransactionTypeNone)
		events[i] = e
		if err := queue.Enqueue(e); err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}

	for i := 0; i < 5; i++ {
		dequeued, _ := queue.Dequeue()
		if dequeued.Payload["index"] != i {
			t.Fatalf("expected index %d, got %v", i, dequeued.Payload["index"])
		}
	}
}

func TestGUIDQueueFull(t *testing.T) {
	guid := uuid.New()
	queue := NewGUIDQueue(guid, 2)

	event1 := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	event2 := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	event3 := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)

	if err := queue.Enqueue(event1); err != nil {
		t.Fatalf("enqueue 1 failed: %v", err)
	}
	if err := queue.Enqueue(event2); err != nil {
		t.Fatalf("enqueue 2 failed: %v", err)
	}
	if !queue.Full() {
		t.Fatal("expected queue to be full with 2/2 items")
	}

	if err := queue.Enqueue(event3); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}

func TestGUIDQueueGUIDValidation(t *testing.T) {
	guid1 := uuid.New()
	guid2 := uuid.New()
	queue := NewGUIDQueue(guid1, 10)

	event := NewEvent(guid2, EventTypeCreate, nil, TransactionTypeNone)
	if err := queue.Enqueue(event); err == nil {
		t.Fatal("expected error when enqueueing event with mismatched GUID")
	}
}

func TestGUIDQueueDequeueAll(t *testing.T) {
	guid := uuid.New()
	queue := NewGUIDQueue(guid, 10)

	for i := 0; i < 5; i++ {
		event := NewEvent(guid, EventTypeCreate, map[string]interface{}{"index": i}, TransactionTypeNone)
		queue.Enqueue(event)
	}

	all := queue.DequeueAll()
	if len(all) != 5 {
		t.Fatalf("expected 5 events, got %d", len(all))
	}
	if queue.Len() != 0 {
		t.Fatal("expected queue to be empty after DequeueAll")
	}

	for i := 0; i < 5; i++ {
		if all[i].Payload["index"] != i {
			t.Fatalf("expected index %d, got %v", i, all[i].Payload["index"])
		}
	}
}

func TestGUIDQueueEmptyOperations(t *testing.T) {
	guid := uuid.New()
	queue := NewGUIDQueue(guid, 10)

	if _, err := queue.Peek(); err == nil {
		t.Fatal("expected peek on empty queue to fail")
	}

	if _, err := queue.Dequeue(); err == nil {
		t.Fatal("expected dequeue on empty queue to fail")
	}

	all := queue.DequeueAll()
	if len(all) != 0 {
		t.Fatalf("expected empty slice from DequeueAll, got %d", len(all))
	}
}
