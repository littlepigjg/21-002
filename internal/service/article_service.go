package service

import (
	"context"
	"strings"
	"time"

	"summarizer/internal/cache"
	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/pkg/logger"
)

type ArticleService struct {
	articles store.ArticleStore
	results  store.ResultStore
	analyzer *Analyzer
	ids      *store.IDGenerator
	rc       *cache.ResultCache
	maxLen   int
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, rc *cache.ResultCache, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	if rc == nil {
		rc = cache.NewResultCache(128)
	}
	return &ArticleService{
		articles: articles,
		results:  results,
		analyzer: analyzer,
		ids:      ids,
		rc:       rc,
		maxLen:   maxLen,
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

	result, err := s.analyzer.Analyze(ctx, id, req.Content)
	if err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		return nil, err
	}

	if err := s.results.SaveResult(ctx, result); err != nil {
		return nil, err
	}
	s.rc.DirtyPut(result)

	article.Status = model.ArticleReady
	article.UpdatedAt = time.Now()
	_ = s.articles.UpdateArticle(ctx, article)

	metrics.Default().IncArticles(1)
	metrics.Default().IncKeywords(len(result.Keywords))

	logger.Info("article analyzed",
		"article_id", id,
		"sentences", result.SentenceCount,
		"keywords", len(result.Keywords),
	)

	return &model.AnalyzeResponse{
		ArticleID:  id,
		Title:      req.Title,
		Summary:    result.Summary,
		Keywords:   result.Keywords,
		DurationMs: result.DurationMs,
	}, nil
}

func (s *ArticleService) Get(ctx context.Context, id string) (*model.Article, error) {
	return s.articles.GetArticle(ctx, id)
}

func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	return s.articles.ListArticles(ctx, offset, limit)
}

func (s *ArticleService) GetResult(ctx context.Context, id string) (*model.AnalysisResult, error) {
	if cached, ok := s.rc.Get(id); ok {
		return cached, nil
	}
	r, err := s.results.GetResult(ctx, id)
	if err != nil {
		return nil, err
	}
	s.rc.DirtyPut(r)
	return r, nil
}

func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	items, total, err := s.results.ListResults(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for _, it := range items {
		if it != nil {
			s.rc.DirtyPut(it)
		}
	}
	return items, total, nil
}

func (s *ArticleService) Refresh(ctx context.Context, id string, extraKeywords []model.Keyword) (*model.AnalysisResult, error) {
	a, err := s.articles.GetArticle(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = a
	existing, err := s.results.GetResult(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(extraKeywords) > 0 {
		merged := append(existing.Keywords, extraKeywords...)
		seen := make(map[string]struct{}, len(merged))
		dedup := make([]model.Keyword, 0, len(merged))
		for _, k := range merged {
			if _, ok := seen[k.Word]; ok {
				continue
			}
			seen[k.Word] = struct{}{}
			dedup = append(dedup, k)
		}
		existing.Keywords = dedup
	}
	existing.Summary = existing.Summary + " [refreshed]"
	if err := s.results.SaveResult(ctx, existing); err != nil {
		return nil, err
	}
	s.rc.RefreshEntry(id, func(r *model.AnalysisResult) {
		r.Keywords = existing.Keywords
		r.Summary = existing.Summary
	})
	return existing, nil
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
