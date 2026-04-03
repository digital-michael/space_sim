package persist

import (
	"log"

	"github.com/digital-michael/space_sim/internal/protocol"
)

// Autosave implements protocol.Subscriber and persists WorldSnapshots to disk
// without stalling the physics loop.
//
// It uses a single-slot buffered channel: if the background writer is busy
// when a new snapshot arrives the oldest pending snapshot is silently replaced
// by the newest one, ensuring saves are always current.
type Autosave struct {
	path string
	slot chan protocol.WorldSnapshot
	done chan struct{}
}

// NewAutosave creates an Autosave that writes snapshots to path.
// Call Start to launch the background writer, then register it as a subscriber.
func NewAutosave(path string) *Autosave {
	return &Autosave{
		path: path,
		slot: make(chan protocol.WorldSnapshot, 1),
		done: make(chan struct{}),
	}
}

// Start launches the background goroutine that drains the slot channel and
// calls SaveSnapshot for each snapshot received. It returns immediately.
// Call Stop to shut down the background writer.
func (a *Autosave) Start() {
	go func() {
		for snap := range a.slot {
			if err := SaveSnapshot(a.path, snap); err != nil {
				log.Printf("autosave: SaveSnapshot %s: %v", a.path, err)
			}
		}
		close(a.done)
	}()
}

// Stop closes the slot channel and waits for the background writer to finish
// its current save (if any) before returning.
func (a *Autosave) Stop() {
	close(a.slot)
	<-a.done
}

// Receive implements protocol.Subscriber. It is called by the Broadcaster on
// every frame. If the background writer is still busy the existing pending
// snapshot is replaced so saves stay current without blocking the caller.
func (a *Autosave) Receive(snap protocol.WorldSnapshot) {
	// Drain any pending snapshot then enqueue the new one.
	select {
	case <-a.slot:
	default:
	}
	select {
	case a.slot <- snap:
	default:
		// slot was just filled by another goroutine; safe to drop this frame.
	}
}
