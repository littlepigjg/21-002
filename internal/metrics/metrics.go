// Package metrics 提供进程内的原子计数器，用于轻量级观测。
package metrics

import "sync/atomic"

// Registry 保存一组可通过原子操作安全更新的计数器。
type Registry struct {
	requestsTotal  atomic.Int64
	tasksSubmitted atomic.Int64
	tasksCompleted atomic.Int64
	tasksFailed    atomic.Int64
	articlesTotal  atomic.Int64
	keywordsTotal  atomic.Int64
}

// Snapshot 是计数器当前值的只读快照。
type Snapshot struct {
	RequestsTotal  int64
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
	ArticlesTotal  int64
	KeywordsTotal  int64
}

// defaultRegistry 是进程内共享的默认注册表。
var defaultRegistry = &Registry{}

// Default 返回默认注册表。
func Default() *Registry {
	return defaultRegistry
}

// IncRequests 请求计数加一。
func (r *Registry) IncRequests() { r.requestsTotal.Add(1) }

// IncTasksSubmitted 已提交任务计数加一。
func (r *Registry) IncTasksSubmitted() { r.tasksSubmitted.Add(1) }

// IncTasksCompleted 已完成任务计数加一。
func (r *Registry) IncTasksCompleted() { r.tasksCompleted.Add(1) }

// IncTasksFailed 失败任务计数加一。
func (r *Registry) IncTasksFailed() { r.tasksFailed.Add(1) }

// IncArticles 文章计数增加 n。
func (r *Registry) IncArticles(n int) { r.articlesTotal.Add(int64(n)) }

// IncKeywords 提取关键词计数增加 n。
func (r *Registry) IncKeywords(n int) { r.keywordsTotal.Add(int64(n)) }

// Snapshot 返回当前所有计数器的快照。
func (r *Registry) Snapshot() Snapshot {
	return Snapshot{
		RequestsTotal:  r.requestsTotal.Load(),
		TasksSubmitted: r.tasksSubmitted.Load(),
		TasksCompleted: r.tasksCompleted.Load(),
		TasksFailed:    r.tasksFailed.Load(),
		ArticlesTotal:  r.articlesTotal.Load(),
		KeywordsTotal:  r.keywordsTotal.Load(),
	}
}
