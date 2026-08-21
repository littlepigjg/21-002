package cache

import "summarizer/internal/model"

// ResultCache 是基于 LRU 的分析结果缓存，按文章 ID 索引。
type ResultCache struct {
	lru *LRU
}

// NewResultCache 构造指定容量的分析结果缓存。
func NewResultCache(capacity int) *ResultCache {
	return &ResultCache{lru: NewLRU(capacity)}
}

// Get 按文章 ID 查询缓存的分析结果。
func (c *ResultCache) Get(articleID string) (*model.AnalysisResult, bool) {
	v, ok := c.lru.Get(articleID)
	if !ok {
		return nil, false
	}
	r, ok := v.(*model.AnalysisResult)
	return r, ok
}

// Put 将分析结果写入缓存。
func (c *ResultCache) Put(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	c.lru.Put(r.ArticleID, r)
}

// Remove 删除指定文章 ID 的缓存项。
func (c *ResultCache) Remove(articleID string) {
	c.lru.Remove(articleID)
}

// Len 返回当前缓存项数量。
func (c *ResultCache) Len() int {
	return c.lru.Len()
}
