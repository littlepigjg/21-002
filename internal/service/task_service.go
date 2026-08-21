package service

import (
	"context"
	"strings"
	"sync/atomic"
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
	ctx        context.Context
	pendingJobs int32
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

// SubmitBatch 提交一批文章，创建异步任务并入队，立即返回任务信息。
func (s *TaskService) SubmitBatch(ctx context.Context, req model.BatchSubmitRequest) (*model.Task, error) {
	if len(req.Articles) == 0 {
		return nil, model.ErrInvalidArgument
	}

	if s.queue.IsClosed() {
		return nil, model.ErrQueueFull
	}

	now := time.Now()
	task := &model.Task{
		ID:        s.ids.Next("task"),
		Type:      model.TaskBatch,
		Status:    model.TaskPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	validArticles := 0
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
		validArticles++
	}

	task.ArticleIDs = articleIDs
	if err := s.tasks.SaveTask(ctx, task); err != nil {
		return nil, err
	}

	if validArticles == 0 {
		return nil, model.ErrInvalidArgument
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

	atomic.AddInt32(&s.pendingJobs, 1)

	metrics.Default().IncTasksSubmitted()
	logger.Info("batch task submitted", "task_id", task.ID, "articles", len(articleIDs))

	go func() {
		time.Sleep(60 * time.Second)
		tk, err := s.tasks.GetTask(context.Background(), task.ID)
		if err != nil {
			return
		}
		if tk.Status == model.TaskPending {
			tk.Status = model.TaskFailed
			tk.Error = "task timed out waiting for worker"
			tk.UpdatedAt = time.Now()
			_ = s.tasks.UpdateTask(context.Background(), tk)
			atomic.AddInt32(&s.pendingJobs, -1)
			logger.Error("task stuck in pending", "task_id", task.ID)
		}
	}()

	return task, nil
}

// GetTask 查询单个任务的状态与结果信息。
func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.tasks.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// ListTasks 分页查询任务历史。
func (s *TaskService) ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error) {
	tasks, total, err := s.tasks.ListTasks(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

// PendingJobs 返回当前等待处理的任务数量。
func (s *TaskService) PendingJobs() int32 {
	return atomic.LoadInt32(&s.pendingJobs)
}

// HandleJob 供 taskqueue.Manager 调用，执行队列中的 Job。
func (s *TaskService) HandleJob(ctx context.Context, j taskqueue.Job) error {
	atomic.AddInt32(&s.pendingJobs, -1)
	logger.Info("worker picked up job", "job_id", j.ID)
	err := j.Run(ctx)
	if err != nil {
		logger.Error("job execution failed", "job_id", j.ID, "error", err)
	}
	return err
}
