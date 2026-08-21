package handler

import (
	"net/http"
	"sync/atomic"

	"summarizer/internal/store"
	"summarizer/pkg/response"
)

// HealthHandler 提供健康检查与就绪检查端点。
type HealthHandler struct {
	store *store.MemoryStore
	ready atomic.Bool
}

// NewHealthHandler 构造 HealthHandler，初始状态为未就绪。
func NewHealthHandler(store *store.MemoryStore) *HealthHandler {
	h := &HealthHandler{store: store}
	h.ready.Store(false)
	return h
}

// SetReady 设置服务就绪状态。
func (h *HealthHandler) SetReady(v bool) {
	h.ready.Store(v)
}

// Health 处理 GET /health，返回服务存活状态。
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]interface{}{"status": "up"})
}

// Ready 处理 GET /ready，返回服务就绪状态与存储统计。
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		response.Fail(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	stats := h.store.Stats()
	response.OK(w, map[string]interface{}{
		"status": "ready",
		"stats":  stats,
	})
}
