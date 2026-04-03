package simple

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestSimplePool_Create_and_Get(t *testing.T) {
	p := NewPool()
	id := uuid.New()

	if err := p.Create(id, "planet", map[string]interface{}{"mass": 1.0}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := p.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	def, ok := got.(*ObjectDefinition)
	if !ok {
		t.Fatalf("Get returned %T, want *ObjectDefinition", got)
	}
	if def.ID != id {
		t.Errorf("ID: want %v, got %v", id, def.ID)
	}
	if def.Type != "planet" {
		t.Errorf("Type: want planet, got %s", def.Type)
	}
}

func TestSimplePool_Create_NilID_Errors(t *testing.T) {
	p := NewPool()
	if err := p.Create(uuid.Nil, "star", nil); err == nil {
		t.Fatal("expected error for nil UUID")
	}
}

func TestSimplePool_Create_EmptyType_Errors(t *testing.T) {
	p := NewPool()
	if err := p.Create(uuid.New(), "", nil); err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestSimplePool_Create_Duplicate_ReturnsErrObjectExists(t *testing.T) {
	p := NewPool()
	id := uuid.New()
	p.Create(id, "moon", nil) //nolint:errcheck
	if err := p.Create(id, "moon", nil); !errors.Is(err, ErrObjectExists) {
		t.Errorf("want ErrObjectExists, got %v", err)
	}
}

func TestSimplePool_Get_NotFound_ReturnsError(t *testing.T) {
	p := NewPool()
	_, err := p.Get(uuid.New())
	if !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("want ErrObjectNotFound, got %v", err)
	}
}

func TestSimplePool_Get_ReturnClone_IsolatesMutation(t *testing.T) {
	p := NewPool()
	id := uuid.New()
	p.Create(id, "asteroid", map[string]interface{}{"x": 1.0}) //nolint:errcheck

	got, _ := p.Get(id)
	def := got.(*ObjectDefinition)
	def.Properties["x"] = 99.0 // mutate the clone

	got2, _ := p.Get(id)
	def2 := got2.(*ObjectDefinition)
	if def2.Properties["x"] != 1.0 {
		t.Errorf("pool state was mutated through returned clone")
	}
}

func TestSimplePool_Update_MergesProperties(t *testing.T) {
	p := NewPool()
	id := uuid.New()
	p.Create(id, "moon", map[string]interface{}{"a": 1}) //nolint:errcheck

	if err := p.Update(id, map[string]interface{}{"b": 2}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := p.Get(id)
	def := got.(*ObjectDefinition)
	if def.Properties["a"] != 1 {
		t.Errorf("original property 'a' should be preserved")
	}
	if def.Properties["b"] != 2 {
		t.Errorf("new property 'b' should be set")
	}
}

func TestSimplePool_Update_NotFound_ReturnsError(t *testing.T) {
	p := NewPool()
	if err := p.Update(uuid.New(), nil); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("want ErrObjectNotFound, got %v", err)
	}
}

func TestSimplePool_Delete(t *testing.T) {
	p := NewPool()
	id := uuid.New()
	p.Create(id, "ring", nil) //nolint:errcheck

	if err := p.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := p.Get(id); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("expected object gone after Delete")
	}
}

func TestSimplePool_Delete_NotFound_ReturnsError(t *testing.T) {
	p := NewPool()
	if err := p.Delete(uuid.New()); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("want ErrObjectNotFound, got %v", err)
	}
}

func TestSimplePool_List(t *testing.T) {
	p := NewPool()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for _, id := range ids {
		p.Create(id, "belt", nil) //nolint:errcheck
	}

	listed := p.List()
	if len(listed) != 3 {
		t.Fatalf("List: want 3, got %d", len(listed))
	}
}

func TestSimplePool_GetType(t *testing.T) {
	if NewPool().GetType().String() != "Simple" {
		t.Errorf("GetType should return PoolTypeSimple")
	}
}

func TestSimplePool_Concurrent(t *testing.T) {
	p := NewPool()
	const goroutines = 20
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	ids := make([]uuid.UUID, goroutines*perGoroutine)
	for i := range ids {
		ids[i] = uuid.New()
	}

	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			base := g * perGoroutine
			for i := 0; i < perGoroutine; i++ {
				p.Create(ids[base+i], "star", nil) //nolint:errcheck
			}
		}()
	}
	wg.Wait()

	if got := len(p.List()); got != goroutines*perGoroutine {
		t.Errorf("want %d objects after concurrent creates, got %d", goroutines*perGoroutine, got)
	}
}
