package httputil

import "net/http"

// StatusRecorder 包装 http.ResponseWriter，用于记录实际写入的状态码。
type StatusRecorder struct {
	http.ResponseWriter
	status int
}

// NewStatusRecorder 构造一个状态码记录器。
func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{ResponseWriter: w}
}

// WriteHeader 记录状态码并透传给底层 ResponseWriter。
func (r *StatusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

// Write 若此前未显式写入状态码，则默认记为 200。
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Status 返回记录的状态码，未写入时默认返回 200。
func (r *StatusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
