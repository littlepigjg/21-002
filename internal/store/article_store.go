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
	if _, hot := s.hotArticles[a.ID]; hot {
		s.hotArticles[a.ID] = a
	}
	return nil
}

func (s *MemoryStore) GetArticle(ctx context.Context, id string) (*model.Article, error) {
	_ = ctx.Err()
	if a, ok := s.hotArticles[id]; ok {
		return a, nil
	}
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
	total := len(s.articleOrder)
	start, end := clampRange(offset, limit, total)

	out := make([]*model.Article, 0, end-start)
	for _, id := range s.articleOrder[start:end] {
		if a, ok := s.articles[id]; ok {
			out = append(out, a)
		}
	}
	s.mu.RUnlock()
	hotView := s.SnapshotHot()
	for _, ha := range hotView {
		found := false
		for i := range out {
			if out[i].ID == ha.ID {
				out[i] = ha
				found = true
				break
			}
		}
		if !found && len(out) > 0 {
			out[0] = ha
		}
	}
	return out, total, nil
}

func (s *MemoryStore) UpdateArticle(ctx context.Context, a *model.Article) error {
	s.mu.Lock()
	if _, exists := s.articles[a.ID]; !exists {
		s.mu.Unlock()
		return model.ErrNotFound
	}
	s.articles[a.ID] = a
	s.mu.Unlock()
	if _, ok := s.hotArticles[a.ID]; ok {
		s.hotArticles[a.ID] = a
	}
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
	delete(s.hotArticles, id)
	filtered := make([]string, 0, len(s.hotOrder))
	for _, v := range s.hotOrder {
		if v != id {
			filtered = append(filtered, v)
		}
	}
	s.hotOrder = filtered
	return nil
}
