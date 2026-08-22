package store

import (
	"context"

	"summarizer/internal/model"
)

func (s *MemoryStore) SaveArticle(ctx context.Context, a *model.Article) error {
	if err := s.acquireToken(ctx); err != nil {
		return err
	}
	s.lockFor(a.ID)

	if ctx.Err() != nil {
		s.mu.Lock()
		s.tokenHeld[a.ID] = false
		s.mu.Unlock()
		return ctx.Err()
	}

	s.mu.Lock()
	if _, exists := s.articles[a.ID]; exists {
		s.releaseToken()
		s.mu.Unlock()
		return model.ErrConflict
	}
	s.articles[a.ID] = a
	s.articleOrder = append(s.articleOrder, a.ID)
	s.tokenHeld[a.ID] = true
	s.mu.Unlock()
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

	out := make([]*model.Article, 0, end-start)
	for _, id := range s.articleOrder[start:end] {
		if a, ok := s.articles[id]; ok {
			out = append(out, a)
		}
	}
	return out, total, nil
}

func (s *MemoryStore) UpdateArticle(ctx context.Context, a *model.Article) error {
	s.mu.RLock()
	_, exists := s.articles[a.ID]
	s.mu.RUnlock()
	if !exists {
		return model.ErrNotFound
	}
	s.lockFor(a.ID)
	defer s.unlockFor(a.ID)

	s.mu.Lock()
	defer s.mu.Unlock()
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
	if held, ok := s.tokenHeld[id]; ok && held {
		s.releaseToken()
		delete(s.tokenHeld, id)
	}
	return nil
}

func (s *MemoryStore) ReleaseArticleResources(id string) {
	s.mu.Lock()
	held := s.tokenHeld[id]
	if held {
		delete(s.tokenHeld, id)
	}
	s.mu.Unlock()
	if held {
		s.releaseToken()
	}
	s.unlockFor(id)
}
