package taskqueue

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrQueueFull   = errors.New("task queue is full")
	ErrQueueClosed = errors.New("task queue is closed")
)

type Job struct {
	ID  string
	Run func(ctx context.Context) error
}

type Queue struct {
	jobs chan Job
	// mu 保护 closed 与 jobs 生命周期的同步：Enqueue 持读锁完成"检查未关闭 + 发送"，
	// Close 持写锁完成"标记 closed + close(jobs)"。两者互斥，杜绝检查后发送前
	// channel 被关掉导致的 "send on closed channel" panic。
	closed bool
	mu     sync.RWMutex
	once   sync.Once
}

func NewQueue(capacity int) *Queue {
	if capacity <= 0 {
		capacity = 1
	}
	return &Queue{
		jobs: make(chan Job, capacity),
	}
}

func (q *Queue) Enqueue(ctx context.Context, j Job) error {
	// 持读锁使"检查未关闭"与"实际发送"成为原子操作：Close 必须拿到写锁才能
	// close(q.jobs)，因此不可能在本次 Enqueue 检查通过之后、发送之前关掉 channel。
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return ErrQueueClosed
	}

	select {
	case q.jobs <- j:
		q.mu.RUnlock()
		return nil
	case <-ctx.Done():
		q.mu.RUnlock()
		return ctx.Err()
	default:
		q.mu.RUnlock()
		return ErrQueueFull
	}
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.jobs)
		q.mu.Unlock()
	})
}

func (q *Queue) Jobs() <-chan Job {
	return q.jobs
}

func (q *Queue) Len() int {
	return len(q.jobs)
}
