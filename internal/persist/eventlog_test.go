package persist_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/digital-michael/space_sim/internal/persist"
	"github.com/digital-michael/space_sim/internal/server/eventqueue"
	"github.com/google/uuid"
)

func makeEvent(t string) *eventqueue.Event {
	return eventqueue.NewEvent(
		uuid.New(),
		eventqueue.EventType(t),
		map[string]interface{}{"key": t},
		eventqueue.TransactionTypeNone,
	)
}

func TestEventLogAppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	el, err := persist.OpenEventLog(path)
	if err != nil {
		t.Fatalf("OpenEventLog: %v", err)
	}

	events := []*eventqueue.Event{
		makeEvent("create"),
		makeEvent("update"),
		makeEvent("delete"),
	}
	for _, e := range events {
		if err := el.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if err := el.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var replayed []*eventqueue.Event
	if err := persist.Replay(path, func(e *eventqueue.Event) {
		replayed = append(replayed, e)
	}); err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if got, want := len(replayed), len(events); got != want {
		t.Fatalf("replayed count: got %d, want %d", got, want)
	}
	for i, orig := range events {
		got := replayed[i]
		if got.ID != orig.ID {
			t.Errorf("event[%d].ID: got %v, want %v", i, got.ID, orig.ID)
		}
		if got.Type != orig.Type {
			t.Errorf("event[%d].Type: got %v, want %v", i, got.Type, orig.Type)
		}
	}
}

func TestEventLogAppend_NilEvent(t *testing.T) {
	dir := t.TempDir()
	el, err := persist.OpenEventLog(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer el.Close()

	if err := el.Append(nil); err == nil {
		t.Error("expected error appending nil event, got nil")
	}
}

func TestEventLogReplay_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	el, err := persist.OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := el.Append(makeEvent("create")); err != nil {
		t.Fatal(err)
	}
	if err := el.Close(); err != nil {
		t.Fatal(err)
	}

	// Append a malformed line directly.
	f, _ := openAppend(path)
	f.WriteString("{bad json}\n")
	f.Close()

	var count int
	err = persist.Replay(path, func(*eventqueue.Event) { count++ })
	if err == nil {
		t.Error("expected error reporting skipped lines, got nil")
	}
	// The one valid event should still have been replayed.
	if count != 1 {
		t.Errorf("expected 1 valid event, got %d", count)
	}
}

func TestEventLogConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	el, err := persist.OpenEventLog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer el.Close()

	const goroutines = 10
	const perGoroutine = 50

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				if err := el.Append(makeEvent("update")); err != nil {
					t.Errorf("concurrent Append: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	if err := el.Close(); err != nil {
		t.Fatal(err)
	}

	var total int
	if err := persist.Replay(path, func(*eventqueue.Event) { total++ }); err != nil {
		t.Fatalf("Replay after concurrent writes: %v", err)
	}
	if total != goroutines*perGoroutine {
		t.Errorf("expected %d events, got %d", goroutines*perGoroutine, total)
	}
}

// openAppend is a test helper that opens a file in append mode.
func openAppend(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
}
