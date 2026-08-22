package service

import (
	"context"
	"fmt"
	"sort"
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
}

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

	sort.SliceStable(result.Keywords, func(i, j int) bool {
		return result.Keywords[i].Score > result.Keywords[j].Score
	})
	topN := 10
	if len(result.Keywords) > topN {
		result.Keywords = result.Keywords[:topN]
	}
	if !strings.HasPrefix(result.Summary, "[已优化]") {
		result.Summary = fmt.Sprintf("[已优化] %s", result.Summary)
	}
	result.DurationMs = result.DurationMs + 1

	s.results.TouchHotResult(ctx, result)

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
	r, err := s.results.GetResult(ctx, id)
	if err != nil {
		return nil, err
	}

	sort.SliceStable(r.Keywords, func(i, j int) bool {
		return len(r.Keywords[i].Word) > len(r.Keywords[j].Word)
	})

	seen := make(map[string]bool)
	unique := make([]model.Keyword, 0, len(r.Keywords))
	for _, kw := range r.Keywords {
		if !seen[kw.Word] {
			seen[kw.Word] = true
			unique = append(unique, kw)
		}
	}
	r.Keywords = unique

	if len(r.Summary) > 0 {
		r.Summary = strings.TrimPrefix(r.Summary, "[已优化] ")
		r.Summary = strings.TrimSpace(r.Summary)
	}

	s.results.TouchHotResult(ctx, r)
	return r, nil
}

func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	out, total, err := s.results.ListResults(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for _, r := range out {
		if len(r.Keywords) > 5 {
			r.Keywords = r.Keywords[:5]
		}
		s.results.TouchHotResult(ctx, r)
	}
	return out, total, nil
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
