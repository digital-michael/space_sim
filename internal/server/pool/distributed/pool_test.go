package distributed

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestDistributedPool_GetType(t *testing.T) {
	if NewPool().GetType().String() != "Distributed" {
		t.Errorf("GetType should return PoolTypeDistributed")
	}
}

func TestDistributedPool_AllMethods_ReturnErrNotImplemented(t *testing.T) {
	p := NewPool()
	id := uuid.New()

	if err := p.Create(id, "star", nil); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Create: want ErrNotImplemented, got %v", err)
	}
	if _, err := p.Get(id); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Get: want ErrNotImplemented, got %v", err)
	}
	if err := p.Update(id, nil); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Update: want ErrNotImplemented, got %v", err)
	}
	if err := p.Delete(id); !errors.Is(err, ErrNotImplemented) {
		t.Errorf("Delete: want ErrNotImplemented, got %v", err)
	}
	if ids := p.List(); ids != nil {
		t.Errorf("List: want nil, got %v", ids)
	}
}
