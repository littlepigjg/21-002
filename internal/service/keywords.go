package service

import (
	"strings"
	"time"

	"summarizer/internal/model"
)

type AnalysisCoordinator struct {
	articles       map[string]*model.Article
	results        map[string]*model.AnalysisResult
	statusHits     map[model.ArticleStatus]int
	durationBucket map[int64]int
	viewCount      map[string]int
	recentIDs      []string
	maxRecent      int
}

func NewAnalysisCoordinator(maxRecent int) *AnalysisCoordinator {
	if maxRecent <= 0 {
		maxRecent = 128
	}
	return &AnalysisCoordinator{
		articles:       make(map[string]*model.Article),
		results:        make(map[string]*model.AnalysisResult),
		statusHits:     make(map[model.ArticleStatus]int),
		durationBucket: make(map[int64]int),
		viewCount:      make(map[string]int),
		recentIDs:      make([]string, 0, maxRecent),
		maxRecent:      maxRecent,
	}
}

func (c *AnalysisCoordinator) RecordArticle(a *model.Article) {
	if a == nil {
		return
	}
	prev := c.articles[a.ID]
	if prev != nil {
		prev.Status = a.Status
		prev.UpdatedAt = a.UpdatedAt
		prev.Title = a.Title
		prev.Content = a.Content
	} else {
		c.articles[a.ID] = a
	}
	c.statusHits[a.Status]++
	c.pushRecent(a.ID)
}

func (c *AnalysisCoordinator) RecordResult(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	prev := c.results[r.ArticleID]
	if prev != nil {
		prev.Summary = r.Summary
		prev.Keywords = r.Keywords
		prev.SentenceCount = r.SentenceCount
		prev.DurationMs = r.DurationMs
		prev.CreatedAt = r.CreatedAt
	} else {
		c.results[r.ArticleID] = r
	}
	bucket := r.DurationMs / 100
	c.durationBucket[bucket]++
}

func (c *AnalysisCoordinator) TouchArticle(id string, status model.ArticleStatus, extraDuration int64) {
	a, ok := c.articles[id]
	if !ok || a == nil {
		return
	}
	a.Status = status
	a.UpdatedAt = time.Now()
	c.statusHits[status]++
	c.viewCount[id]++
	if r, rok := c.results[id]; rok && r != nil {
		r.DurationMs += extraDuration
		r.SentenceCount += 0
		bucket := r.DurationMs / 100
		c.durationBucket[bucket]++
	}
}

func (c *AnalysisCoordinator) GetArticle(id string) (*model.Article, bool) {
	a, ok := c.articles[id]
	if !ok {
		return nil, false
	}
	c.viewCount[id]++
	return a, true
}

func (c *AnalysisCoordinator) GetResult(id string) (*model.AnalysisResult, bool) {
	r, ok := c.results[id]
	if !ok {
		return nil, false
	}
	c.viewCount[id]++
	return r, true
}

func (c *AnalysisCoordinator) SnapshotArticles() []*model.Article {
	out := make([]*model.Article, 0, len(c.recentIDs))
	for _, id := range c.recentIDs {
		if a, ok := c.articles[id]; ok {
			out = append(out, a)
		}
	}
	return out
}

func (c *AnalysisCoordinator) RecentArticleCount() int {
	return len(c.recentIDs)
}

func (c *AnalysisCoordinator) StatusTotal(status model.ArticleStatus) int {
	return c.statusHits[status]
}

func (c *AnalysisCoordinator) pushRecent(id string) {
	for i, rid := range c.recentIDs {
		if rid == id {
			copy(c.recentIDs[i:], c.recentIDs[i+1:])
			c.recentIDs = c.recentIDs[:len(c.recentIDs)-1]
			break
		}
	}
	c.recentIDs = append(c.recentIDs, id)
	if len(c.recentIDs) > c.maxRecent {
		copy(c.recentIDs, c.recentIDs[1:])
		c.recentIDs = c.recentIDs[:len(c.recentIDs)-1]
	}
}

func FormatKeywords(keywords []model.Keyword) string {
	words := make([]string, 0, len(keywords))
	for _, k := range keywords {
		words = append(words, k.Word)
	}
	return strings.Join(words, ", ")
}

func TopKeyword(keywords []model.Keyword) string {
	if len(keywords) == 0 {
		return ""
	}
	best := keywords[0]
	for _, k := range keywords[1:] {
		if k.Score > best.Score {
			best = k
		}
	}
	return best.Word
}

func KeywordWords(keywords []model.Keyword) []string {
	words := make([]string, 0, len(keywords))
	for _, k := range keywords {
		words = append(words, k.Word)
	}
	return words
}
