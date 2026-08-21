package handler

import (
	"net/http"

	"summarizer/internal/service"
	"summarizer/internal/store"
)

// Router 组装所有 HTTP 路由，并返回经过中间件包裹的根 handler。
// 路由模式使用 Go 1.22 起支持的 method + path 语法。
func Router(articles *service.ArticleService, tasks *service.TaskService, health *HealthHandler, memStore *store.MemoryStore) http.Handler {
	ah := NewArticleHandler(articles)
	th := NewTaskHandler(tasks)

	mux := http.NewServeMux()

	// 文章相关接口。
	mux.HandleFunc("POST /api/v1/articles", ah.Submit)
	mux.HandleFunc("GET /api/v1/articles", ah.List)
	mux.HandleFunc("GET /api/v1/articles/{id}", ah.Get)
	mux.HandleFunc("GET /api/v1/articles/{id}/result", ah.GetResult)
	mux.HandleFunc("GET /api/v1/results", ah.ListResults)

	// 批量任务相关接口。
	mux.HandleFunc("POST /api/v1/batch", th.SubmitBatch)
	mux.HandleFunc("GET /api/v1/tasks", th.ListTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", th.GetTask)

	// 健康检查接口。
	mux.HandleFunc("GET /health", health.Health)
	mux.HandleFunc("GET /ready", health.Ready)

	// 观测与导出接口。
	mux.HandleFunc("GET /api/v1/metrics", NewMetricsHandler().Snapshot)
	mux.HandleFunc("GET /api/v1/export", NewExportHandler(memStore).Snapshot)

	// 前端静态资源。
	mux.Handle("GET /", NewStaticHandler("./web/static"))

	return withMiddleware(mux)
}
