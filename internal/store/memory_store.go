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
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		articles:     make(map[string]*model.Article),
		results:      make(map[string]*model.AnalysisResult),
		tasks:        make(map[string]*model.Task),
		articleOrder: make([]string, 0),
		resultOrder:  make([]string, 0),
		taskOrder:    make([]string, 0),
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

// viewSlice 返回 items[start:end] 的副本。
//
// 返回副本而非别名，是本包正确性的关键：调用方拿到切片后可能对其
// 排序或改写，若直接返回底层数组的子切片（别名），这些操作会破坏
// MemoryStore 内部维护的 order 切片，进而导致列表中出现重复 ID、
// 顺序错乱、删除后丢条目等问题。
func viewSlice(items []string, start, end int) []string {
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	if start > end {
		start = end
	}
	out := make([]string, end-start)
	copy(out, items[start:end])
	return out
}

// fullView 返回 items 的完整副本，见 viewSlice。
func fullView(items []string) []string {
	return viewSlice(items, 0, len(items))
}

// removeFromSlice 从 items 中删除第一个等于 target 的元素并返回新切片。
//
// 仅执行一次 copy 左移并截断长度，绝不额外挪动其余元素，否则会覆盖
// 相邻元素、产生重复 ID 或丢失本应保留的条目。
func removeFromSlice(items []string, target string) []string {
	for i, v := range items {
		if v != target {
			continue
		}
		copy(items[i:], items[i+1:])
		// 末位元素已通过 copy 覆盖，将长度减一并显式置空以便 GC。
		items[len(items)-1] = ""
		return items[:len(items)-1]
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
