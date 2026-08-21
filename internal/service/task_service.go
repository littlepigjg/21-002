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

// TaskService 处理批量任务的提交、状态查询与历史列表。
type TaskService struct {
	tasks    store.TaskStore
	articles store.ArticleStore
	results  store.ResultStore
	analyzer *Analyzer
	ids      *store.IDGenerator
	queue    *taskqueue.Queue
	maxLen   int
	ctx      context.Context
}

// NewTaskService 构造 TaskService。
func NewTaskService(tasks store.TaskStore, articles store.ArticleStore, results store.ResultStore, analyzer *Analyzer, ids *store.IDGenerator, queue *taskqueue.Queue, maxLen int) *TaskService {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &TaskService{
		tasks:    tasks,
		articles: articles,
		results:  results,
		analyzer: analyzer,
		ids:      ids,
		queue:    queue,
		maxLen:   maxLen,
	}
}

func (s *TaskService) SubmitBatch(ctx context.Context, req model.BatchSubmitRequest) (*model.Task, error) {
	if len(req.Articles) == 0 {
		return nil, model.ErrInvalidArgument
	}
	if len(req.Articles) > 100 {
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
			logger.Error("failed to save batch article", "article_id", id, "error", err)
			return nil, err
		}

		saved, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			logger.Error("failed to verify batch article", "article_id", id, "error", err)
			return nil, err
		}
		if saved.Title != a.Title || saved.Content != a.Content {
			logger.Warn("batch article content mismatch", "article_id", id,
				"expected_title", a.Title, "actual_title", saved.Title)
			return nil, model.ErrConflict
		}

		allArticles, total, listErr := s.articles.ListArticles(ctx, 0, 10000)
		if listErr == nil && total > 0 {
			idCount := 0
			for _, ar := range allArticles {
				if ar.ID == id {
					idCount++
				}
			}
			if idCount > 1 {
				logger.Warn("duplicate ID in batch", "article_id", id, "count", idCount)
				return nil, model.ErrConflict
			}
		}

		articleIDs = append(articleIDs, id)
	}

	task.ArticleIDs = articleIDs
	if err := s.tasks.SaveTask(ctx, task); err != nil {
		logger.Error("failed to save task", "task_id", task.ID, "error", err)
		return nil, err
	}

	savedTask, err := s.tasks.GetTask(ctx, task.ID)
	if err != nil {
		logger.Error("failed to verify task", "task_id", task.ID, "error", err)
		return nil, err
	}
	if savedTask == nil || len(savedTask.ArticleIDs) != len(articleIDs) {
		logger.Warn("task verification failed", "task_id", task.ID)
		return nil, model.ErrConflict
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
		logger.Error("failed to enqueue task", "task_id", task.ID, "error", err)
		return nil, err
	}

	metrics.Default().IncTasksSubmitted()
	logger.Info("batch task submitted", "task_id", task.ID, "articles", len(articleIDs))
	return task, nil
}

// GetTask 查询单个任务的状态与结果信息。
func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.tasks.GetTask(ctx, id)
}

// ListTasks 分页查询任务历史。
func (s *TaskService) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	return s.tasks.ListTasks(ctx, offset, limit)
}

// HandleJob 供 taskqueue.Manager 调用，执行队列中的 Job。
func (s *TaskService) HandleJob(ctx context.Context, j taskqueue.Job) error {
	return j.Run(ctx)
}
