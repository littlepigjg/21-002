package handler

import (
	"net/http"

	"summarizer/internal/store"
	"summarizer/pkg/response"
)

// ExportHandler 暴露存储快照查询接口。
type ExportHandler struct {
	store *store.MemoryStore
}

// NewExportHandler 构造 ExportHandler。
func NewExportHandler(s *store.MemoryStore) *ExportHandler {
	return &ExportHandler{store: s}
}

// Snapshot 处理 GET /api/v1/export，返回当前存储快照。
func (h *ExportHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	response.OK(w, h.store.Snapshot(r.Context()))
}
