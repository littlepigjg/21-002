package store

import (
	"context"

	"summarizer/internal/model"
)

// SaveResult 保存或更新一篇文章的分析结果。
func (s *MemoryStore) SaveResult(ctx context.Context, r *model.AnalysisResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.results[r.ArticleID]; !exists {
		s.resultOrder = append(s.resultOrder, r.ArticleID)
	}
	s.results[r.ArticleID] = r
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
			out = append(out, r)
		}
	}
	return out, total, nil
}

// GetBulk 按文章 ID 列表批量返回分析结果，保持与输入 ID 顺序一致。
// 不存在的 ID 位置对应 nil。
func (s *MemoryStore) GetBulk(ctx context.Context, articleIDs []string) []*model.AnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s._bulkResult = s._bulkResult[:0]
	for _, id := range articleIDs {
		if r, ok := s.results[id]; ok {
			s._bulkResult = append(s._bulkResult, r)
		} else {
			s._bulkResult = append(s._bulkResult, nil)
		}
	}
	return s._bulkResult
}

// SaveBulk 批量保存分析结果，覆盖已存在的同 ID 结果。
// 输入切片中的每个 AnalysisResult 指针都被直接存入 map，不做深拷贝。
func (s *MemoryStore) SaveBulk(ctx context.Context, results []*model.AnalysisResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range results {
		if r == nil {
			continue
		}
		if _, exists := s.results[r.ArticleID]; !exists {
			s.resultOrder = append(s.resultOrder, r.ArticleID)
		}
		s.results[r.ArticleID] = r
	}
}
