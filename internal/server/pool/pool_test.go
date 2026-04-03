package pool

import (
	"testing"
)

func TestPool_Add(t *testing.T) {
	p := New(10, 100)

	obj := NewObject("cube")
	err := p.Add(obj)
	if err != nil {
		t.Fatalf("Failed to add object: %v", err)
	}

	if p.Count() != 1 {
		t.Errorf("Expected count 1, got %d", p.Count())
	}
}

func TestPool_Get(t *testing.T) {
	p := New(10, 100)

	obj := NewObject("sphere")
	p.Add(obj)

	retrieved, err := p.Get(obj.GUID)
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}

	if retrieved.GUID != obj.GUID {
		t.Errorf("Expected GUID %s, got %s", obj.GUID, retrieved.GUID)
	}
}

func TestPool_Delete(t *testing.T) {
	p := New(10, 100)

	obj := NewObject("cylinder")
	p.Add(obj)

	err := p.Delete(obj.GUID)
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	if p.Count() != 0 {
		t.Errorf("Expected count 0 after delete, got %d", p.Count())
	}
}

func TestPool_MaxObjects(t *testing.T) {
	p := New(1, 2)

	obj1 := NewObject("cube")
	obj2 := NewObject("sphere")
	obj3 := NewObject("cylinder")

	p.Add(obj1)
	p.Add(obj2)
	err := p.Add(obj3)

	if err != ErrPoolFull {
		t.Errorf("Expected ErrPoolFull, got %v", err)
	}
}

func TestPool_List(t *testing.T) {
	p := New(10, 100)

	p.Add(NewObject("cube"))
	p.Add(NewObject("sphere"))
	p.Add(NewObject("cube"))

	all := p.List("")
	if len(all) != 3 {
		t.Errorf("Expected 3 objects, got %d", len(all))
	}

	cubes := p.List("cube")
	if len(cubes) != 2 {
		t.Errorf("Expected 2 cubes, got %d", len(cubes))
	}
}

func TestPool_Concurrent(t *testing.T) {
	p := New(100, 1000)

	// Test concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				obj := NewObject("test")
				p.Add(obj)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if p.Count() != 100 {
		t.Errorf("Expected 100 objects after concurrent adds, got %d", p.Count())
	}
}
