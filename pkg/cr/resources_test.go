package cr

import (
	"math"
	"testing"
)

func TestValidateResourceLimitsRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name   string
		shares int64
		cpus   float32
		memory int64
		pids   int64
	}{
		{name: "CPU shares", shares: 1, cpus: 1, memory: minimumContainerBytes, pids: 1},
		{name: "CPU quota", shares: 2, cpus: 0.001, memory: minimumContainerBytes, pids: 1},
		{name: "CPU NaN", shares: 2, cpus: float32(math.NaN()), memory: minimumContainerBytes, pids: 1},
		{name: "memory", shares: 2, cpus: 1, memory: minimumContainerBytes - 1, pids: 1},
		{name: "PIDs", shares: 2, cpus: 1, memory: minimumContainerBytes, pids: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateResourceLimits(test.shares, test.cpus, test.memory, test.pids); err == nil {
				t.Fatal("expected resource validation error")
			}
		})
	}
}

func TestCPUQuotaNeverDisablesLimit(t *testing.T) {
	if quota := CPUQuota(0.000001); quota != 1000 {
		t.Fatalf("expected minimum quota, got %d", quota)
	}
	if quota := CPUQuota(0.25); quota != 25000 {
		t.Fatalf("expected quarter CPU quota, got %d", quota)
	}
}
