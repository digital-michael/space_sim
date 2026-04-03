package eventqueue

import (
	"testing"

	"github.com/google/uuid"
)

func TestQueueManagerBasic(t *testing.T) {
	mgr := NewQueueManager(10)

	guid1 := uuid.New()
	event := NewEvent(guid1, EventTypeCreate, nil, TransactionTypeNone)

	if err := mgr.Enqueue(event); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	retrieved, err := mgr.Dequeue(guid1)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if retrieved.ID != event.ID {
		t.Fatal("retrieved event doesn't match")
	}

	metrics := mgr.Metrics()
	if metrics.TotalEnqueued != 1 || metrics.TotalDequeued != 1 {
		t.Fatalf("metrics mismatch: enqueued=%d dequeued=%d", metrics.TotalEnqueued, metrics.TotalDequeued)
	}
}

func TestQueueManagerPerGUIDIndependence(t *testing.T) {
	mgr := NewQueueManager(5)

	guid1 := uuid.New()
	guid2 := uuid.New()

	// Queue 3 events to guid1
	for i := 0; i < 3; i++ {
		e := NewEvent(guid1, EventTypeUpdate, map[string]interface{}{"index": i}, TransactionTypeNone)
		if err := mgr.Enqueue(e); err != nil {
			t.Fatalf("enqueue to guid1 failed: %v", err)
		}
	}

	// Queue 2 events to guid2
	for i := 0; i < 2; i++ {
		e := NewEvent(guid2, EventTypeDelete, map[string]interface{}{"index": i}, TransactionTypeNone)
		if err := mgr.Enqueue(e); err != nil {
			t.Fatalf("enqueue to guid2 failed: %v", err)
		}
	}

	len1, _ := mgr.LenGUID(guid1)
	len2, _ := mgr.LenGUID(guid2)
	if len1 != 3 || len2 != 2 {
		t.Fatalf("expected len1=3, len2=2; got len1=%d, len2=%d", len1, len2)
	}

	// Dequeue from guid1 doesn't affect guid2
	mgr.Dequeue(guid1)
	len1, _ = mgr.LenGUID(guid1)
	len2, _ = mgr.LenGUID(guid2)
	if len1 != 2 || len2 != 2 {
		t.Fatalf("expected len1=2, len2=2 after dequeue; got len1=%d, len2=%d", len1, len2)
	}
}

func TestQueueManagerCapacityPerGUID(t *testing.T) {
	mgr := NewQueueManager(3)

	guid := uuid.New()

	// Fill to capacity
	for i := 0; i < 3; i++ {
		e := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
		if err := mgr.Enqueue(e); err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}

	// Attempt to exceed capacity
	e := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	if err := mgr.Enqueue(e); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	metrics := mgr.Metrics()
	if metrics.TotalRejected != 1 {
		t.Fatalf("expected TotalRejected=1, got %d", metrics.TotalRejected)
	}
}

func TestQueueManagerDequeueAllForGUID(t *testing.T) {
	mgr := NewQueueManager(10)

	guid := uuid.New()

	// Enqueue 5 events
	for i := 0; i < 5; i++ {
		e := NewEvent(guid, EventTypeCreate, map[string]interface{}{"index": i}, TransactionTypeNone)
		mgr.Enqueue(e)
	}

	// DequeueAll
	all, err := mgr.DequeueAllForGUID(guid)
	if err != nil {
		t.Fatalf("dequeue all failed: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 events, got %d", len(all))
	}

	len, _ := mgr.LenGUID(guid)
	if len != 0 {
		t.Fatalf("expected queue to be empty after DequeueAll")
	}

	// Verify order
	for i := 0; i < 5; i++ {
		if all[i].Payload["index"] != i {
			t.Fatalf("expected index %d at position %d, got %v", i, i, all[i].Payload["index"])
		}
	}

	metrics := mgr.Metrics()
	if metrics.TotalDequeued != 5 {
		t.Fatalf("expected TotalDequeued=5, got %d", metrics.TotalDequeued)
	}
}

func TestQueueManagerPeekGUID(t *testing.T) {
	mgr := NewQueueManager(10)

	guid := uuid.New()
	e := NewEvent(guid, EventTypeCreate, map[string]interface{}{"data": "test"}, TransactionTypeNone)
	mgr.Enqueue(e)

	peeked, err := mgr.PeekGUID(guid)
	if err != nil {
		t.Fatalf("peek failed: %v", err)
	}
	if peeked.Payload["data"] != "test" {
		t.Fatal("peek returned wrong event")
	}

	// Peek doesn't remove
	len, _ := mgr.LenGUID(guid)
	if len != 1 {
		t.Fatal("peek should not remove event")
	}
}

func TestQueueManagerFullGUID(t *testing.T) {
	mgr := NewQueueManager(2)

	guid := uuid.New()

	full, _ := mgr.FullGUID(guid)
	if full {
		t.Fatal("expected new queue to not be full")
	}

	e1 := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	e2 := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	mgr.Enqueue(e1)
	mgr.Enqueue(e2)

	full, _ = mgr.FullGUID(guid)
	if !full {
		t.Fatal("expected queue to be full with 2/2 items")
	}
}

func TestQueueManagerGUIDExists(t *testing.T) {
	mgr := NewQueueManager(10)

	guid := uuid.New()

	if mgr.GUIDExists(guid) {
		t.Fatal("expected GUID to not exist initially")
	}

	e := NewEvent(guid, EventTypeCreate, nil, TransactionTypeNone)
	mgr.Enqueue(e)

	if !mgr.GUIDExists(guid) {
		t.Fatal("expected GUID to exist after enqueue")
	}
}

func TestQueueManagerMetrics(t *testing.T) {
	mgr := NewQueueManager(10)

	guid1 := uuid.New()
	guid2 := uuid.New()

	// Enqueue to 2 GUIDs
	for i := 0; i < 3; i++ {
		e := NewEvent(guid1, EventTypeCreate, nil, TransactionTypeNone)
		mgr.Enqueue(e)
	}
	for i := 0; i < 2; i++ {
		e := NewEvent(guid2, EventTypeCreate, nil, TransactionTypeNone)
		mgr.Enqueue(e)
	}

	// Dequeue some
	mgr.Dequeue(guid1)
	mgr.Dequeue(guid2)

	metrics := mgr.Metrics()
	if metrics.TotalEnqueued != 5 {
		t.Fatalf("expected TotalEnqueued=5, got %d", metrics.TotalEnqueued)
	}
	if metrics.TotalDequeued != 2 {
		t.Fatalf("expected TotalDequeued=2, got %d", metrics.TotalDequeued)
	}
	if metrics.ActiveQueues != 2 {
		t.Fatalf("expected ActiveQueues=2, got %d", metrics.ActiveQueues)
	}
	if metrics.PeakQueueSize != 2 {
		t.Fatalf("expected PeakQueueSize=2, got %d", metrics.PeakQueueSize)
	}
}

func TestQueueManagerErrorCases(t *testing.T) {
	mgr := NewQueueManager(10)
	guid := uuid.New()

	// Dequeue from non-existent queue
	if _, err := mgr.Dequeue(guid); err == nil {
		t.Fatal("expected error for dequeue on non-existent queue")
	}

	// DequeueAllForGUID from non-existent queue
	if _, err := mgr.DequeueAllForGUID(guid); err == nil {
		t.Fatal("expected error for dequeue all on non-existent queue")
	}

	// PeekGUID from non-existent queue
	if _, err := mgr.PeekGUID(guid); err == nil {
		t.Fatal("expected error for peek on non-existent queue")
	}

	// LenGUID from non-existent queue
	if _, err := mgr.LenGUID(guid); err == nil {
		t.Fatal("expected error for len on non-existent queue")
	}

	// FullGUID from non-existent queue
	if _, err := mgr.FullGUID(guid); err == nil {
		t.Fatal("expected error for full on non-existent queue")
	}

	// Enqueue nil event
	if err := mgr.Enqueue(nil); err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestQueueManagerDequeueNext(t *testing.T) {
	mgr := NewQueueManager(10)

	guid1 := uuid.New()
	guid2 := uuid.New()

	if err := mgr.Enqueue(NewEvent(guid1, EventTypeCreate, map[string]interface{}{"n": 1}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue guid1 failed: %v", err)
	}
	if err := mgr.Enqueue(NewEvent(guid2, EventTypeUpdate, map[string]interface{}{"n": 2}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue guid2 failed: %v", err)
	}

	eventA, guidA, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("dequeue next (1) failed: %v", err)
	}
	if eventA == nil {
		t.Fatal("expected first event, got nil")
	}
	if guidA == uuid.Nil {
		t.Fatal("expected non-nil guid for first dequeue")
	}

	eventB, guidB, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("dequeue next (2) failed: %v", err)
	}
	if eventB == nil {
		t.Fatal("expected second event, got nil")
	}
	if guidB == uuid.Nil {
		t.Fatal("expected non-nil guid for second dequeue")
	}

	if guidA == guidB {
		lenA, _ := mgr.LenGUID(guidA)
		if lenA != 0 {
			t.Fatalf("expected guid %s queue to be empty after two dequeues, got len=%d", guidA, lenA)
		}
	}

	metrics := mgr.Metrics()
	if metrics.TotalDequeued < 2 {
		t.Fatalf("expected TotalDequeued >= 2, got %d", metrics.TotalDequeued)
	}
}

func TestQueueManagerDequeueNextNoEvents(t *testing.T) {
	mgr := NewQueueManager(10)

	event, guid, err := mgr.DequeueNext()
	if err != ErrNoEvents {
		t.Fatalf("expected ErrNoEvents, got %v", err)
	}
	if event != nil {
		t.Fatal("expected nil event when no events available")
	}
	if guid != uuid.Nil {
		t.Fatalf("expected nil guid when no events available, got %s", guid)
	}
}

func TestQueueManagerDequeueNextRoundRobinFairness(t *testing.T) {
	mgr := NewQueueManager(10)

	guid1 := uuid.New()
	guid2 := uuid.New()

	if err := mgr.Enqueue(NewEvent(guid1, EventTypeCreate, map[string]interface{}{"id": "g1-1"}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue g1-1 failed: %v", err)
	}
	if err := mgr.Enqueue(NewEvent(guid2, EventTypeCreate, map[string]interface{}{"id": "g2-1"}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue g2-1 failed: %v", err)
	}
	if err := mgr.Enqueue(NewEvent(guid1, EventTypeCreate, map[string]interface{}{"id": "g1-2"}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue g1-2 failed: %v", err)
	}
	if err := mgr.Enqueue(NewEvent(guid2, EventTypeCreate, map[string]interface{}{"id": "g2-2"}, TransactionTypeNone)); err != nil {
		t.Fatalf("enqueue g2-2 failed: %v", err)
	}

	_, firstGUID, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("first dequeue next failed: %v", err)
	}
	_, secondGUID, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("second dequeue next failed: %v", err)
	}
	_, thirdGUID, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("third dequeue next failed: %v", err)
	}
	_, fourthGUID, err := mgr.DequeueNext()
	if err != nil {
		t.Fatalf("fourth dequeue next failed: %v", err)
	}

	if firstGUID == secondGUID {
		t.Fatalf("expected first two dequeues to come from different GUIDs, got %s then %s", firstGUID, secondGUID)
	}
	if firstGUID != thirdGUID {
		t.Fatalf("expected third dequeue to cycle back to first GUID, got first=%s third=%s", firstGUID, thirdGUID)
	}
	if secondGUID != fourthGUID {
		t.Fatalf("expected fourth dequeue to cycle back to second GUID, got second=%s fourth=%s", secondGUID, fourthGUID)
	}
}
