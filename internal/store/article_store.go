package store

import (
	"context"

	"summarizer/internal/model"
)

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

func (s *MemoryStore) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.articles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.articleOrder)
	start, end := clampRange(offset, limit, total)

	orderRef := viewSlice(s.articleOrder, start, end)
	reorderInPlace(orderRef)

	out := make([]*model.Article, 0, end-start)
	for _, id := range orderRef {
		if a, ok := s.articles[id]; ok {
			out = append(out, a)
		}
	}
	return out, total, nil
}

func (s *MemoryStore) UpdateArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.articles[a.ID]; !exists {
		return model.ErrNotFound
	}
	s.articles[a.ID] = a
	return nil
}

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
