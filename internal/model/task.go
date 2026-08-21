package model

import "time"

// TaskStatus 描述异步任务的执行状态。
type TaskStatus string

// 任务可能处于的状态。
const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskSuccess TaskStatus = "success"
	TaskFailed  TaskStatus = "failed"
)

// TaskType 区分任务类型。
type TaskType string

const (
	// TaskSingle 表示单篇文章分析任务。
	TaskSingle TaskType = "single"
	// TaskBatch 表示批量分析任务。
	TaskBatch TaskType = "batch"
)

// Task 表示一次异步分析任务，可关联一篇或多篇文章。
type Task struct {
	ID         string     `json:"id"`
	Type       TaskType   `json:"type"`
	Status     TaskStatus `json:"status"`
	Progress   int        `json:"progress"`
	ArticleIDs []string   `json:"article_ids"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
