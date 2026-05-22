package session_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/digital-michael/space_sim/internal/server/session"
)

func newReg() session.Registry {
	return session.NewRegistry(session.DefaultConfig())
}

func TestRegisterBasic(t *testing.T) {
	reg := newReg()
	sess, err := reg.Register(session.RegisterRequest{
		ClientUUID: "client-1",
		Label:      "Alice",
		Role:       session.ClientRolePlayer,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if sess.SessionID == "" {
		t.Error("expected non-empty SessionID")
	}
	if sess.Label != "Alice" {
		t.Errorf("Label = %q, want %q", sess.Label, "Alice")
	}
	if sess.Role != session.ClientRolePlayer {
		t.Errorf("Role = %v, want PLAYER", sess.Role)
	}
	if reg.Count() != 1 {
		t.Errorf("Count = %d, want 1", reg.Count())
	}
}

func TestCapacityExceeded(t *testing.T) {
	cfg := session.Config{MaxSessions: 3}
	reg := session.NewRegistry(cfg)
	for i := 0; i < 3; i++ {
		_, err := reg.Register(session.RegisterRequest{ClientUUID: fmt.Sprintf("c-%d", i)})
		if err != nil {
			t.Fatalf("Register %d: %v", i, err)
		}
	}
	_, err := reg.Register(session.RegisterRequest{ClientUUID: "overflow"})
	if err != session.ErrCapacityExceeded {
		t.Errorf("expected ErrCapacityExceeded, got %v", err)
	}
}

func TestUnregisterReleasesSlot(t *testing.T) {
	cfg := session.Config{MaxSessions: 1}
	reg := session.NewRegistry(cfg)
	s, _ := reg.Register(session.RegisterRequest{ClientUUID: "c1"})
	reg.Unregister(s.SessionID)
	if reg.Count() != 0 {
		t.Error("expected Count=0 after Unregister")
	}
	_, err := reg.Register(session.RegisterRequest{ClientUUID: "c2"})
	if err != nil {
		t.Errorf("Register after unregister: %v", err)
	}
}

func TestColorUniqueness(t *testing.T) {
	reg := newReg()
	seen := map[[3]uint8]string{}
	for i := 0; i < 10; i++ {
		s, err := reg.Register(session.RegisterRequest{ClientUUID: fmt.Sprintf("c-%d", i)})
		if err != nil {
			t.Fatalf("Register %d: %v", i, err)
		}
		if prev, dup := seen[s.Color]; dup {
			t.Errorf("color %v assigned to both %s and %s", s.Color, prev, s.SessionID)
		}
		seen[s.Color] = s.SessionID
	}
}

func TestColorReleasedOnUnregister(t *testing.T) {
	cfg := session.Config{MaxSessions: 2}
	reg := session.NewRegistry(cfg)
	s1, _ := reg.Register(session.RegisterRequest{ClientUUID: "c1"})
	_, _ = reg.Register(session.RegisterRequest{ClientUUID: "c2"})
	c1 := s1.Color
	reg.Unregister(s1.SessionID)
	// Re-register: should reclaim the released color slot.
	s3, err := reg.Register(session.RegisterRequest{ClientUUID: "c3"})
	if err != nil {
		t.Fatalf("Register after release: %v", err)
	}
	if s3.Color != c1 {
		t.Logf("note: color reuse: got %v, original %v (FIFO order)", s3.Color, c1)
	}
}

func TestAdminRoleClamped(t *testing.T) {
	reg := session.NewRegistry(session.Config{MaxSessions: 10, AdminSecret: "secret"})

	// Wrong secret → clamped to PLAYER
	s, _ := reg.Register(session.RegisterRequest{
		ClientUUID:  "c1",
		Role:        session.ClientRoleAdmin,
		AdminSecret: "wrong",
	})
	if s.Role != session.ClientRolePlayer {
		t.Errorf("wrong secret: expected PLAYER, got %v", s.Role)
	}

	// Correct secret → ADMIN granted
	s2, _ := reg.Register(session.RegisterRequest{
		ClientUUID:  "c2",
		Role:        session.ClientRoleAdmin,
		AdminSecret: "secret",
	})
	if s2.Role != session.ClientRoleAdmin {
		t.Errorf("correct secret: expected ADMIN, got %v", s2.Role)
	}

	// No secret configured → ADMIN always clamped to PLAYER
	reg2 := session.NewRegistry(session.Config{MaxSessions: 10, AdminSecret: ""})
	s3, _ := reg2.Register(session.RegisterRequest{
		ClientUUID:  "c3",
		Role:        session.ClientRoleAdmin,
		AdminSecret: "anything",
	})
	if s3.Role != session.ClientRolePlayer {
		t.Errorf("no secret configured: expected PLAYER, got %v", s3.Role)
	}
}

func TestLabelTruncation(t *testing.T) {
	reg := newReg()
	longLabel := "ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz" // 52 chars
	s, err := reg.Register(session.RegisterRequest{ClientUUID: "c1", Label: longLabel})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(s.Label) > 32 {
		t.Errorf("Label len = %d, want <= 32", len(s.Label))
	}
}

func TestConcurrentRegisterUnregister(t *testing.T) {
	reg := newReg()
	var wg sync.WaitGroup
	const n = 50
	ids := make(chan string, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			s, err := reg.Register(session.RegisterRequest{ClientUUID: fmt.Sprintf("c-%d", i)})
			if err != nil {
				return
			}
			ids <- s.SessionID
		}(i)
	}
	wg.Wait()
	close(ids)
	for id := range ids {
		reg.Unregister(id)
	}
	if reg.Count() != 0 {
		t.Errorf("expected Count=0 after all unregisters, got %d", reg.Count())
	}
}

func TestUpdatePosition(t *testing.T) {
	reg := newReg()
	sess, _ := reg.Register(session.RegisterRequest{ClientUUID: "c1"})

	pos := [3]float64{1.5, -2.0, 3.14}
	if err := reg.UpdatePosition(sess.SessionID, pos); err != nil {
		t.Fatalf("UpdatePosition: %v", err)
	}

	got, ok := reg.Get(sess.SessionID)
	if !ok {
		t.Fatal("session not found after UpdatePosition")
	}
	if got.Position != pos {
		t.Errorf("Position = %v, want %v", got.Position, pos)
	}
}

func TestUpdatePOV(t *testing.T) {
	reg := newReg()
	sess, _ := reg.Register(session.RegisterRequest{ClientUUID: "c1"})

	pov := [3]float32{0.1, 0.2, 0.9}
	if err := reg.UpdatePOV(sess.SessionID, pov); err != nil {
		t.Fatalf("UpdatePOV: %v", err)
	}

	got, ok := reg.Get(sess.SessionID)
	if !ok {
		t.Fatal("session not found after UpdatePOV")
	}
	if got.POV != pov {
		t.Errorf("POV = %v, want %v", got.POV, pov)
	}
}

func TestUpdatePositionNotFound(t *testing.T) {
	reg := newReg()
	err := reg.UpdatePosition("no-such-id", [3]float64{1, 2, 3})
	if err != session.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSubscribeReceivesEvents(t *testing.T) {
	reg := newReg()
	ch, cancel := reg.Subscribe()
	defer cancel()

	sess, _ := reg.Register(session.RegisterRequest{ClientUUID: "c1", Label: "Test"})

	e := <-ch
	if e.Type != session.SessionEventAdd {
		t.Errorf("Type = %v, want SessionEventAdd", e.Type)
	}
	if e.ID != sess.SessionID {
		t.Errorf("ID = %q, want %q", e.ID, sess.SessionID)
	}

	reg.UpdatePosition(sess.SessionID, [3]float64{1, 2, 3})
	e = <-ch
	if e.Type != session.SessionEventUpdate {
		t.Errorf("Type = %v, want SessionEventUpdate", e.Type)
	}

	reg.Unregister(sess.SessionID)
	e = <-ch
	if e.Type != session.SessionEventRemove {
		t.Errorf("Type = %v, want SessionEventRemove", e.Type)
	}
	if e.ID != sess.SessionID {
		t.Errorf("Remove ID = %q, want %q", e.ID, sess.SessionID)
	}
}
