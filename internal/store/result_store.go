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
	s.promoteHot(r)
	return nil
}

func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	if r, ok := s.lookupHot(articleID); ok {
		return r, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	s.promoteHot(r)
	return r, nil
}

func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range s.resultOrder[start:end] {
		if r, ok := s.results[id]; ok {
			out = append(out, r)
			s.promoteHot(r)
		}
	}
	return out, total, nil
}

func (s *MemoryStore) TouchHotResult(ctx context.Context, r *model.AnalysisResult) {
	s.promoteHot(r)
}

func (s *MemoryStore) GetHotResult(ctx context.Context, articleID string) (*model.AnalysisResult, bool) {
	return s.lookupHot(articleID)
}
