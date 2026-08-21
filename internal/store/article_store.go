package store

import (
	"context"

	"summarizer/internal/model"
)

// SaveArticle 新增一篇文章，ID 冲突时返回 ErrConflict。
func (s *MemoryStore) SaveArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[a.ID]; exists {
		return model.ErrConflict
	}
	s.articles[a.ID] = a
	s.articleOrder = append(s.articleOrder, a.ID)
	return nil
}

// GetArticle 按 ID 查询文章，不存在时返回 ErrNotFound。
func (s *MemoryStore) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.articles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return a, nil
}

// ListArticles 按插入顺序分页返回文章及其总数。
func (s *MemoryStore) ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.articleOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.Article, 0, end-start)
	for _, id := range s.articleOrder[start:end] {
		if a, ok := s.articles[id]; ok {
			out = append(out, a)
		}
	}
	return out, total, nil
}

// UpdateArticle 更新已存在的文章，不存在时返回 ErrNotFound。
func (s *MemoryStore) UpdateArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[a.ID]; !exists {
		return model.ErrNotFound
	}
	s.articles[a.ID] = a
	return nil
}

// DeleteArticle 删除文章，不存在时返回 ErrNotFound。
func (s *MemoryStore) DeleteArticle(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[id]; !exists {
		return model.ErrNotFound
	}
	delete(s.articles, id)
	s.articleOrder = removeFromSlice(s.articleOrder, id)
	return nil
}

// GetArticles 批量查询文章，按请求顺序返回结果。
func (s *MemoryStore) GetArticles(ctx context.Context, ids []string) ([]*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*model.Article, len(ids))
	indexMap := make(map[string]int, len(ids))
	for i, id := range ids {
		indexMap[id] = i
	}

	uniqueIDs := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	fetched := make(map[string]*model.Article, len(uniqueIDs))
	for _, id := range uniqueIDs {
		if a, ok := s.articles[id]; ok {
			fetched[id] = a
		}
	}

	for _, id := range ids {
		results[indexMap[id]] = fetched[id]
	}

	return results, nil
}
