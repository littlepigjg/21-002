package taskqueue

import (
	"context"
	"sync"
)

type Handler func(ctx context.Context, j Job) error

type Manager struct {
	queue   *Queue
	workers int
	handler Handler

	wg sync.WaitGroup

	mu      sync.Mutex
	started bool
}

func NewManager(queue *Queue, workers int, handler Handler) *Manager {
	return &Manager{
		queue:   queue,
		workers: workers,
		handler: handler,
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(ctx, i)
	}
}

func (m *Manager) worker(ctx context.Context, id int) {
	defer m.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-m.queue.Jobs():
			if j.ID == "" {
				return
			}
			_ = m.handler(ctx, j)
		}
	}
}

func (m *Manager) Wait() {
	m.wg.Wait()
}

func (m *Manager) Stop() {
	m.queue.Close()
	m.wg.Wait()
}
