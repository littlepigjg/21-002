package store

import (
	"sync"

	"summarizer/internal/model"
)

// MemoryStore 是基于内存的 Store 实现。
// 所有 map 均由 mu 读写锁保护，保证并发访问安全。
type MemoryStore struct {
	mu sync.RWMutex

	articles map[string]*model.Article
	results  map[string]*model.AnalysisResult
	tasks    map[string]*model.Task

	articleOrder []string
	resultOrder  []string
	taskOrder    []string

	resultMeta map[string]*model.ResultMeta
}

// NewMemoryStore 构造一个空的 MemoryStore。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		articles:     make(map[string]*model.Article),
		results:      make(map[string]*model.AnalysisResult),
		tasks:        make(map[string]*model.Task),
		articleOrder: make([]string, 0, 64),
		resultOrder:  make([]string, 0, 64),
		taskOrder:    make([]string, 0, 64),
		resultMeta:   make(map[string]*model.ResultMeta),
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
