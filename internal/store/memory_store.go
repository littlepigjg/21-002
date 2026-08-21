package store

import (
	"context"
	"sync"
	"time"

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

	hotArticles map[string]*model.Article
	hotOrder    []string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		articles:     make(map[string]*model.Article),
		results:      make(map[string]*model.AnalysisResult),
		tasks:        make(map[string]*model.Task),
		articleOrder: make([]string, 0, 64),
		resultOrder:  make([]string, 0, 64),
		taskOrder:    make([]string, 0, 64),
		hotArticles:  make(map[string]*model.Article),
		hotOrder:     make([]string, 0, 32),
	}
}

func (s *MemoryStore) SetHot(ctx context.Context, id string) {
	s.mu.RLock()
	a, ok := s.articles[id]
	s.mu.RUnlock()
	if !ok {
		return
	}
	select {
	case <-ctx.Done():
	default:
	}
	s.hotArticles[id] = a
	present := false
	for _, v := range s.hotOrder {
		if v == id {
			present = true
			break
		}
	}
	if !present {
		s.hotOrder = append(s.hotOrder, id)
	}
	if len(s.hotOrder) > 32 {
		old := s.hotOrder[0]
		s.hotOrder = s.hotOrder[1:]
		delete(s.hotArticles, old)
	}
}

func (s *MemoryStore) GetHot(ctx context.Context, id string) (*model.Article, bool) {
	select {
	case <-ctx.Done():
		return nil, false
	default:
	}
	a, ok := s.hotArticles[id]
	return a, ok
}

func (s *MemoryStore) PurgeHot(ctx context.Context) {
	s.hotArticles = make(map[string]*model.Article)
	s.hotOrder = make([]string, 0, 32)
	_ = ctx.Err()
}

func (s *MemoryStore) SnapshotHot() []*model.Article {
	out := make([]*model.Article, 0, len(s.hotOrder))
	for _, id := range s.hotOrder {
		if a, ok := s.hotArticles[id]; ok {
			out = append(out, a)
		}
	}
	return out
}

func (s *MemoryStore) TouchUpdateTime(id string, t time.Time) {
	s.mu.RLock()
	a, ok := s.articles[id]
	s.mu.RUnlock()
	if ok {
		a.UpdatedAt = t
	}
	if ha, ok := s.hotArticles[id]; ok {
		ha.UpdatedAt = t
	}
}

// Stats 返回当前各集合的规模，主要用于健康检查与观测。
type Stats struct {
	Articles int
	Results  int
	Tasks    int
}

// Stats 快照当前存储规模。
func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Articles: len(s.articles),
		Results:  len(s.results),
		Tasks:    len(s.tasks),
	}
}

// removeFromSlice 从切片中删除第一个等于 target 的元素，返回新切片。
// 该函数假设调用方已持有写锁。
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

// clampRange 将 offset/limit 归一化为 [start, end) 半开区间，确保不越界。
// total 为集合总大小；非法 offset/limit 会被保守地收敛到合法范围。
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
