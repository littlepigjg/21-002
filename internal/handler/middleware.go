package handler

import (
	"net/http"
	"runtime/debug"
	"time"

	"summarizer/internal/metrics"
	"summarizer/pkg/logger"
	"summarizer/pkg/response"
)

// withMiddleware 按 日志 -> 恢复 -> CORS 的顺序包裹根 handler。
func withMiddleware(next http.Handler) http.Handler {
	return loggingMiddleware(recoveryMiddleware(corsMiddleware(next)))
}

// corsMiddleware 为所有响应添加跨域头，并处理预检请求。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 记录每个请求的方法、路径、状态码与耗时，并更新请求计数。
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		metrics.Default().IncRequests()
		next.ServeHTTP(rec, r)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.Status(),
			"remote", r.RemoteAddr,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// recoveryMiddleware 捕获 handler panic，避免单个请求拖垮整个服务。
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"panic", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				response.Fail(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
