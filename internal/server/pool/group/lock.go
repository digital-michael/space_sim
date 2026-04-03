package group

import "github.com/google/uuid"

func (pool *Pool) LockGroupHierarchy(groupID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}

	group.Lock()
	for _, descendantID := range pool.descendantGroupIDs(groupID) {
		pool.groups[descendantID].Lock()
	}

	return nil
}

func (pool *Pool) UnlockGroupHierarchy(groupID uuid.UUID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	group, exists := pool.groups[groupID]
	if !exists {
		return ErrGroupNotFound
	}

	group.Unlock()
	for _, descendantID := range pool.descendantGroupIDs(groupID) {
		pool.groups[descendantID].Unlock()
	}

	return nil
}

func (pool *Pool) descendantGroupIDs(rootID uuid.UUID) []uuid.UUID {
	queue := []uuid.UUID{rootID}
	visited := map[uuid.UUID]struct{}{rootID: {}}
	descendants := make([]uuid.UUID, 0)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		group, exists := pool.groups[current]
		if !exists {
			continue
		}

		for _, memberID := range group.Members {
			if _, isGroup := pool.groups[memberID]; !isGroup {
				continue
			}
			if _, seen := visited[memberID]; seen {
				continue
			}
			visited[memberID] = struct{}{}
			descendants = append(descendants, memberID)
			queue = append(queue, memberID)
		}
	}

	return descendants
}
