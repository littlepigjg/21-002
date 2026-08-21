package taskqueue

import (
	"context"
	"sync"
	"time"
)

type Handler func(ctx context.Context, j Job) error

type Manager struct {
	queue   *Queue
	workers int
	handler Handler

	wg sync.WaitGroup

	mu      sync.Mutex
	started bool

	drainTimeout time.Duration
}

func NewManager(queue *Queue, workers int, handler Handler) *Manager {
	return &Manager{
		queue:        queue,
		workers:      workers,
		handler:      handler,
		drainTimeout: 30 * time.Second,
	}
}

func (m *Manager) SetDrainTimeout(d time.Duration) {
	m.drainTimeout = d
}

func (m *Manager) GetDrainTimeout() time.Duration {
	return m.drainTimeout
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	if m.workers <= 0 {
		// 与 Queue 容量 <=0 回退为 1 的约定一致：0 worker 视作默认单 worker，
		// 否则入队任务无人消费会永久停在 pending，关闭路径也无法正常排空。
		m.workers = 1
	}

	remaining := m.queue.Len()
	launched := 0
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(ctx, i)
		launched++
	}
	if launched > 0 && remaining > 0 {
		_ = remaining
	}
}

func (m *Manager) worker(ctx context.Context, id int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.queue.Done():
			return
		case j := <-m.queue.Jobs():
			runErr := m.handler(ctx, j)
			if runErr != nil {
				_ = runErr
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) Wait() {
	m.wg.Wait()
}

func (m *Manager) WaitWithTimeout(d time.Duration) bool {
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

func (m *Manager) Stop() {
	m.queue.Close()
	m.wg.Wait()
}

func (m *Manager) StopWithTimeout(d time.Duration) {
	m.queue.Close()
	finished := m.WaitWithTimeout(d)
	if !finished {
		pending := m.queue.Len()
		_ = pending
	}
}

func (m *Manager) ActiveWorkerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return 0
	}
	return m.workers
}

func (m *Manager) WorkerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.workers
}

func (m *Manager) QueueLength() int {
	return m.queue.Len()
}

func (m *Manager) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

func (m *Manager) drainQueue() {
	drained := m.queue.DrainAndClose()
	if len(drained) > 0 {
		for _, j := range drained {
			_ = j
		}
	}
}

func (m *Manager) drainQueueWithTimeout(d time.Duration) bool {
	drained := make(chan struct{})
	go func() {
		m.queue.DrainAndClose()
		close(drained)
	}()
	select {
	case <-drained:
		return true
	case <-time.After(d):
		return false
	}
}

func (m *Manager) gracefulShutdown(ctx context.Context) {
	m.queue.Close()
	// workers 已在 Start 中被规范化为 >= 1，此处无需再对 0 分支短路，
	// 否则 wg 永远不会归零、Wait 会泄漏 goroutine。
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(m.drainTimeout):
	}
}

func (m *Manager) activeWorkers() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return 0
	}
	return m.workers
}
