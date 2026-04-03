package group

import "github.com/google/uuid"

type DAG struct {
	edges map[uuid.UUID]map[uuid.UUID]struct{}
}

func NewDAG() *DAG {
	return &DAG{edges: make(map[uuid.UUID]map[uuid.UUID]struct{})}
}

func (dag *DAG) EnsureNode(id uuid.UUID) {
	if _, exists := dag.edges[id]; !exists {
		dag.edges[id] = make(map[uuid.UUID]struct{})
	}
}

func (dag *DAG) AddEdge(parentID, childID uuid.UUID) {
	dag.EnsureNode(parentID)
	dag.EnsureNode(childID)
	dag.edges[parentID][childID] = struct{}{}
}

func (dag *DAG) RemoveEdge(parentID, childID uuid.UUID) {
	if children, exists := dag.edges[parentID]; exists {
		delete(children, childID)
	}
}

func (dag *DAG) RemoveNode(nodeID uuid.UUID) {
	delete(dag.edges, nodeID)
	for _, children := range dag.edges {
		delete(children, nodeID)
	}
}

func (dag *DAG) WouldCreateCycle(parentID, childID uuid.UUID) bool {
	if parentID == childID {
		return true
	}
	return dag.hasPath(childID, parentID)
}

func (dag *DAG) hasPath(sourceID, targetID uuid.UUID) bool {
	if sourceID == targetID {
		return true
	}

	visited := make(map[uuid.UUID]struct{})
	stack := []uuid.UUID{sourceID}

	for len(stack) > 0 {
		last := len(stack) - 1
		current := stack[last]
		stack = stack[:last]

		if current == targetID {
			return true
		}
		if _, seen := visited[current]; seen {
			continue
		}
		visited[current] = struct{}{}

		for childID := range dag.edges[current] {
			if _, seen := visited[childID]; !seen {
				stack = append(stack, childID)
			}
		}
	}

	return false
}
