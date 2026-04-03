package protocol

import "sync"

// Subscriber receives WorldSnapshot publications.
type Subscriber interface {
	Receive(WorldSnapshot)
}

// Broadcaster distributes WorldSnapshot values to all registered subscribers.
// Push calls are synchronous; subscriber Receive implementations must be cheap
// or non-blocking to avoid stalling the physics loop.
type Broadcaster struct {
	mu   sync.RWMutex
	subs []Subscriber
}

// Register adds a subscriber. Duplicate registrations result in one
// notification per registration.
func (b *Broadcaster) Register(s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs = append(b.subs, s)
}

// Unregister removes the first occurrence of s. Safe to call if s was never
// registered.
func (b *Broadcaster) Unregister(s Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, sub := range b.subs {
		if sub == s {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			return
		}
	}
}

// Push sends snap to every currently registered subscriber in registration
// order. Safe to call concurrently.
func (b *Broadcaster) Push(snap WorldSnapshot) {
	b.mu.RLock()
	subs := make([]Subscriber, len(b.subs))
	copy(subs, b.subs)
	b.mu.RUnlock()
	for _, s := range subs {
		s.Receive(snap)
	}
}
