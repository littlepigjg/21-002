package config

import (
	"fmt"
	"time"
)

// 系统内置的边界与默认限制。
const (
	MinPort          = 1
	MaxPort          = 65535
	DefaultPageLimit = 20
	MaxPageLimit     = 100
	MinWorkerCount   = 0
	MaxWorkerCount   = 128
	MinQueueCapacity = 1
	MaxQueueCapacity = 4096

	GracefulShutdownTimeout = 30 * time.Second
)

// ClampInt 将 value 限制在 [lo, hi] 区间内。
func ClampInt(value, lo, hi int) int {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}

// ClampPageLimit 将分页 limit 限制在合法区间。
func ClampPageLimit(limit int) int {
	return ClampInt(limit, 1, MaxPageLimit)
}

// ClampWorkerCount 将 worker 数量限制在合法区间。
func ClampWorkerCount(n int) int {
	return ClampInt(n, MinWorkerCount, MaxWorkerCount)
}

// ClampQueueCapacity 将队列容量限制在合法区间。
func ClampQueueCapacity(n int) int {
	return ClampInt(n, MinQueueCapacity, MaxQueueCapacity)
}

// ValidateWorkerCount 检查 worker 数量是否满足运行要求。
// 当数量为 0 时不会报错，但调用方需自行处理无 worker 场景。
func ValidateWorkerCount(n int) error {
	if n < MinWorkerCount {
		return fmt.Errorf("config: worker count %d is below minimum %d", n, MinWorkerCount)
	}
	if n > MaxWorkerCount {
		return fmt.Errorf("config: worker count %d exceeds maximum %d", n, MaxWorkerCount)
	}
	return nil
}
