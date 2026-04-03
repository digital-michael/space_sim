package runtime

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/digital-michael/space_sim/internal/server/pool/group"
	"github.com/google/uuid"
)

func BenchmarkGroupStateGetCacheHit(b *testing.B) {
	definitions := group.NewPool()
	groupID := uuid.New()
	objectIDs := make([]uuid.UUID, 100)

	if err := definitions.CreateGroup(groupID, "large", nil, nil); err != nil {
		b.Fatalf("CreateGroup failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		id := uuid.New()
		objectIDs[i] = id
		if err := definitions.CreateObject(id, fmt.Sprintf("obj_%d", i), nil); err != nil {
			b.Fatalf("CreateObject failed: %v", err)
		}
		if err := definitions.AddGroupMember(groupID, id); err != nil {
			b.Fatalf("AddGroupMember failed: %v", err)
		}
	}

	environment := NewRuntimeEnvironment(definitions)
	for i, id := range objectIDs {
		idx := i
		if err := environment.InitializeObject(id, func(j int) Vector3 {
			return Vector3{X: float64(idx), Y: float64(idx % 10), Z: float64(idx % 5)}
		}, Vector3{}); err != nil {
			b.Fatalf("InitializeObject failed: %v", err)
		}
	}

	if err := environment.InitializeGroup(groupID); err != nil {
		b.Fatalf("InitializeGroup failed: %v", err)
	}

	state, _ := environment.GetGroupState(groupID)
	if !environment.groups[groupID].CachedValid {
		b.Fatal("expected cache to be valid after first read")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = environment.GetGroupState(groupID)
	}

	finalState, _ := environment.GetGroupState(groupID)
	if finalState.MemberCount != 100 {
		b.Fatalf("unexpected member count: %d", finalState.MemberCount)
	}
	if state.Center != finalState.Center {
		b.Fatalf("state changed during benchmark")
	}
}

func BenchmarkGroupStateGetCacheMiss(b *testing.B) {
	definitions := group.NewPool()
	groupID := uuid.New()
	objectIDs := make([]uuid.UUID, 100)

	if err := definitions.CreateGroup(groupID, "large", nil, nil); err != nil {
		b.Fatalf("CreateGroup failed: %v", err)
	}

	for i := 0; i < 100; i++ {
		id := uuid.New()
		objectIDs[i] = id
		if err := definitions.CreateObject(id, fmt.Sprintf("obj_%d", i), nil); err != nil {
			b.Fatalf("CreateObject failed: %v", err)
		}
		if err := definitions.AddGroupMember(groupID, id); err != nil {
			b.Fatalf("AddGroupMember failed: %v", err)
		}
	}

	environment := NewRuntimeEnvironment(definitions)
	for i, id := range objectIDs {
		idx := i
		if err := environment.InitializeObject(id, func(j int) Vector3 {
			return Vector3{X: float64(idx), Y: float64(idx % 10), Z: float64(idx % 5)}
		}, Vector3{}); err != nil {
			b.Fatalf("InitializeObject failed: %v", err)
		}
	}

	if err := environment.InitializeGroup(groupID); err != nil {
		b.Fatalf("InitializeGroup failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		environment.groups[groupID].CachedValid = false
		b.StartTimer()
		_, _ = environment.GetGroupState(groupID)
	}

	finalState, _ := environment.GetGroupState(groupID)
	if finalState.MemberCount != 100 {
		b.Fatalf("unexpected member count: %d", finalState.MemberCount)
	}
}

func BenchmarkCacheInvalidationPropagation(b *testing.B) {
	definitions := group.NewPool()
	objectID := uuid.New()
	groupIDs := make([]uuid.UUID, 5)

	if err := definitions.CreateObject(objectID, "sphere", nil); err != nil {
		b.Fatalf("CreateObject failed: %v", err)
	}

	parentID := uuid.New()
	if err := definitions.CreateGroup(parentID, "root", nil, nil); err != nil {
		b.Fatalf("CreateGroup root failed: %v", err)
	}
	groupIDs[0] = parentID

	for i := 1; i < 5; i++ {
		id := uuid.New()
		groupIDs[i] = id
		prevID := groupIDs[i-1]
		if err := definitions.CreateGroup(id, fmt.Sprintf("group_%d", i), &prevID, nil); err != nil {
			b.Fatalf("CreateGroup %d failed: %v", i, err)
		}
	}

	if err := definitions.AddGroupMember(groupIDs[4], objectID); err != nil {
		b.Fatalf("AddGroupMember failed: %v", err)
	}

	environment := NewRuntimeEnvironment(definitions)
	if err := environment.InitializeObject(objectID, OriginPosition(), Vector3{}); err != nil {
		b.Fatalf("InitializeObject failed: %v", err)
	}

	for _, groupID := range groupIDs {
		if err := environment.InitializeGroup(groupID); err != nil {
			b.Fatalf("InitializeGroup failed: %v", err)
		}
	}

	for _, groupID := range groupIDs {
		_, _ = environment.GetGroupState(groupID)
	}
	for _, groupID := range groupIDs {
		if !environment.groups[groupID].CachedValid {
			b.Fatal("expected all caches to be valid")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = environment.UpdateObjectState(objectID, func(state *ObjectState) {
			state.Position.X += 1.0
		})
	}

	for _, groupID := range groupIDs {
		if environment.groups[groupID].CachedValid {
			b.Fatalf("expected cache for %s to be invalidated", groupID)
		}
	}
}

func BenchmarkBatchQueryOperations(b *testing.B) {
	definitions := group.NewPool()
	objectIDs := make([]uuid.UUID, 50)
	groupIDs := make([]uuid.UUID, 10)

	for i := 0; i < 50; i++ {
		id := uuid.New()
		objectIDs[i] = id
		if err := definitions.CreateObject(id, fmt.Sprintf("obj_%d", i), nil); err != nil {
			b.Fatalf("CreateObject failed: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		id := uuid.New()
		groupIDs[i] = id
		if err := definitions.CreateGroup(id, fmt.Sprintf("group_%d", i), nil, nil); err != nil {
			b.Fatalf("CreateGroup failed: %v", err)
		}
		for j := 0; j < 5; j++ {
			if err := definitions.AddGroupMember(id, objectIDs[i*5+j]); err != nil {
				b.Fatalf("AddGroupMember failed: %v", err)
			}
		}
	}

	environment := NewRuntimeEnvironment(definitions)
	for i, id := range objectIDs {
		idx := i
		if err := environment.InitializeObject(id, func(j int) Vector3 {
			return Vector3{X: float64(idx), Y: 0, Z: 0}
		}, Vector3{}); err != nil {
			b.Fatalf("InitializeObject failed: %v", err)
		}
	}

	for _, id := range groupIDs {
		if err := environment.InitializeGroup(id); err != nil {
			b.Fatalf("InitializeGroup failed: %v", err)
		}
	}

	queriedGroupIDs := groupIDs[:5]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = environment.GetAggregatesByIDs(queriedGroupIDs)
	}
}

func BenchmarkRandomAccessPattern(b *testing.B) {
	definitions := group.NewPool()
	objectIDs := make([]uuid.UUID, 100)

	for i := 0; i < 100; i++ {
		id := uuid.New()
		objectIDs[i] = id
		if err := definitions.CreateObject(id, fmt.Sprintf("obj_%d", i), nil); err != nil {
			b.Fatalf("CreateObject failed: %v", err)
		}
	}

	environment := NewRuntimeEnvironment(definitions)
	for i, id := range objectIDs {
		idx := i
		if err := environment.InitializeObject(id, func(j int) Vector3 {
			return Vector3{X: float64(idx), Y: 0, Z: 0}
		}, Vector3{X: 1, Y: 0, Z: 0}); err != nil {
			b.Fatalf("InitializeObject failed: %v", err)
		}
	}

	rng := rand.New(rand.NewSource(42))
	queryIDs := make([]uuid.UUID, 10)
	for i := 0; i < 10; i++ {
		queryIDs[i] = objectIDs[rng.Intn(100)]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = environment.GetObjectStatesByIDs(queryIDs)
	}
}
