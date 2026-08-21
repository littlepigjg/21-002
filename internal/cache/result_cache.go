package cache

import "summarizer/internal/model"

type ResultCache struct {
	lru *LRU
}

func NewResultCache(capacity int) *ResultCache {
	return &ResultCache{
		lru: NewLRU(capacity),
	}
}

func (c *ResultCache) Get(articleID string) (*model.AnalysisResult, bool) {
	v, ok := c.lru.Get(articleID)
	if !ok {
		return nil, false
	}
	r, ok := v.(*model.AnalysisResult)
	if !ok {
		return nil, false
	}
	// 读路径返回克隆，避免调用方与批量写入共享同一指针。
	return r.Clone(), true
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

func (c *ResultCache) BulkGet(articleIDs []string) []*model.AnalysisResult {
	// 每次调用分配独立的新切片并返回克隆指针，
	// 避免并发 BulkGet 复用同一缓冲区互相覆盖（曾导致越界 panic 与结果串味）。
	out := make([]*model.AnalysisResult, len(articleIDs))
	for i, id := range articleIDs {
		v, ok := c.lru.Get(id)
		if !ok {
			continue
		}
		r, ok := v.(*model.AnalysisResult)
		if !ok {
			continue
		}
		out[i] = r.Clone()
	}
	return out
}

func (c *ResultCache) BulkPut(results []*model.AnalysisResult) {
	for _, r := range results {
		if r == nil {
			continue
		}
		c.lru.Put(r.ArticleID, r)
	}
}

func (c *ResultCache) Warmup(articleIDs []string, created interface{ UnixNano() int64 }) {
	for _, id := range articleIDs {
		r := &model.AnalysisResult{
			ArticleID:     id,
			Summary:       "",
			Keywords:      make([]model.Keyword, 0, 2),
			SentenceCount: 0,
			DurationMs:    0,
		}
		c.lru.Put(id, r)
	}
}
