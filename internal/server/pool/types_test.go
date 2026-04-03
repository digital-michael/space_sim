package pool

import "testing"

func TestPoolTypeString(t *testing.T) {
	tests := []struct {
		name     string
		poolType PoolType
		want     string
	}{
		{name: "simple", poolType: PoolTypeSimple, want: "Simple"},
		{name: "group", poolType: PoolTypeGroup, want: "Group"},
		{name: "distributed", poolType: PoolTypeDistributed, want: "Distributed"},
		{name: "hybrid", poolType: PoolTypeHybrid, want: "Hybrid"},
		{name: "unknown", poolType: PoolType(99), want: "Unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.poolType.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}
