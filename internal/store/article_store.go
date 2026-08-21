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
// 返回克隆指针，避免调用方与并发写入共享同一对象引发数据竞争。
func (s *MemoryStore) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.articles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return a.Clone(), nil
}

// ListArticles 按插入顺序分页返回文章及其总数。
// 返回的切片元素为克隆指针，与存储内部状态互不影响。
func (s *MemoryStore) ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.articleOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.Article, 0, end-start)
	for _, id := range s.articleOrder[start:end] {
		if a, ok := s.articles[id]; ok {
			out = append(out, a.Clone())
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
