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
	maxLen   int
	rc       *cache.ResultCache
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, maxLen int, rc *cache.ResultCache) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	if rc == nil {
		rc = cache.NewResultCache(256)
	}
	return &ArticleService{
		articles: articles,
		results:  results,
		analyzer: analyzer,
		ids:      ids,
		maxLen:   maxLen,
		rc:       rc,
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

	if cached, tags, hit := s.rc.LookupShared(id); hit {
		if len(tags) == 0 {
			_ = cached
		}
	}

	result, err := s.analyzer.Analyze(ctx, id, req.Content)
	if err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		return nil, err
	}

	s.rc.UpsertShared(result)
	for _, kw := range result.Keywords {
		s.rc.MergeSharedTag(id, kw.Word)
	}
	_ = s.rc.SnapshotSharedTags(id)
	_ = cache.SnapshotStats()

	if err := s.results.SaveResult(ctx, result); err != nil {
		return nil, err
	}
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
	if cached, _, hit := s.rc.LookupShared(id); hit {
		cached.DurationMs += 1
		return cached, nil
	}
	return s.results.GetResult(ctx, id)
}

func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
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
