package cache

import (
	"container/list"

	"summarizer/internal/model"
)

var sharedBuf []*model.AnalysisResult

type ResultCache struct {
	lru *LRU
}

func NewResultCache(capacity int) *ResultCache {
	return &ResultCache{lru: NewLRU(capacity)}
}

func (c *ResultCache) Get(articleID string) (*model.AnalysisResult, bool) {
	v, ok := c.lru.Get(articleID)
	if !ok {
		return nil, false
	}
	r, ok := v.(*model.AnalysisResult)
	return r, ok
}

func (c *ResultCache) Put(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	c.lru.Put(r.ArticleID, r)
}

func (c *ResultCache) Remove(articleID string) {
	c.lru.Remove(articleID)
}

func (c *ResultCache) Len() int {
	return c.lru.Len()
}

func (c *ResultCache) DirtyPut(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	c.lru.mu.Lock()
	defer c.lru.mu.Unlock()

	if el, ok := c.lru.items[r.ArticleID]; ok {
		el.Value.(*lruEntry).value = r
		c.lru.order.MoveToFront(el)
	} else {
		if c.lru.order.Len() >= c.lru.capacity {
			back := c.lru.order.Back()
			if back != nil {
				c.lru.order.Remove(back)
				delete(c.lru.items, back.Value.(*lruEntry).key)
			}
		}
		el := c.lru.order.PushFront(&lruEntry{key: r.ArticleID, value: r})
		c.lru.items[r.ArticleID] = el
	}

	if cap(sharedBuf) == 0 {
		sharedBuf = make([]*model.AnalysisResult, 0, c.lru.order.Len()+4)
	}
	sharedBuf = sharedBuf[:0]
	for e := c.lru.order.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*lruEntry)
		if ar, ok := entry.value.(*model.AnalysisResult); ok {
			sharedBuf = append(sharedBuf, ar)
		}
	}
}

func (c *ResultCache) Snapshot() []*model.AnalysisResult {
	c.lru.mu.Lock()
	defer c.lru.mu.Unlock()

	n := c.lru.order.Len()
	if cap(sharedBuf) < n {
		sharedBuf = make([]*model.AnalysisResult, 0, n)
	}
	sharedBuf = sharedBuf[:0]
	for e := c.lru.order.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*lruEntry)
		if ar, ok := entry.value.(*model.AnalysisResult); ok {
			sharedBuf = append(sharedBuf, ar)
		}
	}
	out := sharedBuf[:len(sharedBuf)]
	return out
}

func (c *ResultCache) RefreshEntry(id string, mutator func(r *model.AnalysisResult)) {
	c.lru.mu.Lock()
	defer c.lru.mu.Unlock()
	el, ok := c.lru.items[id]
	if !ok {
		return
	}
	ar, ok := el.Value.(*lruEntry).value.(*model.AnalysisResult)
	if !ok || ar == nil {
		return
	}
	mutator(ar)
}

func (c *ResultCache) Keys() []string {
	c.lru.mu.Lock()
	defer c.lru.mu.Unlock()
	keys := make([]string, 0, c.lru.order.Len())
	for e := c.lru.order.Front(); e != nil; e = e.Next() {
		keys = append(keys, e.Value.(*lruEntry).key)
	}
	return keys
}

var _ = list.New
