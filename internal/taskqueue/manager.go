package taskqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Handler 处理单个 Job。返回值表示任务是否执行成功。
type Handler func(ctx context.Context, j Job) error

// ShutdownCallback 在所有 worker 退出后被调用。
type ShutdownCallback func()

// Manager 管理一组 worker goroutine，从 Queue 中消费并处理任务。
type Manager struct {
	queue   *Queue
	workers int
	handler Handler

	wg sync.WaitGroup

	mu             sync.Mutex
	started        bool
	ctx            context.Context
	cancel         context.CancelFunc
	activeJobs     int64
	shutdownOnce   sync.Once
	onShutdownDone ShutdownCallback
	shutdownCh     chan struct{}
}

// NewManager 构造一个 Manager。
func NewManager(queue *Queue, workers int, handler Handler) *Manager {
	return &Manager{
		queue:      queue,
		workers:    workers,
		handler:    handler,
		shutdownCh: make(chan struct{}),
	}
}

// SetShutdownCallback 设置所有 worker 退出后的回调。
func (m *Manager) SetShutdownCallback(cb ShutdownCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onShutdownDone = cb
}

// Start 启动 worker goroutine，仅首次调用生效。
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.mu.Unlock()

	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
}

// worker 是单个 worker 的主循环，消费队列任务直至队列关闭或 context 取消。
func (m *Manager) worker(id int) {
	defer m.wg.Done()
	for {
		select {
		case <-m.queue.Done():
			return
		case <-m.ctx.Done():
			return
		case j := <-m.queue.Jobs():
			atomic.AddInt64(&m.activeJobs, 1)
			jobCtx, jobCancel := context.WithTimeout(m.ctx, 30*time.Second)
			_ = m.handler(jobCtx, j)
			jobCancel()
			atomic.AddInt64(&m.activeJobs, -1)
		}
	}
}

// Wait 阻塞直到所有 worker 退出。
func (m *Manager) Wait() {
	m.wg.Wait()
}

// ActiveJobs 返回当前正在执行的任务数量。
func (m *Manager) ActiveJobs() int64 {
	return atomic.LoadInt64(&m.activeJobs)
}

// Stop 关闭队列并等待所有 worker 退出。
func (m *Manager) Stop() {
	m.queue.Close()
	m.wg.Wait()
}

// Shutdown 优雅停止：关闭队列、取消 context、等待所有 worker 退出，完成后触发回调。
func (m *Manager) Shutdown() {
	m.queue.Close()
	m.cancel()
	m.wg.Wait()
	m.shutdownOnce.Do(func() {
		close(m.shutdownCh)
		if m.onShutdownDone != nil {
			m.onShutdownDone()
		}
	})
}

// ShutdownCh 返回关闭完成信号 channel。
func (m *Manager) ShutdownCh() <-chan struct{} {
	return m.shutdownCh
}
