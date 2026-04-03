package protocol_test

import (
	"sync/atomic"
	"testing"

	"github.com/digital-michael/space_sim/internal/protocol"
)

type countSubscriber struct{ n atomic.Int64 }

func (c *countSubscriber) Receive(_ protocol.WorldSnapshot) { c.n.Add(1) }

func TestBroadcasterPushReachesAllSubscribers(t *testing.T) {
	var b protocol.Broadcaster
	a, bb := &countSubscriber{}, &countSubscriber{}
	b.Register(a)
	b.Register(bb)
	b.Push(protocol.WorldSnapshot{})
	if a.n.Load() != 1 || bb.n.Load() != 1 {
		t.Fatal("not all subscribers received the snapshot")
	}
}

func TestBroadcasterUnregisterStopsDelivery(t *testing.T) {
	var b protocol.Broadcaster
	c := &countSubscriber{}
	b.Register(c)
	b.Unregister(c)
	b.Push(protocol.WorldSnapshot{})
	if c.n.Load() != 0 {
		t.Fatal("unregistered subscriber should not receive")
	}
}

func TestBroadcasterConcurrentPush(t *testing.T) {
	var b protocol.Broadcaster
	c := &countSubscriber{}
	b.Register(c)
	const goroutines = 20
	done := make(chan struct{}, goroutines)
	for range goroutines {
		go func() {
			b.Push(protocol.WorldSnapshot{})
			done <- struct{}{}
		}()
	}
	for range goroutines {
		<-done
	}
	if c.n.Load() != goroutines {
		t.Fatalf("expected %d receives, got %d", goroutines, c.n.Load())
	}
}

func TestBroadcasterUnregisterUnknownIsNoOp(t *testing.T) {
	var b protocol.Broadcaster
	c := &countSubscriber{}
	b.Unregister(c)
	b.Push(protocol.WorldSnapshot{})
	if c.n.Load() != 0 {
		t.Fatal("unexpected delivery")
	}
}
