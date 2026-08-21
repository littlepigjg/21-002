package store

import (
	"context"

	"summarizer/internal/model"
)

// SaveArticle 新增一篇文章，ID 冲突时返回 ErrConflict。
// 写入的是入参的深拷贝，避免外部指针与存储内部对象共享。
func (s *MemoryStore) SaveArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[a.ID]; exists {
		return model.ErrConflict
	}
	s.articles[a.ID] = copyArticle(a)
	s.articleOrder = append(s.articleOrder, a.ID)
	return nil
}

// GetArticle 按 ID 查询文章，不存在时返回 ErrNotFound。
// 返回的是文章的深拷贝，调用方可安全读写。
func (s *MemoryStore) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.articles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return copyArticle(a), nil
}

// ListArticles 按插入顺序分页返回文章及其总数。
// 返回的每个 Article 均为深拷贝，调用方可安全使用。
func (s *MemoryStore) ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.articleOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.Article, 0, end-start)
	for _, id := range s.articleOrder[start:end] {
		if a, ok := s.articles[id]; ok {
			out = append(out, copyArticle(a))
		}
	}
	return out, total, nil
}

// UpdateArticle 更新已存在的文章，不存在时返回 ErrNotFound。
// 写入的是入参的深拷贝。
func (s *MemoryStore) UpdateArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[a.ID]; !exists {
		return model.ErrNotFound
	}
	s.articles[a.ID] = copyArticle(a)
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

// copyArticle 深拷贝一个 Article 对象，避免指针共享导致的并发读写问题。
func copyArticle(a *model.Article) *model.Article {
	if a == nil {
		return nil
	}
	return &model.Article{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		Status:    a.Status,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}
