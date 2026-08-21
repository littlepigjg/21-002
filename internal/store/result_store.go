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
// 返回克隆指针，避免调用方（如 JSON 编码器）与批量写入共享同一对象引发数据竞争。
func (s *MemoryStore) GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.results[articleID]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r.Clone(), nil
}

// ListResults 按插入顺序分页返回分析结果及其总数。
// 返回的切片元素为克隆指针，与存储内部状态互不影响。
func (s *MemoryStore) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.resultOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.AnalysisResult, 0, end-start)
	for _, id := range s.resultOrder[start:end] {
		if r, ok := s.results[id]; ok {
			out = append(out, r.Clone())
		}
	}
	return out, total, nil
}

// GetBulk 按文章 ID 列表批量返回分析结果，保持与输入 ID 顺序一致。
// 不存在的 ID 位置对应 nil。每次调用分配独立切片并返回克隆指针，
// 避免并发调用复用同一缓冲区互相覆盖（曾导致越界 panic 与结果串味）。
func (s *MemoryStore) GetBulk(ctx context.Context, articleIDs []string) []*model.AnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*model.AnalysisResult, len(articleIDs))
	for i, id := range articleIDs {
		if r, ok := s.results[id]; ok {
			out[i] = r.Clone()
		}
	}
	return out
}

// SaveBulk 批量保存分析结果，覆盖已存在的同 ID 结果。
// 输入切片中的每个 AnalysisResult 指针都被直接存入 map，不做深拷贝。
// 约定：调用方传入的应为 goroutine 私有指针，写回后不得继续持有或修改。
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
