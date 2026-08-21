package store

import (
	"context"

	"summarizer/internal/model"
)

func (s *MemoryStore) SaveResult(ctx context.Context, r *model.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.results[r.ArticleID]; !exists {
		s.resultOrder = append(s.resultOrder, r.ArticleID)
	}
	s.results[r.ArticleID] = r
	return nil
}

func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	// viewSlice 返回副本，避免改动污染 store 内部的 resultOrder；维持插入顺序。
	orderRef := viewSlice(s.resultOrder, start, end)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range orderRef {
		if r, ok := s.results[id]; ok {
			out = append(out, r)
		}
	}
	return out, total, nil
}
