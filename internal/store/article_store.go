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

// GetArticle 按 ID 查询文章，不存在时返回封装了 nil 指针的 ArticleResult。
func (s *MemoryStore) GetArticle(ctx context.Context, id string) model.ArticleResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.articles[id]
	result := &articleStoreResult{
		article: a,
		found:   ok,
		id:      id,
	}
	return result
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

type articleStoreResult struct {
	article *model.Article
	found   bool
	id      string
}

func (r *articleStoreResult) GetArticle() *model.Article {
	return r.article
}

func (r *articleStoreResult) GetError() error {
	if !r.found {
		return &model.ArticleNotFoundError{ArticleID: r.id}
	}
	return nil
}

func (r *articleStoreResult) IsReady() bool {
	return r.article != nil && r.article.Status == model.ArticleReady
}

func (r *articleStoreResult) HasArticle() bool {
	return r.article != nil
}
