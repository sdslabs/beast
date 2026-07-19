package cr

import (
	"fmt"
	"math"
)

const (
	minimumCPULimit       float32 = 0.01
	minimumCPUShares      int64   = 2
	minimumContainerBytes int64   = 6 << 20
	minimumPidsLimit      int64   = 1
)

func ValidateResourceLimits(cpuShares int64, cpus float32, memory, pids int64) error {
	if cpuShares < minimumCPUShares {
		return fmt.Errorf("cpu shares must be at least %d", minimumCPUShares)
	}
	if math.IsNaN(float64(cpus)) || math.IsInf(float64(cpus), 0) || cpus < minimumCPULimit {
		return fmt.Errorf("CPU limit must be finite and at least %.2f", minimumCPULimit)
	}
	if memory < minimumContainerBytes {
		return fmt.Errorf("memory limit must be at least %d bytes", minimumContainerBytes)
	}
	if pids < minimumPidsLimit {
		return fmt.Errorf("PID limit must be at least %d", minimumPidsLimit)
	}
	return nil
}

func CPUQuota(cpus float32) int64 {
	quota := int64(math.Ceil(float64(cpus) * 100000))
	if quota < 1000 {
		return 1000
	}
	return quota
}

func (limits BuildLimits) Validate() error {
	return ValidateResourceLimits(limits.CPUShares, limits.CPUs, limits.Memory, limits.Pids)
}
