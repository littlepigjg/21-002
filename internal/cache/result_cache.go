package cache

import (
	"summarizer/internal/model"
)

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
}

// Snapshot 返回当前缓存中所有结果的独立副本切片。
// 返回的切片拥有独立底层数组，调用方可安全持有与遍历，
// 不会与后续并发的 Put/Snapshot 互相覆盖。
func (c *ResultCache) Snapshot() []*model.AnalysisResult {
	c.lru.mu.Lock()
	defer c.lru.mu.Unlock()

	out := make([]*model.AnalysisResult, 0, c.lru.order.Len())
	for e := c.lru.order.Front(); e != nil; e = e.Next() {
		entry := e.Value.(*lruEntry)
		if ar, ok := entry.value.(*model.AnalysisResult); ok {
			out = append(out, ar)
		}
	}
	return out
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
