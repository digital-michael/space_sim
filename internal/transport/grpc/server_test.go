package grpcserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/digital-michael/space_sim/api/gen/spacesim/v1"
	"github.com/digital-michael/space_sim/api/gen/spacesim/v1/spacesimv1connect"
	"github.com/digital-michael/space_sim/internal/protocol"
	"github.com/digital-michael/space_sim/internal/sim/engine"
	"github.com/digital-michael/space_sim/internal/sim/world"
)

// nilWorld is a worldFn that always returns nil, used to test the
// CodeUnimplemented guard path without a real simulation session.
func nilWorld() *world.World { return nil }

// real routing path without starting a net.Listener.
func newTestMux(sim spacesimv1connect.SimulationServiceHandler, world spacesimv1connect.WorldServiceHandler) http.Handler {
	mux := http.NewServeMux()
	simPath, simH := spacesimv1connect.NewSimulationServiceHandler(sim)
	mux.Handle(simPath, simH)
	worldPath, worldH := spacesimv1connect.NewWorldServiceHandler(world)
	mux.Handle(worldPath, worldH)
	return mux
}

// ─── Transport routing ─────────────────────────────────────────────────────

// TestSetSpeed_NilWorld_Unimplemented confirms the full ConnectRPC transport
// round-trip works: request serialised, routed to SimulationHandler, nil-world
// guard fires, error deserialised back to CodeUnimplemented.
func TestSetSpeed_NilWorld_Unimplemented(t *testing.T) {
	srv := httptest.NewServer(newTestMux(NewSimulationHandler(nilWorld), NewWorldHandler()))
	t.Cleanup(srv.Close)

	client := spacesimv1connect.NewSimulationServiceClient(srv.Client(), srv.URL)
	_, err := client.SetSpeed(context.Background(), connect.NewRequest(&v1.SetSpeedRequest{
		SecondsPerSecond: 2.0,
	}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if cerr.Code() != connect.CodeUnimplemented {
		t.Errorf("expected CodeUnimplemented, got %v", cerr.Code())
	}
}

// TestGetSpeed_NilWorld_Unimplemented exercises a different RPC path to confirm
// all handler methods correctly guard against a nil world.
func TestGetSpeed_NilWorld_Unimplemented(t *testing.T) {
	srv := httptest.NewServer(newTestMux(NewSimulationHandler(nilWorld), NewWorldHandler()))
	t.Cleanup(srv.Close)

	client := spacesimv1connect.NewSimulationServiceClient(srv.Client(), srv.URL)
	_, err := client.GetSpeed(context.Background(), connect.NewRequest(&v1.GetSpeedRequest{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var cerr *connect.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if cerr.Code() != connect.CodeUnimplemented {
		t.Errorf("expected CodeUnimplemented, got %v", cerr.Code())
	}
}

// ─── Connection limiter ────────────────────────────────────────────────────

// TestConnLimit_ExceedLimit_Returns429 verifies that a second concurrent
// request is rejected with HTTP 429 when MaxConns == 1.
func TestConnLimit_ExceedLimit_Returns429(t *testing.T) {
	release := make(chan struct{})
	inFlight := make(chan struct{}, 1)

	// A blocking inner handler; signals once it is processing.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case inFlight <- struct{}{}:
		default:
		}
		<-release
		w.WriteHeader(http.StatusOK)
	})

	h := newConnLimitHandler(1, inner)

	// First request: blocks until release is closed.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	w1 := httptest.NewRecorder()
	go h.ServeHTTP(w1, req1)

	// Wait until the first request has incremented the counter.
	select {
	case <-inFlight:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: first request never reached inner handler")
	}

	// Second request should be rejected immediately.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w2.Code)
	}

	// Release the first request and let the goroutine finish.
	close(release)
}

// TestConnLimit_WithinLimit_PassesThrough confirms that a request within the
// configured limit reaches the inner handler normally.
func TestConnLimit_WithinLimit_PassesThrough(t *testing.T) {
	const sentinel = "reached"
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", sentinel)
		w.WriteHeader(http.StatusOK)
	})

	h := newConnLimitHandler(5, inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Test") != sentinel {
		t.Errorf("inner handler not reached")
	}
}

// TestConnLimit_Zero_NoLimiting confirms that max == 0 disables limiting:
// requests pass through to the inner handler without rejection.
func TestConnLimit_Zero_NoLimiting(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := newConnLimitHandler(0, inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — inner handler was not reached or limiting was applied", w.Code)
	}
}

// ─── WorldHandler fan-out ────────────────────────────────────────────────────

// TestWorldHandler_Receive_DeliversToStream verifies that a snapshot pushed via
// Receive is delivered to a registered stream channel.
func TestWorldHandler_Receive_DeliversToStream(t *testing.T) {
	h := NewWorldHandler()

	ch := make(chan protocol.WorldSnapshot, 1)
	h.addStream(ch)
	defer h.removeStream(ch)

	snap := protocol.WorldSnapshot{
		Speed: 3.5,
		State: &engine.SimulationState{Time: 99999.0},
	}
	h.Receive(snap)

	select {
	case got := <-ch:
		if got.Speed != snap.Speed {
			t.Errorf("Speed: want %v, got %v", snap.Speed, got.Speed)
		}
		if got.State.Time != snap.State.Time {
			t.Errorf("State.Time: want %v, got %v", snap.State.Time, got.State.Time)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: snapshot not delivered")
	}
}

// TestWorldHandler_Receive_DropsOnSlowConsumer confirms Receive is non-blocking:
// a full stream channel does not stall the caller.
func TestWorldHandler_Receive_DropsOnSlowConsumer(t *testing.T) {
	h := NewWorldHandler()

	// Buffer of 1; fill it so subsequent Receive calls would block if not dropped.
	ch := make(chan protocol.WorldSnapshot, 1)
	ch <- protocol.WorldSnapshot{} // pre-fill
	h.addStream(ch)
	defer h.removeStream(ch)

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Receive(protocol.WorldSnapshot{Speed: 1.0}) // must not block
	}()

	select {
	case <-done:
		// pass: Receive returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Receive blocked on a full stream channel")
	}
}

// TestWorldHandler_RemoveStream_Deregisters confirms that a removed stream no
// longer receives snapshots after deregistration.
func TestWorldHandler_RemoveStream_Deregisters(t *testing.T) {
	h := NewWorldHandler()

	ch := make(chan protocol.WorldSnapshot, 1)
	h.addStream(ch)
	h.removeStream(ch)

	h.Receive(protocol.WorldSnapshot{Speed: 7.0})

	select {
	case <-ch:
		t.Error("snapshot delivered to removed stream")
	default:
		// pass: channel is empty as expected
	}
}

// TestWorldHandler_MultipleStreams_AllReceive confirms fan-out delivers to all
// registered stream channels.
func TestWorldHandler_MultipleStreams_AllReceive(t *testing.T) {
	h := NewWorldHandler()

	const n = 4
	channels := make([]chan protocol.WorldSnapshot, n)
	for i := range channels {
		channels[i] = make(chan protocol.WorldSnapshot, 1)
		h.addStream(channels[i])
	}
	defer func() {
		for _, ch := range channels {
			h.removeStream(ch)
		}
	}()

	snap := protocol.WorldSnapshot{Speed: 42.0}
	h.Receive(snap)

	var wg sync.WaitGroup
	for i, ch := range channels {
		wg.Add(1)
		go func(idx int, c chan protocol.WorldSnapshot) {
			defer wg.Done()
			select {
			case got := <-c:
				if got.Speed != snap.Speed {
					t.Errorf("stream %d: Speed: want %v, got %v", idx, snap.Speed, got.Speed)
				}
			case <-time.After(time.Second):
				t.Errorf("stream %d: timeout waiting for snapshot", idx)
			}
		}(i, ch)
	}
	wg.Wait()
}
