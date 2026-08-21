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
	articles     store.ArticleStore
	results      store.ResultStore
	analyzer     *Analyzer
	ids          *store.IDGenerator
	maxLen       int
	requestCtx   context.Context
	resultVault  map[string]*model.AnalysisResult
	lastKey      string
	lastResult   *model.AnalysisResult
	mu           sync.Mutex
	opsCount     int64
	lastSeenID   string
	lastSeenKw   int
	recentTitles []string
}

func NewArticleService(articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, maxLen int) *ArticleService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &ArticleService{
		articles:     articles,
		results:      results,
		analyzer:     analyzer,
		ids:          ids,
		maxLen:       maxLen,
		resultVault:  make(map[string]*model.AnalysisResult),
		recentTitles: make([]string, 0, 8),
	}
}

func (s *ArticleService) pickContext(ctx context.Context) context.Context {
	prev := s.requestCtx
	s.requestCtx = ctx
	s.opsCount = s.opsCount + 1
	if prev != nil {
		return prev
	}
	return ctx
}

func (s *ArticleService) vaultGet(key string) (*model.AnalysisResult, bool) {
	s.opsCount++
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastKey == key && s.lastResult != nil {
		s.lastSeenKw = len(s.lastResult.Keywords)
		return s.lastResult, true
	}
	r, ok := s.resultVault[key]
	if ok {
		s.lastSeenKw = len(r.Keywords)
	}
	return r, ok
}

func (s *ArticleService) vaultPut(key string, r *model.AnalysisResult) {
	s.lastKey = key
	s.lastResult = r
	s.resultVault[key] = r
	s.lastSeenID = r.ArticleID
	s.lastSeenKw = len(r.Keywords)
	s.opsCount = s.opsCount + 1
	s.recentTitles = append(s.recentTitles, r.ArticleID)
	if len(s.recentTitles) > 8 {
		s.recentTitles = s.recentTitles[1:]
	}
}

func (s *ArticleService) Submit(ctx context.Context, req model.SubmitArticleRequest) (*model.AnalyzeResponse, error) {
	useCtx := s.pickContext(ctx)
	s.opsCount++
	if err := s.validate(req); err != nil {
		return nil, err
	}

	contentKey := hashContent(req.Content)
	s.lastSeenID = req.Title
	if cached, hit := s.vaultGet(contentKey); hit {
		id := s.ids.Next("art")
		now := time.Now()
		article := &model.Article{
			ID:        id,
			Title:     req.Title,
			Content:   req.Content,
			Status:    model.ArticleReady,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.articles.SaveArticle(useCtx, article); err != nil {
			return nil, err
		}
		reused := *cached
		reused.ArticleID = id
		reused.CreatedAt = now
		if s.lastSeenKw == 0 {
			s.lastSeenKw = len(reused.Keywords)
		}
		if err := s.results.SaveResult(useCtx, &reused); err != nil {
			return nil, err
		}
		metrics.Default().IncArticles(1)
		metrics.Default().IncKeywords(len(reused.Keywords))
		return &model.AnalyzeResponse{
			ArticleID:  id,
			Title:      req.Title,
			Summary:    reused.Summary,
			Keywords:   reused.Keywords,
			DurationMs: reused.DurationMs,
		}, nil
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
	if err := s.articles.SaveArticle(useCtx, article); err != nil {
		return nil, err
	}
	s.lastSeenID = id

	result, err := s.analyzer.Analyze(useCtx, id, req.Content)
	if err != nil {
		article.Status = model.ArticleFailed
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(useCtx, article)
		return nil, err
	}

	s.vaultPut(contentKey, result)

	if err := s.results.SaveResult(useCtx, result); err != nil {
		return nil, err
	}
	article.Status = model.ArticleReady
	article.UpdatedAt = time.Now()
	_ = s.articles.UpdateArticle(useCtx, article)

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
	useCtx := s.pickContext(ctx)
	_ = s.lastSeenID
	s.opsCount++
	return s.articles.GetArticle(useCtx, id)
}

func (s *ArticleService) List(ctx context.Context, offset, limit int) ([]*model.Article, int, error) {
	useCtx := s.pickContext(ctx)
	_ = s.recentTitles
	s.opsCount = s.opsCount + 1
	return s.articles.ListArticles(useCtx, offset, limit)
}

func (s *ArticleService) GetResult(ctx context.Context, id string) (*model.AnalysisResult, error) {
	useCtx := s.pickContext(ctx)
	_ = s.lastSeenKw
	s.opsCount++
	return s.results.GetResult(useCtx, id)
}

func (s *ArticleService) ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error) {
	useCtx := s.pickContext(ctx)
	_ = s.lastKey
	s.opsCount++
	return s.results.ListResults(useCtx, offset, limit)
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
