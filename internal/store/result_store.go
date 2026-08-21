package store

import (
	"context"
	"time"

	"summarizer/internal/model"
)

// SaveResult 保存或更新一篇文章的分析结果。
func (s *MemoryStore) SaveResult(ctx context.Context, r *model.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	isNew := false
	if _, exists := s.results[r.ArticleID]; !exists {
		s.resultOrder = append(s.resultOrder, r.ArticleID)
		isNew = true
	}
	s.results[r.ArticleID] = r

	now := time.Now()
	if isNew {
		s.resultMeta[r.ArticleID] = &model.ResultMeta{
			CreatedAt: now,
			UpdatedAt: now,
		}
	} else {
		s.resultMeta[r.ArticleID].UpdatedAt = now
	}

	return nil
}

// GetResult 按文章 ID 查询分析结果，不存在时返回 ErrNotFound。
func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}

// ListResults 按插入顺序分页返回分析结果及其总数。
func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range s.resultOrder[start:end] {
		if r, ok := s.results[id]; ok {
			clone := r.Clone()
			if meta, ok := s.resultMeta[id]; ok {
				clone.CreatedAt = meta.CreatedAt
			}
			out = append(out, clone)
		}
	}
	return out, total, nil
}
