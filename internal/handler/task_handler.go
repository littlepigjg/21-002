package handler

import (
	"net/http"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/pkg/response"
)

// TaskHandler 处理批量任务相关的 HTTP 请求。
type TaskHandler struct {
	tasks *service.TaskService
}

// NewTaskHandler 构造 TaskHandler。
func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// SubmitBatch 处理 POST /api/v1/batch，提交批量任务。
func (h *TaskHandler) SubmitBatch(w http.ResponseWriter, r *http.Request) {
	var req model.BatchSubmitRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.tasks.SubmitBatch(r.Context(), req)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, task)
}

// GetTask 处理 GET /api/v1/tasks/{id}，查询任务状态。
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.tasks.GetTask(r.Context(), id)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, task)
}

// ListTasks 处理 GET /api/v1/tasks，分页查询任务历史。
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.tasks.ListTasks(r.Context(), offset, limit)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, map[string]interface{}{
		"items":  items,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}
