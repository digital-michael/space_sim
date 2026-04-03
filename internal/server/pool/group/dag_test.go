package group

import (
	"testing"

	"github.com/google/uuid"
)

func TestDAGWouldCreateCycle(t *testing.T) {
	dag := NewDAG()
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()

	dag.AddEdge(a, b)
	dag.AddEdge(b, c)

	if !dag.WouldCreateCycle(c, a) {
		t.Fatal("expected C->A to create cycle")
	}
	if dag.WouldCreateCycle(a, c) {
		t.Fatal("expected A->C to be acyclic")
	}
}

func TestDAGRemoveNode(t *testing.T) {
	dag := NewDAG()
	a := uuid.New()
	b := uuid.New()

	dag.AddEdge(a, b)
	dag.RemoveNode(b)

	if dag.WouldCreateCycle(b, a) {
		t.Fatal("expected removed node to have no path")
	}
}
