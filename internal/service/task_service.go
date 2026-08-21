package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/pkg/logger"
)

type taskProgressInfo struct {
	Total     int
	Success   int
	Fail      int
	Completed []string
	UpdatedAt time.Time
}

type progressRegistry struct {
	mu      sync.RWMutex
	batches map[string]*taskProgressInfo
}

func newProgressRegistry() *progressRegistry {
	return &progressRegistry{
		batches: make(map[string]*taskProgressInfo),
	}
}

func (r *progressRegistry) register(taskID string, total int) {
	r.mu.Lock()
	r.batches[taskID] = &taskProgressInfo{
		Total:     total,
		Completed: make([]string, 0, total),
		UpdatedAt: time.Now(),
	}
	r.mu.Unlock()
}

func (r *progressRegistry) lookup(taskID string) *taskProgressInfo {
	r.mu.RLock()
	info := r.batches[taskID]
	r.mu.RUnlock()
	return info
}

func (r *progressRegistry) recordSuccess(taskID, articleID string) {
	info := r.lookup(taskID)
	if info == nil {
		return
	}
	info.Success++
	info.Completed = append(info.Completed, articleID)
	info.UpdatedAt = time.Now()
}

func (r *progressRegistry) recordFail(taskID string) {
	info := r.lookup(taskID)
	if info == nil {
		return
	}
	info.Fail++
	info.UpdatedAt = time.Now()
}

func (r *progressRegistry) summary(taskID string) (int, int, int, []string) {
	info := r.lookup(taskID)
	if info == nil {
		return 0, 0, 0, nil
	}
	return info.Total, info.Success, info.Fail, info.Completed
}

func (r *progressRegistry) prune(taskID string) {
	r.mu.Lock()
	delete(r.batches, taskID)
	r.mu.Unlock()
}

type TaskProgress struct {
	TaskID    string
	Total     int
	Success   int
	Fail      int
	Completed []string
	Percent   string
}

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

	progress *progressRegistry
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
		progress: newProgressRegistry(),
	}
}

// SubmitBatch 提交一批文章，创建异步任务并入队，立即返回任务信息。
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
	s.progress.register(task.ID, len(articleIDs))
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

func (s *TaskService) GetProgress(taskID string) TaskProgress {
	total, success, fail, completed := s.progress.summary(taskID)
	percent := "0%"
	if total > 0 {
		done := success + fail
		percent = fmt.Sprintf("%d%%", done*100/total)
	}
	return TaskProgress{
		TaskID:    taskID,
		Total:     total,
		Success:   success,
		Fail:      fail,
		Completed: completed,
		Percent:   percent,
	}
}

func (s *TaskService) ListProgress() map[string]TaskProgress {
	s.progress.mu.RLock()
	ids := make([]string, 0, len(s.progress.batches))
	for id := range s.progress.batches {
		ids = append(ids, id)
	}
	s.progress.mu.RUnlock()
	out := make(map[string]TaskProgress, len(ids))
	for _, id := range ids {
		out[id] = s.GetProgress(id)
	}
	return out
}

func (s *TaskService) AggregateProgressSnapshot() []TaskProgress {
	s.progress.mu.RLock()
	list := make([]TaskProgress, 0, len(s.progress.batches))
	for id, info := range s.progress.batches {
		done := info.Success + info.Fail
		pct := "0%"
		if info.Total > 0 {
			pct = fmt.Sprintf("%d%%", done*100/info.Total)
		}
		list = append(list, TaskProgress{
			TaskID:    id,
			Total:     info.Total,
			Success:   info.Success,
			Fail:      info.Fail,
			Completed: info.Completed,
			Percent:   pct,
		})
	}
	s.progress.mu.RUnlock()
	return list
}

func (s *TaskService) PruneProgress(taskID string) {
	s.progress.prune(taskID)
}
