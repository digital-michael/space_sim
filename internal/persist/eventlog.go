package persist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/digital-michael/space_sim/internal/server/eventqueue"
)

// EventLog is an append-only, JSON-lines event journal. Each Append call
// writes one JSON object per line to the underlying file. Replay reads the
// file line-by-line and calls the provided handler for each valid event.
//
// EventLog is safe for concurrent Append calls.
type EventLog struct {
	mu   sync.Mutex
	path string
	f    *os.File
	w    *bufio.Writer
}

// OpenEventLog opens (or creates) a JSON-lines event log at path.
// Existing content is preserved; new events are appended.
func OpenEventLog(path string) (*EventLog, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("persist: open event log %s: %w", path, err)
	}
	return &EventLog{
		path: path,
		f:    f,
		w:    bufio.NewWriter(f),
	}, nil
}

// Append serialises e as a JSON object and writes it as a single line.
// The line is flushed and synced before returning so callers know the
// entry is durable.
func (el *EventLog) Append(e *eventqueue.Event) error {
	if e == nil {
		return fmt.Errorf("persist: cannot append nil event")
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("persist: marshal event %s: %w", e.ID, err)
	}

	el.mu.Lock()
	defer el.mu.Unlock()

	if _, err := el.w.Write(data); err != nil {
		return fmt.Errorf("persist: write event log: %w", err)
	}
	if err := el.w.WriteByte('\n'); err != nil {
		return fmt.Errorf("persist: write event log newline: %w", err)
	}
	if err := el.w.Flush(); err != nil {
		return fmt.Errorf("persist: flush event log: %w", err)
	}
	if err := el.f.Sync(); err != nil {
		return fmt.Errorf("persist: sync event log: %w", err)
	}
	return nil
}

// Close closes the underlying file.
func (el *EventLog) Close() error {
	el.mu.Lock()
	defer el.mu.Unlock()
	if err := el.w.Flush(); err != nil {
		return fmt.Errorf("persist: flush on close: %w", err)
	}
	return el.f.Close()
}

// Replay reads the event log at path and calls fn for each decoded event.
// Lines that fail to decode are skipped and counted; if any were skipped the
// returned error is non-nil and includes a count.
//
// Replay is a standalone function (not a method) so callers can replay a
// log file without holding an open EventLog.
func Replay(path string, fn func(*eventqueue.Event)) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("persist: open event log for replay %s: %w", path, err)
	}
	defer f.Close()

	var skipped int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e eventqueue.Event
		if err := json.Unmarshal(line, &e); err != nil {
			skipped++
			continue
		}
		fn(&e)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("persist: scan event log: %w", err)
	}
	if skipped > 0 {
		return fmt.Errorf("persist: replay skipped %d malformed line(s)", skipped)
	}
	return nil
}
