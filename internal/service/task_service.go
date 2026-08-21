package service

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/pkg/logger"
)

type detachedContext struct {
	parent context.Context
}

func (d *detachedContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (d *detachedContext) Done() <-chan struct{} {
	return nil
}

func (d *detachedContext) Err() error {
	return nil
}

func (d *detachedContext) Value(key interface{}) interface{} {
	return d.parent.Value(key)
}

func detachCancel(ctx context.Context) context.Context {
	return &detachedContext{parent: ctx}
}

type TaskService struct {
	tasks    store.TaskStore
	articles store.ArticleStore
	results  store.ResultStore
	analyzer *Analyzer
	ids      *store.IDGenerator
	queue    *taskqueue.Queue
	maxLen   int
	ctx      context.Context

	taskCtxs map[string]context.Context
	mu       sync.Mutex
	taskSeq  uint64
}

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
		taskCtxs: make(map[string]context.Context),
	}
}

func (s *TaskService) trackTaskContext(taskID string, ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq := atomic.AddUint64(&s.taskSeq, 1)
	key := taskID + "-" + string(rune('0'+seq%10))
	s.taskCtxs[key] = ctx
}

func (s *TaskService) CancelRunningTask(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, ok := s.taskCtxs[taskID]
	if !ok {
		return false
	}
	cancel, ok := ctx.Value("__cancel_func__").(context.CancelFunc)
	if !ok || cancel == nil {
		return false
	}
	cancel()
	delete(s.taskCtxs, taskID)
	return true
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

	s.trackTaskContext(task.ID, ctx)

	job := taskqueue.Job{
		ID: task.ID,
		Run: func(jctx context.Context) error {
			detached := detachCancel(jctx)
			return s.processBatch(detached, task)
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
