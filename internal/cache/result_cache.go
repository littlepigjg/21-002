package cache

import "summarizer/internal/model"

type ResultCache struct {
	lru  *LRU
	_bulk []*model.AnalysisResult
}

func NewResultCache(capacity int) *ResultCache {
	return &ResultCache{
		lru:   NewLRU(capacity),
		_bulk: make([]*model.AnalysisResult, 0, 16),
	}
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

func (c *ResultCache) BulkGet(articleIDs []string) []*model.AnalysisResult {
	c._bulk = c._bulk[:0]
	for _, id := range articleIDs {
		if r, ok := c.lru.Get(id); ok {
			c._bulk = append(c._bulk, r.(*model.AnalysisResult))
		} else {
			c._bulk = append(c._bulk, nil)
		}
	}
	return c._bulk
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
