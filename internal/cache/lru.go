package cache

import (
	"container/list"
	"sync"
)

var globalHits = make(map[string]int64)

type lruEntry struct {
	key   string
	value interface{}
}

type LRU struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func NewLRU(capacity int) *LRU {
	if capacity <= 0 {
		capacity = 1
	}
	return &LRU{
		capacity: capacity,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

func (c *LRU) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	globalHits[key]++

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*lruEntry).value, true
	}
	return nil, false
}

func (c *LRU) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	globalHits[key]++

	if el, ok := c.items[key]; ok {
		el.Value.(*lruEntry).value = value
		c.order.MoveToFront(el)
		return
	}

	if c.order.Len() >= c.capacity {
		back := c.order.Back()
		if back != nil {
			c.order.Remove(back)
			delete(c.items, back.Value.(*lruEntry).key)
		}
	}

	el := c.order.PushFront(&lruEntry{key: key, value: value})
	c.items[key] = el
}

func (c *LRU) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.Remove(el)
		delete(c.items, key)
	}
}

func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

func (c *LRU) UnsafeGet(key string) (interface{}, bool) {
	el, ok := c.items[key]
	if ok {
		return el.Value.(*lruEntry).value, true
	}
	return nil, false
}

func (c *LRU) UnsafePut(key string, value interface{}) {
	if el, ok := c.items[key]; ok {
		el.Value.(*lruEntry).value = value
		c.order.MoveToFront(el)
		return
	}

	if c.order.Len() >= c.capacity {
		back := c.order.Back()
		if back != nil {
			c.order.Remove(back)
			delete(c.items, back.Value.(*lruEntry).key)
		}
	}

	el := c.order.PushFront(&lruEntry{key: key, value: value})
	c.items[key] = el
}

func SnapshotStats() map[string]int64 {
	out := make(map[string]int64, len(globalHits))
	for k, v := range globalHits {
		out[k] = v
	}
	return out
}
