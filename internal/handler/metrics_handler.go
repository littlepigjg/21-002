package handler

import (
	"net/http"

	"summarizer/internal/metrics"
	"summarizer/pkg/response"
)

// MetricsHandler 暴露运行时指标查询接口。
type MetricsHandler struct{}

// NewMetricsHandler 构造 MetricsHandler。
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// Snapshot 处理 GET /api/v1/metrics，返回当前指标快照。
func (h *MetricsHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	response.OK(w, metrics.Default().Snapshot())
}
