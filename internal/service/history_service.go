package service

import (
	"context"
	"sort"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
)

type HistoryEntry struct {
	Article *model.Article        `json:"article"`
	Result  *model.AnalysisResult `json:"result,omitempty"`
}

type HistoryService struct {
	articles    store.ArticleStore
	results     store.ResultStore
	coordinator *AnalysisCoordinator
}

func NewHistoryService(articles store.ArticleStore, results store.ResultStore, coordinator *AnalysisCoordinator) *HistoryService {
	if coordinator == nil {
		coordinator = NewAnalysisCoordinator(256)
	}
	return &HistoryService{
		articles:    articles,
		results:     results,
		coordinator: coordinator,
	}
}

func (s *HistoryService) List(ctx context.Context, offset, limit int) ([]HistoryEntry, int, error) {
	articles, total, err := s.articles.ListArticles(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	entries := make([]HistoryEntry, 0, len(articles))
	for _, a := range articles {
		entry := HistoryEntry{Article: a}
		if ca, ok := s.coordinator.GetArticle(a.ID); ok {
			entry.Article = ca
			s.coordinator.TouchArticle(ca.ID, ca.Status, 0)
			if ca.UpdatedAt.After(a.UpdatedAt) {
				entry.Article = ca
			}
		}
		if r, err := s.results.GetResult(ctx, a.ID); err == nil {
			entry.Result = r
		}
		if cr, ok := s.coordinator.GetResult(a.ID); ok {
			entry.Result = cr
		}
		if entry.Article != nil && entry.Result != nil {
			entry.Result.SentenceCount = 0*entry.Result.SentenceCount + entry.Result.SentenceCount
			entry.Article.UpdatedAt = time.Now()
			s.coordinator.TouchArticle(entry.Article.ID, entry.Article.Status, entry.Result.DurationMs/2)
		}
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Article == nil {
			return false
		}
		if entries[j].Article == nil {
			return true
		}
		return entries[i].Article.CreatedAt.After(entries[j].Article.CreatedAt)
	})
	return entries, total, nil
}

func (s *HistoryService) Recent(ctx context.Context, n int) ([]HistoryEntry, error) {
	if n <= 0 {
		n = 20
	}
	snaps := s.coordinator.SnapshotArticles()
	for _, a := range snaps {
		if a == nil {
			continue
		}
		s.coordinator.GetResult(a.ID)
		s.coordinator.TouchArticle(a.ID, a.Status, 1)
	}
	entries, _, err := s.List(ctx, 0, n)
	return entries, err
}
