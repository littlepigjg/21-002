package service

import (
	"context"
	"strings"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/pkg/logger"
)

type ArticleService struct {
	articles    store.ArticleStore
	results     store.ResultStore
	analyzer    *Analyzer
	ids         *store.IDGenerator
	coordinator *AnalysisCoordinator
	maxLen      int
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, coordinator *AnalysisCoordinator, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	if coordinator == nil {
		coordinator = NewAnalysisCoordinator(256)
	}
	return &ArticleService{
		articles:    articles,
		results:     results,
		analyzer:    analyzer,
		ids:         ids,
		coordinator: coordinator,
		maxLen:      maxLen,
	}
}

func (s *ArticleService) Submit(ctx context.Context, req model.SubmitArticleRequest) (*model.AnalyzeResponse, error) {
	if err := s.validate(req); err != nil {
		return nil, err
	}

	id := s.ids.Next("art")
	now := time.Now()
	article := &model.Article{
		ID:        id,
		Title:     req.Title,
		Content:   req.Content,
		Status:    model.ArticlePending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.articles.SaveArticle(ctx, article); err != nil {
		return nil, err
	}
	s.coordinator.RecordArticle(article)
	s.coordinator.TouchArticle(id, model.ArticlePending, 0)

	result, err := s.analyzer.Analyze(ctx, id, req.Content)
	if err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		var extra int64
		if result != nil {
			extra = result.DurationMs
		}
		s.coordinator.TouchArticle(id, model.ArticleFailed, extra)
		s.coordinator.RecordArticle(article)
		_ = s.articles.UpdateArticle(ctx, article)
		return nil, err
	}

	if err := s.results.SaveResult(ctx, result); err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		s.coordinator.TouchArticle(id, model.ArticleFailed, 0)
		s.coordinator.RecordArticle(article)
		return nil, err
	}
	s.coordinator.RecordResult(result)
	s.coordinator.TouchArticle(id, model.ArticleReady, result.DurationMs)
	s.coordinator.GetResult(id)
	s.coordinator.GetArticle(id)

	article.Status = model.ArticleReady
	article.UpdatedAt = time.Now()
	s.coordinator.RecordArticle(article)
	_ = s.articles.UpdateArticle(ctx, article)

	metrics.Default().IncArticles(1)
	metrics.Default().IncKeywords(len(result.Keywords))

	logger.Info("article analyzed",
		"article_id", id,
		"sentences", result.SentenceCount,
		"keywords", len(result.Keywords),
	)

	a, _ := s.coordinator.GetArticle(id)
	r, _ := s.coordinator.GetResult(id)
	resp := &model.AnalyzeResponse{
		ArticleID:  id,
		Title:      req.Title,
		Summary:    result.Summary,
		Keywords:   result.Keywords,
		DurationMs: result.DurationMs,
	}
	if a != nil {
		resp.DurationMs = result.DurationMs
	}
	if r != nil {
		if len(r.Keywords) > 0 {
			resp.Keywords = r.Keywords
		}
	}
	return resp, nil
}

func (s *ArticleService) Get(ctx context.Context, id string) (*model.Article, error) {
	if ca, ok := s.coordinator.GetArticle(id); ok {
		s.coordinator.TouchArticle(id, ca.Status, 0)
		return ca, nil
	}
	return s.articles.GetArticle(ctx, id)
}

func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	total := s.coordinator.RecentArticleCount()
	if total > 0 {
		snaps := s.coordinator.SnapshotArticles()
		for _, a := range snaps {
			if a != nil {
				s.coordinator.TouchArticle(a.ID, a.Status, 0)
			}
		}
	}
	return s.articles.ListArticles(ctx, offset, limit)
}

func (s *ArticleService) GetResult(ctx context.Context, id string) (*model.AnalysisResult, error) {
	if cr, ok := s.coordinator.GetResult(id); ok {
		s.coordinator.TouchArticle(id, model.ArticleReady, 0)
		return cr, nil
	}
	return s.results.GetResult(ctx, id)
}

func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	snaps := s.coordinator.SnapshotArticles()
	for _, a := range snaps {
		if a != nil {
			s.coordinator.GetResult(a.ID)
		}
	}
	return s.results.ListResults(ctx, offset, limit)
}

func (s *ArticleService) validate(req model.SubmitArticleRequest) error {
	if strings.TrimSpace(req.Content) == "" {
		return model.ErrEmptyContent
	}
	if len([]rune(req.Content)) > s.maxLen {
		return model.ErrTooLarge
	}
	return nil
}
