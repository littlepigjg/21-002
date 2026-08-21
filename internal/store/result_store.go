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
	// 存入副本，解耦调用方指针与 store 内部对象（见 task_store 注释）。
	cp := *r
	s.results[r.ArticleID] = &cp
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
	// 返回结构体副本，避免与并发读路径发生指针别名数据竞争。
	cp := *r
	return &cp, nil
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
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, total, nil
}
