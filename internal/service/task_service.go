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

// register 在登记簿中为一个任务登记进度桶。total 为该任务的条目总数。
// 同一任务重复登记会覆盖旧的进度桶，调用方应避免对正在处理的任务重复调用。
func (r *progressRegistry) register(taskID string, total int) {
	r.mu.Lock()
	r.batches[taskID] = &taskProgressInfo{
		Total:     total,
		Completed: make([]string, 0, total),
		UpdatedAt: time.Now(),
	}
	r.mu.Unlock()
}

// recordSuccess 在锁保护下原子地累加一次成功并追加已完成条目 ID。
// 读取/写入 info 字段必须在写锁下进行，避免与并发快照读取竞争。
func (r *progressRegistry) recordSuccess(taskID, articleID string) {
	r.mu.Lock()
	info := r.batches[taskID]
	if info != nil {
		info.Success++
		info.Completed = append(info.Completed, articleID)
		info.UpdatedAt = time.Now()
	}
	r.mu.Unlock()
}

// recordFail 在锁保护下原子地累加一次失败。
func (r *progressRegistry) recordFail(taskID string) {
	r.mu.Lock()
	info := r.batches[taskID]
	if info != nil {
		info.Fail++
		info.UpdatedAt = time.Now()
	}
	r.mu.Unlock()
}

// snapshot 返回某个任务进度的一份一致快照，completed 切片会被复制，
// 调用方拿到的副本不会与后续写入共享底层数组。任务不存在时返回零值快照。
func (r *progressRegistry) snapshot(taskID string) taskProgressInfo {
	r.mu.RLock()
	info := r.batches[taskID]
	if info == nil {
		r.mu.RUnlock()
		return taskProgressInfo{}
	}
	out := taskProgressInfo{
		Total:     info.Total,
		Success:   info.Success,
		Fail:      info.Fail,
		UpdatedAt: info.UpdatedAt,
	}
	if len(info.Completed) > 0 {
		out.Completed = append([]string(nil), info.Completed...)
	} else {
		out.Completed = []string{}
	}
	r.mu.RUnlock()
	return out
}

// summarySnapshot 返回单个任务的总量/成功/失败计数以及已复制完成的条目 ID 列表，
// 用于批次处理结束后的汇总日志。所有读取均在 RLock 下完成，completed 为独立副本。
func (r *progressRegistry) summarySnapshot(taskID string) (total int, success int, fail int, completed []string) {
	s := r.snapshot(taskID)
	return s.Total, s.Success, s.Fail, s.Completed
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
	snap := s.progress.snapshot(taskID)
	percent := "0%"
	if snap.Total > 0 {
		done := snap.Success + snap.Fail
		percent = fmt.Sprintf("%d%%", done*100/snap.Total)
	}
	return TaskProgress{
		TaskID:    taskID,
		Total:     snap.Total,
		Success:   snap.Success,
		Fail:      snap.Fail,
		Completed: snap.Completed,
		Percent:   percent,
	}
}

// ListProgress 返回所有任务进度的一份快照映射。它在 RLock 下一次性采集任务 ID，
// 再逐个通过 snapshot 复制字段，确保返回的 Completed 切片与并发写入互不干扰。
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

// AggregateProgressSnapshot 返回所有任务进度的一份一致快照切片。
// 每个条目的字段都在同一把 RLock 下读取，Completed 切片会被复制，
// 因此调用方拿到的数据不会与 worker 的并发写入产生数据竞争。
func (s *TaskService) AggregateProgressSnapshot() []TaskProgress {
	s.progress.mu.RLock()
	list := make([]TaskProgress, 0, len(s.progress.batches))
	for id, info := range s.progress.batches {
		done := info.Success + info.Fail
		pct := "0%"
		if info.Total > 0 {
			pct = fmt.Sprintf("%d%%", done*100/info.Total)
		}
		completed := []string{}
		if len(info.Completed) > 0 {
			completed = append([]string(nil), info.Completed...)
		}
		list = append(list, TaskProgress{
			TaskID:    id,
			Total:     info.Total,
			Success:   info.Success,
			Fail:      info.Fail,
			Completed: completed,
			Percent:   pct,
		})
	}
	s.progress.mu.RUnlock()
	return list
}

func (s *TaskService) PruneProgress(taskID string) {
	s.progress.prune(taskID)
}
