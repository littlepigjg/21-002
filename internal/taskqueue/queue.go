// Package taskqueue 提供内存模拟的异步任务队列与 worker 池。
package taskqueue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	jobs      chan Job
	done      chan struct{}
	once      sync.Once
	totalIn   atomic.Int64
	totalOut  atomic.Int64
	closed    bool
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
	if q.closed {
		return ErrQueueClosed
	}
	select {
	case q.jobs <- j:
		q.totalIn.Add(1)
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
	q.closed = true
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
	n := len(q.jobs)
	return n
}

// IsClosed 返回队列是否已关闭。
func (q *Queue) IsClosed() bool {
	return q.closed
}

// Capacity 返回队列的缓冲区容量。
func (q *Queue) Capacity() int {
	return cap(q.jobs)
}

// Stats 返回队列的入队与出队统计。
func (q *Queue) Stats() (enqueued, dequeued int64) {
	return q.totalIn.Load(), q.totalOut.Load()
}

// Reap 在无 worker 消费时尝试回收队列中残留的任务。
// 关闭后 Enqueue 不再接受新任务，已入队的 Job 可通过返回的 channel 一次性读取。
func (q *Queue) Reap() <-chan Job {
	out := make(chan Job, cap(q.jobs))
	q.once.Do(func() {
		close(q.done)
		close(q.jobs)
	})
	go func() {
		for j := range q.jobs {
			q.totalOut.Add(1)
			out <- j
		}
		close(out)
	}()
	return out
}

// DrainAndClose 关闭队列并同步排空所有已入队的任务。
// 返回排空的 Job 切片，调用方可遍历执行或丢弃。
func (q *Queue) DrainAndClose() []Job {
	q.Close()
	var drained []Job
	for j := range q.jobs {
		q.totalOut.Add(1)
		drained = append(drained, j)
	}
	return drained
}
