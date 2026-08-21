package cache

import (
	"container/list"
	"sync"
)

// lruEntry 是 LRU 链表中的单个节点。
type lruEntry struct {
	key   string
	value interface{}
}

// LRU 是线程安全的最近最少使用缓存。
type LRU struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

// NewLRU 构造指定容量的 LRU 缓存，容量小于等于 0 时回退为 1。
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

// Get 返回 key 对应的值；命中时会将该项移动到最近使用位置。
func (c *LRU) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*lruEntry).value, true
	}
	return nil, false
}

// Put 写入键值对；若 key 已存在则更新值并移动到最近使用位置。
// 超过容量时淘汰最久未使用的项。
func (c *LRU) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

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

// Remove 删除指定 key，若 key 不存在则无操作。
func (c *LRU) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.Remove(el)
		delete(c.items, key)
	}
}

// Len 返回当前缓存项数量。
func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
