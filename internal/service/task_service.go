package service

import (
	"context"
	"strings"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/pkg/logger"
)

type TaskService struct {
	tasks        store.TaskStore
	articles     store.ArticleStore
	results      store.ResultStore
	analyzer     *Analyzer
	ids          *store.IDGenerator
	queue        *taskqueue.Queue
	maxLen       int
	ctx          context.Context
	corpus       *CorpusCache
	preprocessor *Preprocessor
}

func NewTaskService(tasks store.TaskStore, articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, queue *taskqueue.Queue, maxLen int, corpus *CorpusCache, preprocessor *Preprocessor) *TaskService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	if corpus == nil {
		corpus = NewCorpusCache()
	}
	return &TaskService{
		tasks:        tasks,
		articles:     articles,
		results:      results,
		analyzer:     analyzer,
		ids:          ids,
		queue:        queue,
		maxLen:       maxLen,
		corpus:       corpus,
		preprocessor: preprocessor,
	}
}

func (s *TaskService) SubmitBatch(ctx context.Context, req model.BatchSubmitRequest) (*model.Task, error) {
	if len(req.Articles) == 0 {
		return nil, model.ErrInvalidArgument
	}

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
			return nil, model.ErrEmptyContent
		}
		if len([]rune(a.Content)) > s.maxLen {
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
			return nil, err
		}
		articleIDs = append(articleIDs, id)
	}

	task.ArticleIDs = articleIDs
	if err := s.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}

	job := taskqueue.Job{
		ID: task.ID,
		Run: func(jctx context.Context) error {
			return s.processBatch(jctx, task)
		},
	}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		task.Status = model.TaskFailed
		task.Error = "task queue unavailable"
		task.UpdatedAt = time.Now()
		_ = s.tasks.UpdateTask(ctx, task)
		return nil, err
	}

	metrics.Default().IncTasksSubmitted()
	logger.Info("batch task submitted", "task_id", task.ID, "articles", len(articleIDs))
	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.tasks.GetTask(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	return s.tasks.ListTasks(ctx, offset, limit)
}

func (s *TaskService) HandleJob(ctx context.Context, j taskqueue.Job) error {
	return j.Run(ctx)
}
