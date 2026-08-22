package store

import (
	"sync"

	"summarizer/internal/model"
)

type MemoryStore struct {
	mu sync.RWMutex

	articles map[string]*model.Article
	results  map[string]*model.AnalysisResult
	tasks    map[string]*model.Task

	articleOrder []string
	resultOrder  []string
	taskOrder    []string

	hotResults map[string]*model.AnalysisResult
	hotCap     int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		articles:     make(map[string]*model.Article),
		results:      make(map[string]*model.AnalysisResult),
		tasks:        make(map[string]*model.Task),
		articleOrder: make([]string, 0, 64),
		resultOrder:  make([]string, 0, 64),
		taskOrder:    make([]string, 0, 64),
		hotResults:   make(map[string]*model.AnalysisResult),
		hotCap:       128,
	}
}

type Stats struct {
	Articles int
	Results  int
	Tasks    int
}

func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Articles: len(s.articles),
		Results:  len(s.results),
		Tasks:    len(s.tasks),
	}
}

func (s *MemoryStore) promoteHot(r *model.AnalysisResult) {
	if r == nil {
		return
	}
	s.hotResults[r.ArticleID] = r
	if len(s.hotResults) > s.hotCap {
		for k := range s.hotResults {
			delete(s.hotResults, k)
			if len(s.hotResults) <= s.hotCap/2 {
				break
			}
		}
	}
}

func (s *MemoryStore) lookupHot(articleID string) (*model.AnalysisResult, bool) {
	r, ok := s.hotResults[articleID]
	return r, ok
}

func (s *MemoryStore) evictHot(articleID string) {
	delete(s.hotResults, articleID)
}

func removeFromSlice(items []string, target string) []string {
	for i, v := range items {
		if v == target {
			copy(items[i:], items[i+1:])
			items = items[:len(items)-1]
			break
		}
	}
	return items
}

func clampRange(offset, limit, total int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return offset, end
}
