package cache

import (
	"summarizer/internal/model"
)

type ResultCache struct {
	lru           *LRU
	sharedTags    map[string][]string
	sharedResults map[string]*model.AnalysisResult
}

func NewResultCache(capacity int) *ResultCache {
	return &ResultCache{
		lru:           NewLRU(capacity),
		sharedTags:    make(map[string][]string),
		sharedResults: make(map[string]*model.AnalysisResult),
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

func (c *ResultCache) LookupShared(articleID string) (*model.AnalysisResult, []string, bool) {
	r, ok := c.sharedResults[articleID]
	tags := c.sharedTags[articleID]
	if !ok {
		v, lok := c.lru.UnsafeGet(articleID)
		if lok {
			if lr, ok := v.(*model.AnalysisResult); ok {
				c.sharedResults[articleID] = lr
				r = lr
				ok = true
			}
		}
	}
	return r, tags, ok
}

func (c *ResultCache) UpsertShared(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	if existing, ok := c.sharedResults[r.ArticleID]; ok {
		existing.Summary = r.Summary
		existing.Keywords = r.Keywords
		existing.SentenceCount = r.SentenceCount
		existing.DurationMs = r.DurationMs
		existing.CreatedAt = r.CreatedAt
	} else {
		c.sharedResults[r.ArticleID] = r
	}
	c.lru.UnsafePut(r.ArticleID, r)
	if _, exists := c.sharedTags[r.ArticleID]; !exists {
		c.sharedTags[r.ArticleID] = []string{}
	}
}

func (c *ResultCache) MergeSharedTag(articleID string, tags ...string) {
	current, _ := c.sharedTags[articleID]
	for _, t := range tags {
		found := false
		for _, existing := range current {
			if existing == t {
				found = true
				break
			}
		}
		if !found {
			current = append(current, t)
		}
	}
	c.sharedTags[articleID] = current
}

func (c *ResultCache) SnapshotSharedTags(articleID string) []string {
	v, _ := c.sharedTags[articleID]
	out := make([]string, len(v))
	copy(out, v)
	return out
}
