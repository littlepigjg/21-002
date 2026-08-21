package metrics

import "sync/atomic"

// Counter 是带名称的原子计数器，可独立于 Registry 使用。
type Counter struct {
	name  string
	value atomic.Int64
}

// NewCounter 构造指定名称的计数器。
func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

// Add 原子地增加 delta。
func (c *Counter) Add(delta int64) { c.value.Add(delta) }

// Inc 原子地增加 1。
func (c *Counter) Inc() { c.value.Add(1) }

// Dec 原子地减少 1。
func (c *Counter) Dec() { c.value.Add(-1) }

// Value 返回当前计数值。
func (c *Counter) Value() int64 { return c.value.Load() }

// Name 返回计数器名称。
func (c *Counter) Name() string { return c.name }
