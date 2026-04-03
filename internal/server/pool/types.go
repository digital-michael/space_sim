package pool

import "github.com/google/uuid"

type PoolType int

const (
	PoolTypeSimple PoolType = iota
	PoolTypeGroup
	PoolTypeDistributed
	PoolTypeHybrid
)

func (pt PoolType) String() string {
	switch pt {
	case PoolTypeSimple:
		return "Simple"
	case PoolTypeGroup:
		return "Group"
	case PoolTypeDistributed:
		return "Distributed"
	case PoolTypeHybrid:
		return "Hybrid"
	default:
		return "Unknown"
	}
}

type ObjectPool interface {
	GetType() PoolType
	Create(id uuid.UUID, objType string, properties map[string]interface{}) error
	Get(id uuid.UUID) (interface{}, error)
	Update(id uuid.UUID, properties map[string]interface{}) error
	Delete(id uuid.UUID) error
	List() []uuid.UUID
}
