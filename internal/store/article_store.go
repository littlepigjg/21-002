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
	// 存入副本，解耦调用方指针与 store 内部对象（见 task_store 注释）。
	cp := *a
	s.articles[a.ID] = &cp
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
	// 返回结构体副本，避免调用方在锁外改字段时与并发读路径
	//（handler 轮询、JSON 序列化）发生指针别名数据竞争。
	cp := *a
	return &cp, nil
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
			cp := *a
			out = append(out, &cp)
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
	cp := *a
	s.articles[a.ID] = &cp
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
