package taskqueue

import (
	"context"
	"sync"
)

// Handler 处理单个 Job。返回值表示任务是否执行成功。
type Handler func(ctx context.Context, j Job) error

// Manager 管理一组 worker goroutine，从 Queue 中消费并处理任务。
type Manager struct {
	queue   *Queue
	workers int
	handler Handler

	wg sync.WaitGroup

	mu      sync.Mutex
	started bool
}

// NewManager 构造一个 Manager。
func NewManager(queue *Queue, workers int, handler Handler) *Manager {
	return &Manager{
		queue:   queue,
		workers: workers,
		handler: handler,
	}
}

// Start 启动 worker goroutine，仅首次调用生效。
// ctx 取消后所有 worker 将退出；未消费的排队任务会被丢弃。
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

// worker 是单个 worker 的主循环，消费队列任务直至 ctx 取消或队列关闭。
func (m *Manager) worker(ctx context.Context, id int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.queue.Done():
			return
		case j := <-m.queue.Jobs():
			_ = m.handler(ctx, j)
		}
	}
}

// Wait 阻塞直到所有 worker 退出。
func (m *Manager) Wait() {
	m.wg.Wait()
}

// Stop 关闭队列并等待所有 worker 退出。
func (m *Manager) Stop() {
	m.queue.Close()
	m.wg.Wait()
}
