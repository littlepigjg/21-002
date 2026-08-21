package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket 实现经典令牌桶限流算法。
// capacity 为桶容量（最大令牌数），rate 为每秒补充的令牌数。
type TokenBucket struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	tokens   float64
	last     time.Time
}

// NewTokenBucket 构造一个初始满桶的令牌桶。
func NewTokenBucket(capacity, rate float64) *TokenBucket {
	return &TokenBucket{
		capacity: capacity,
		rate:     rate,
		tokens:   capacity,
		last:     time.Now(),
	}
}

// Allow 尝试获取一个令牌。成功获取返回 true，否则返回 false。
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Tokens 返回当前剩余令牌数（观测用）。
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tokens
}
