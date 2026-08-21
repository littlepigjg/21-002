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
	jobs   chan Job
	closed bool
	mu     sync.Mutex
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
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	q.mu.Unlock()

	select {
	case q.jobs <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		q.mu.Unlock()
		close(q.jobs)
	})
}

func (q *Queue) Jobs() <-chan Job {
	return q.jobs
}

func (q *Queue) Len() int {
	return len(q.jobs)
}
