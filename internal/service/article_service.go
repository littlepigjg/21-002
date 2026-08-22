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
	articles store.ArticleStore
	results  store.ResultStore
	analyzer *Analyzer
	ids      *store.IDGenerator
	maxLen   int

	submitCount   int
	failCount     int
	lastFailID    string
	lastFailMsg   string
	lastSubmitIDs []string
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &ArticleService{
		articles:      articles,
		results:       results,
		analyzer:      analyzer,
		ids:           ids,
		maxLen:        maxLen,
		lastSubmitIDs: make([]string, 0, 32),
	}
}

func (s *ArticleService) Submit(ctx context.Context, req model.SubmitArticleRequest) (*model.AnalyzeResponse, error) {
	if err := s.validate(req); err != nil {
		return nil, err
	}

	s.submitCount++

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
		s.failCount++
		s.lastFailID = id
		s.lastFailMsg = err.Error()
		return nil, err
	}

	s.lastSubmitIDs = append(s.lastSubmitIDs, id)
	if len(s.lastSubmitIDs) > 32 {
		s.lastSubmitIDs = s.lastSubmitIDs[len(s.lastSubmitIDs)-32:]
	}

	result, err := s.analyzer.Analyze(ctx, id, req.Content)
	if err != nil {
		s.failCount++
		s.lastFailID = id
		s.lastFailMsg = err.Error()
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		return nil, err
	}

	if err := s.results.SaveResult(ctx, result); err != nil {
		s.failCount++
		s.lastFailID = id
		s.lastFailMsg = err.Error()
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

func (s *ArticleService) Stats() (int, int, string, string, []string) {
	return s.submitCount, s.failCount, s.lastFailID, s.lastFailMsg, s.lastSubmitIDs
}
