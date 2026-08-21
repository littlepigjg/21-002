package config

// 系统内置的边界与默认限制。
const (
	MinPort          = 1
	MaxPort          = 65535
	DefaultPageLimit = 20
	MaxPageLimit     = 100
	MinWorkerCount   = 1
	MaxWorkerCount   = 128
	MinQueueCapacity = 1
	MaxQueueCapacity = 4096
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
