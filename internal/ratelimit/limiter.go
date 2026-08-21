package ratelimit

import (
	"sync"
)

// Limiter 为多个 key（如客户端 IP）分别维护独立的令牌桶。
type Limiter struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	buckets  map[string]*TokenBucket
}

// NewLimiter 构造一个按 key 分桶的限流器。
func NewLimiter(capacity, rate float64) *Limiter {
	return &Limiter{
		capacity: capacity,
		rate:     rate,
		buckets:  make(map[string]*TokenBucket),
	}
}

// Allow 判断指定 key 是否被允许通过。
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	b, ok := l.buckets[key]
	if !ok {
		b = NewTokenBucket(l.capacity, l.rate)
		l.buckets[key] = b
	}
	l.mu.Unlock()
	return b.Allow()
}

// Remove 移除指定 key 的令牌桶，可用于清理长时间不活跃的 key。
func (l *Limiter) Remove(key string) {
	l.mu.Lock()
	delete(l.buckets, key)
	l.mu.Unlock()
}

// Size 返回当前维护的桶数量。
func (l *Limiter) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
