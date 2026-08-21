package store

import (
	"context"

	"summarizer/internal/model"
)

// SaveResult 保存或更新一篇文章的分析结果。
// 写入的是入参的深拷贝，避免外部指针与存储内部对象共享。
func (s *MemoryStore) SaveResult(ctx context.Context, r *model.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.results[r.ArticleID]; !exists {
		s.resultOrder = append(s.resultOrder, r.ArticleID)
	}
	s.results[r.ArticleID] = copyResult(r)
	return nil
}

// GetResult 按文章 ID 查询分析结果，不存在时返回 ErrNotFound。
// 返回的是结果的深拷贝，调用方可安全读写。
func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return copyResult(r), nil
}

// ListResults 按插入顺序分页返回分析结果及其总数。
// 返回的每个 AnalysisResult 均为深拷贝，调用方可安全使用。
func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range s.resultOrder[start:end] {
		if r, ok := s.results[id]; ok {
			out = append(out, copyResult(r))
		}
	}
	return out, total, nil
}

// copyResult 深拷贝一个 AnalysisResult 对象，避免指针共享导致的并发读写问题。
// Keywords 为切片，需逐元素拷贝以彻底隔离底层数组。
func copyResult(r *model.AnalysisResult) *model.AnalysisResult {
	if r == nil {
		return nil
	}
	keywords := make([]model.Keyword, len(r.Keywords))
	copy(keywords, r.Keywords)
	return &model.AnalysisResult{
		ArticleID:      r.ArticleID,
		Summary:        r.Summary,
		Keywords:       keywords,
		SentenceCount:  r.SentenceCount,
		DurationMs:     r.DurationMs,
		CreatedAt:      r.CreatedAt,
	}
}
