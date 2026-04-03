package runtime

import "github.com/google/uuid"

func (environment *RuntimeEnvironment) invalidateGroupCachesLocked(memberID uuid.UUID) {
	if environment.definitions == nil {
		return
	}

	visited := map[uuid.UUID]struct{}{}
	queue := []uuid.UUID{memberID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, groupID := range environment.definitions.GroupsForMember(current) {
			if _, seen := visited[groupID]; seen {
				continue
			}
			visited[groupID] = struct{}{}

			if state, exists := environment.groups[groupID]; exists {
				state.CachedValid = false
			}
			queue = append(queue, groupID)
		}
	}
}
