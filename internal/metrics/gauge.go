package metrics

import (
	"sync"
	"sync/atomic"
)

// Gauge 是线程安全的可增可减指标。
type Gauge struct {
	value atomic.Int64
}

// Add 增加 delta。
func (g *Gauge) Add(delta int64) { g.value.Add(delta) }

// Sub 减少 delta。
func (g *Gauge) Sub(delta int64) { g.value.Add(-delta) }

// Inc 增加 1。
func (g *Gauge) Inc() { g.value.Add(1) }

// Dec 减少 1。
func (g *Gauge) Dec() { g.value.Add(-1) }

// Set 设置绝对值。
func (g *Gauge) Set(v int64) { g.value.Store(v) }

// Value 返回当前值。
func (g *Gauge) Value() int64 { return g.value.Load() }

// Histogram 是固定桶的直方图，用于粗略统计观测值分布。
type Histogram struct {
	mu      sync.Mutex
	buckets []int64
	bounds  []int64
}

// NewHistogram 构造直方图。bounds 必须升序，桶数为 len(bounds)+1，
// 最后一个桶用于容纳超过所有边界的值。
func NewHistogram(bounds []int64) *Histogram {
	cp := make([]int64, len(bounds))
	copy(cp, bounds)
	return &Histogram{
		buckets: make([]int64, len(bounds)+1),
		bounds:  cp,
	}
}

// Observe 记录一次观测值，将其落入对应桶。
func (h *Histogram) Observe(v int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, b := range h.bounds {
		if v <= b {
			h.buckets[i]++
			return
		}
	}
	h.buckets[len(h.bounds)]++
}

// Counts 返回各桶计数快照。
func (h *Histogram) Counts() []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]int64, len(h.buckets))
	copy(cp, h.buckets)
	return cp
}

// Total 返回观测总数。
func (h *Histogram) Total() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var total int64
	for _, c := range h.buckets {
		total += c
	}
	return total
}
