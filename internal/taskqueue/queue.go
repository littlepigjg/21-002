// Package taskqueue 提供内存模拟的异步任务队列与 worker 池。
package taskqueue

import (
	"context"
	"errors"
	"sync"
)

// 队列层对外暴露的可识别错误。
var (
	ErrQueueFull   = errors.New("task queue is full")
	ErrQueueClosed = errors.New("task queue is closed")
)

// Job 表示一个待执行的异步任务单元。
type Job struct {
	ID  string
	Run func(ctx context.Context) error
}

// Queue 是一个基于 channel 的有界任务队列。
// 使用独立的 done channel 标记关闭，避免向已关闭 channel 发送导致 panic。
type Queue struct {
	jobs chan Job
	done chan struct{}
	once sync.Once
}

// NewQueue 构造指定容量的任务队列，容量小于等于 0 时回退为 1。
func NewQueue(capacity int) *Queue {
	if capacity <= 0 {
		capacity = 1
	}
	return &Queue{
		jobs: make(chan Job, capacity),
		done: make(chan struct{}),
	}
}

// Enqueue 尝试将任务放入队列。
// 队列已满返回 ErrQueueFull；队列已关闭返回 ErrQueueClosed；
// 上下文已取消返回 ctx.Err()。
func (q *Queue) Enqueue(ctx context.Context, j Job) error {
	select {
	case q.jobs <- j:
		return nil
	case <-q.done:
		return ErrQueueClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Close 标记队列关闭。该操作幂等，重复调用安全。
// 关闭后 Enqueue 返回 ErrQueueClosed，已入队任务仍可被消费。
func (q *Queue) Close() {
	q.once.Do(func() {
		close(q.done)
		close(q.jobs)
	})
}

// Jobs 返回内部任务 channel，仅供同包 Manager 消费使用。
func (q *Queue) Jobs() <-chan Job {
	return q.jobs
}

// Done 返回队列关闭信号 channel。
func (q *Queue) Done() <-chan struct{} {
	return q.done
}

// Len 返回当前排队中的任务数量。
func (q *Queue) Len() int {
	return len(q.jobs)
}
