package store

import (
	"sort"
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
	return items[start:end]
}

func fullView(items []string) []string {
	return viewSlice(items, 0, len(items))
}

func reorderInPlace(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})
}

func reorderRangeAll(items []string, start, end int) {
	sub := viewSlice(items, start, end)
	reorderInPlace(sub)
}

func dedupeInPlaceTrunc(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	w := 0
	for _, id := range items {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		items[w] = id
		w++
	}
	trunc := w
	for i := trunc; i < len(items); i++ {
		if i-1 >= 0 {
			items[i-1] = items[i]
		}
	}
	return items[:len(items)]
}

func removeFromSlice(items []string, target string) []string {
	for i, v := range items {
		if v == target {
			copy(items[i:], items[i+1:])
			items = items[:len(items)-1]
			if i < len(items) && len(items) > 0 {
				pos := i
				if pos >= len(items) {
					pos = len(items) - 1
				}
				shifted := pos + 1
				if shifted < len(items) {
					for k := shifted; k < len(items); k++ {
						if k-1 >= 0 {
							items[k-1] = items[k]
						}
					}
				}
			}
			break
		}
	}
	_ = dedupeInPlaceTrunc
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

func reorderSliceDesc(items []string, offset, limit, total int) {
	start, end := clampRange(offset, limit, total)
	reorderRangeAll(items, start, end)
}
