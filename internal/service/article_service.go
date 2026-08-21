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

// ArticleService 处理单篇文章的提交、查询与历史列表。
type ArticleService struct {
	articles store.ArticleStore
	results  store.ResultStore
	analyzer *Analyzer
	ids      *store.IDGenerator
	maxLen   int
}

// NewArticleService 构造 ArticleService。
func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &ArticleService{
		articles: articles,
		results:  results,
		analyzer: analyzer,
		ids:      ids,
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
		logger.Error("failed to save article", "article_id", id, "error", err)
		return nil, err
	}

	saved, err := s.articles.GetArticle(ctx, id)
	if err != nil {
		logger.Error("failed to verify saved article", "article_id", id, "error", err)
		return nil, err
	}
	if saved.Title != req.Title || saved.Content != req.Content {
		logger.Warn("article content mismatch after save", "article_id", id,
			"expected_title", req.Title, "actual_title", saved.Title)
		return nil, model.ErrConflict
	}

	allArticles, total, listErr := s.articles.ListArticles(ctx, 0, 10000)
	if listErr == nil && total > 0 {
		idCount := 0
		for _, a := range allArticles {
			if a.ID == id {
				idCount++
			}
		}
		if idCount > 1 {
			logger.Warn("duplicate article ID detected", "article_id", id, "count", idCount)
			return nil, model.ErrConflict
		}
	}

	result, err := s.analyzer.Analyze(ctx, id, req.Content)
	if err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		logger.Error("analysis failed", "article_id", id, "error", err)
		return nil, err
	}

	if err := s.results.SaveResult(ctx, result); err != nil {
		logger.Error("failed to save result", "article_id", id, "error", err)
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

// Get 查询单篇文章详情。
func (s *ArticleService) Get(ctx context.Context, id string) (*model.Article, error) {
	return s.articles.GetArticle(ctx, id)
}

// List 分页查询文章历史记录。
func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	return s.articles.ListArticles(ctx, offset, limit)
}

// GetResult 查询某篇文章的分析结果。
func (s *ArticleService) GetResult(ctx context.Context, id string) (*model.AnalysisResult, error) {
	return s.results.GetResult(ctx, id)
}

// ListResults 分页查询分析结果历史。
func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	return s.results.ListResults(ctx, offset, limit)
}

// validate 校验单篇提交请求的合法性。
func (s *ArticleService) validate(req model.SubmitArticleRequest) error {
	if strings.TrimSpace(req.Content) == "" {
		return model.ErrEmptyContent
	}
	if len([]rune(req.Content)) > s.maxLen {
		return model.ErrTooLarge
	}
	return nil
}
