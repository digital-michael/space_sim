package engine

import "testing"

func TestDoubleBufferSwapUsesObjectPoolWhenCloneModeEnabled(t *testing.T) {
	state := NewSimulationState()
	state.AddObject(&Object{
		Meta:    ObjectMetadata{Name: "Earth", Category: CategoryPlanet, PhysicalRadius: 1},
		Anim:    AnimationState{},
		Visible: true,
		Dataset: -1,
	})

	pool := NewObjectPool()
	db := newDoubleBufferWithPool(state, pool)
	db.DisableInPlaceSwap()

	back := db.GetBack()
	back.Objects[0].Anim.Position = Vector3{X: 10}
	db.Swap()

	stats := pool.Stats()
	if stats.Borrows != 1 {
		t.Fatalf("expected 1 pool borrow after first clone swap, got %d", stats.Borrows)
	}
	if stats.Returns != 0 {
		t.Fatalf("expected 0 pool returns after first clone swap, got %d", stats.Returns)
	}
	if stats.InUse != 1 {
		t.Fatalf("expected 1 pooled object in use after first clone swap, got %d", stats.InUse)
	}

	front := db.GetFront()
	if front.Objects[0] == back.Objects[0] {
		t.Fatal("expected clone-mode front buffer to use a distinct object instance")
	}

	back.Objects[0].Anim.Position = Vector3{X: 20}
	db.Swap()

	stats = pool.Stats()
	if stats.Borrows != 2 {
		t.Fatalf("expected 2 pool borrows after second clone swap, got %d", stats.Borrows)
	}
	if stats.Returns != 1 {
		t.Fatalf("expected 1 pool return after second clone swap, got %d", stats.Returns)
	}
	if stats.InUse != 1 {
		t.Fatalf("expected 1 pooled object still in use after second clone swap, got %d", stats.InUse)
	}
}
