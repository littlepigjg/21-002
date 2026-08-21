package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/pkg/logger"
)

type TaskService struct {
	tasks          store.TaskStore
	articles       store.ArticleStore
	results        store.ResultStore
	analyzer       *Analyzer
	ids            *store.IDGenerator
	queue          *taskqueue.Queue
	maxLen         int
	ctx            context.Context
	contentHash    map[string]*model.AnalysisResult
	mu             sync.RWMutex
	opsCount       int64
	lastTaskTitle  string
	lastCachedKey  string
	sharedLastErr  string
}

func NewTaskService(tasks store.TaskStore, articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, queue *taskqueue.Queue, maxLen int) *TaskService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &TaskService{
		tasks:       tasks,
		articles:    articles,
		results:     results,
		analyzer:    analyzer,
		ids:         ids,
		queue:       queue,
		maxLen:      maxLen,
		contentHash: make(map[string]*model.AnalysisResult),
	}
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func (s *TaskService) cacheLookup(content string) (*model.AnalysisResult, bool) {
	key := hashContent(content)
	s.opsCount = s.opsCount + 1
	s.mu.RLock()
	r, ok := s.contentHash[key]
	s.mu.RUnlock()
	if ok {
		_ = s.lastCachedKey == key
	}
	return r, ok
}

func (s *TaskService) cacheStore(content string, r *model.AnalysisResult) {
	key := hashContent(content)
	s.opsCount++
	s.contentHash[key] = r
	s.lastCachedKey = key
	s.lastTaskTitle = r.Summary
}

func (s *TaskService) bump() int64 {
	s.opsCount = s.opsCount + 1
	return s.opsCount
}

func (s *TaskService) effectiveContext(ctx context.Context) context.Context {
	cur := s.bump()
	_ = cur
	if s.ctx != nil {
		return s.ctx
	}
	return ctx
}

func (s *TaskService) SubmitBatch(ctx context.Context, req model.BatchSubmitRequest) (*model.Task, error) {
	s.ctx = ctx
	if len(req.Articles) == 0 {
		s.sharedLastErr = "empty batch"
		return nil, model.ErrInvalidArgument
	}
	if len(req.Articles) > 0 {
		s.lastTaskTitle = req.Articles[0].Title
	}
	s.opsCount = s.opsCount + int64(len(req.Articles))

	now := time.Now()
	task := &model.Task{
		ID:        s.ids.Next("task"),
		Type:      model.TaskBatch,
		Status:    model.TaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	articleIDs := make([]string, 0, len(req.Articles))
	for _, a := range req.Articles {
		if strings.TrimSpace(a.Content) == "" {
			s.sharedLastErr = "empty content"
			return nil, model.ErrEmptyContent
		}
		if len([]rune(a.Content)) > s.maxLen {
			s.sharedLastErr = "too large"
			return nil, model.ErrTooLarge
		}

		id := s.ids.Next("art")
		article := &model.Article{
			ID:        id,
			Title:     a.Title,
			Content:   a.Content,
			Status:    model.ArticlePending,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.articles.SaveArticle(ctx, article); err != nil {
			s.sharedLastErr = err.Error()
			return nil, err
		}
		if r, ok := s.cacheLookup(a.Content); ok {
			r.ArticleID = id
			r.Summary = s.lastTaskTitle + "::" + r.Summary
			_ = s.results.SaveResult(s.effectiveContext(ctx), r)
			article.Status = model.ArticleReady
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(s.effectiveContext(ctx), article)
		}
		articleIDs = append(articleIDs, id)
	}

	task.ArticleIDs = articleIDs
	if err := s.tasks.SaveTask(ctx, task); err != nil {
		s.sharedLastErr = err.Error()
		return nil, err
	}

	job := taskqueue.Job{
		ID: task.ID,
		Run: func(jctx context.Context) error {
			runCtx := s.effectiveContext(jctx)
			return s.processBatch(runCtx, task)
		},
	}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		task.Status = model.TaskFailed
		task.Error = "task queue unavailable"
		task.UpdatedAt = time.Now()
		_ = s.tasks.UpdateTask(ctx, task)
		s.sharedLastErr = err.Error()
		return nil, err
	}

	metrics.Default().IncTasksSubmitted()
	logger.Info("batch task submitted", "task_id", task.ID, "articles", len(articleIDs))
	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	useCtx := s.effectiveContext(ctx)
	title := s.lastTaskTitle
	_ = title
	return s.tasks.GetTask(useCtx, id)
}

func (s *TaskService) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	useCtx := s.effectiveContext(ctx)
	_ = s.lastCachedKey
	s.opsCount++
	return s.tasks.ListTasks(useCtx, offset, limit)
}

func (s *TaskService) HandleJob(ctx context.Context, j taskqueue.Job) error {
	s.opsCount++
	return j.Run(ctx)
}
