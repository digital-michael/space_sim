package factory

import (
	"testing"

	basepool "github.com/digital-michael/space_sim/internal/server/pool"
	"github.com/google/uuid"
)

// ─── Unit tests ────────────────────────────────────────────────────────────

func TestNew_AllImplementedTypes_NonNil(t *testing.T) {
	implemented := []basepool.PoolType{
		basepool.PoolTypeSimple,
		basepool.PoolTypeGroup,
		basepool.PoolTypeDistributed,
	}
	for _, pt := range implemented {
		p, err := New(pt)
		if err != nil {
			t.Errorf("New(%v): unexpected error: %v", pt, err)
		}
		if p == nil {
			t.Errorf("New(%v): returned nil pool", pt)
		}
	}
}

func TestNew_TypeRoundtrip(t *testing.T) {
	cases := []basepool.PoolType{
		basepool.PoolTypeSimple,
		basepool.PoolTypeGroup,
		basepool.PoolTypeDistributed,
	}
	for _, want := range cases {
		p, err := New(want)
		if err != nil {
			t.Fatalf("New(%v): %v", want, err)
		}
		if got := p.GetType(); got != want {
			t.Errorf("GetType: want %v, got %v", want, got)
		}
	}
}

func TestNew_UnknownType_ReturnsError(t *testing.T) {
	_, err := New(basepool.PoolType(999))
	if err == nil {
		t.Fatal("expected error for unknown PoolType, got nil")
	}
}

func TestNew_HybridType_ReturnsError(t *testing.T) {
	_, err := New(basepool.PoolTypeHybrid)
	if err == nil {
		t.Fatal("expected error for unimplemented PoolTypeHybrid, got nil")
	}
}

// ─── Benchmarks ────────────────────────────────────────────────────────────

// benchCreateGetDelete runs a Create → Get → Delete cycle n times on pool p.
func benchCreateGetDelete(b *testing.B, pt basepool.PoolType) {
	b.Helper()
	p, err := New(pt)
	if err != nil {
		b.Fatalf("factory.New(%v): %v", pt, err)
	}

	ids := make([]uuid.UUID, b.N)
	for i := range ids {
		ids[i] = uuid.New()
	}

	props := map[string]interface{}{"mass": 1.0, "radius": 6371.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := ids[i]
		if err := p.Create(id, "planet", props); err != nil {
			b.Fatalf("Create: %v", err)
		}
		if _, err := p.Get(id); err != nil {
			b.Fatalf("Get: %v", err)
		}
		if err := p.Delete(id); err != nil {
			b.Fatalf("Delete: %v", err)
		}
	}
}

func BenchmarkSimplePool_CreateGetDelete(b *testing.B) {
	benchCreateGetDelete(b, basepool.PoolTypeSimple)
}

func BenchmarkGroupPool_CreateGetDelete(b *testing.B) {
	benchCreateGetDelete(b, basepool.PoolTypeGroup)
}
