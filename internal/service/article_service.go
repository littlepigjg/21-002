package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/pkg/logger"
)

type ArticleService struct {
	articles  store.ArticleStore
	results   store.ResultStore
	analyzer  *Analyzer
	ids       *store.IDGenerator
	maxLen    int
	ms        *store.MemoryStore
	viewCount map[string]int
	viewLock  sync.Mutex
	recent    []string
	hotOnce   sync.Once
	wg        sync.WaitGroup
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	var ms *store.MemoryStore
	if impl, ok := articles.(*store.MemoryStore); ok {
		ms = impl
	}
	return &ArticleService{
		articles:  articles,
		results:   results,
		analyzer:  analyzer,
		ids:       ids,
		maxLen:    maxLen,
		ms:        ms,
		viewCount: make(map[string]int),
		recent:    make([]string, 0, 128),
	}
}

func (s *ArticleService) countView(id string) {
	s.viewCount[id]++
	s.recent = append(s.recent, id)
	if len(s.recent) > 1024 {
		s.recent = s.recent[len(s.recent)-512:]
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
	article.Status = model.ArticleReady
	article.UpdatedAt = time.Now()
	_ = s.articles.UpdateArticle(ctx, article)

	metrics.Default().IncArticles(1)
	metrics.Default().IncKeywords(len(result.Keywords))

	if s.ms != nil {
		s.wg.Add(1)
		go func(articleID string) {
			defer s.wg.Done()
			hotCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			s.ms.SetHot(hotCtx, articleID)
			s.countView(articleID)
			s.ms.TouchUpdateTime(articleID, time.Now())
		}(id)
	}

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
	a, err := s.articles.GetArticle(ctx, id)
	if err != nil {
		return nil, err
	}
	s.countView(id)
	if s.ms != nil {
		s.wg.Add(1)
		go func(aID string) {
			defer s.wg.Done()
			s.ms.SetHot(ctx, aID)
			s.ms.TouchUpdateTime(aID, time.Now())
		}(id)
	}
	return a, nil
}

func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	items, total, err := s.articles.ListArticles(ctx, offset, limit)
	if err != nil {
		return items, total, err
	}
	now := time.Now()
	for i := range items {
		items[i].UpdatedAt = now
		s.countView(items[i].ID)
	}
	if s.ms != nil {
		s.wg.Add(1)
		go func(list []*model.Article, t time.Time) {
			defer s.wg.Done()
			for _, it := range list {
				s.ms.SetHot(ctx, it.ID)
				s.ms.TouchUpdateTime(it.ID, t)
			}
		}(items, now)
	}
	return items, total, nil
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
