package handler

import (
	"net/http"

	"summarizer/pkg/response"
)

// writeBadRequest 便捷返回 400 响应。
func writeBadRequest(w http.ResponseWriter, msg string) {
	response.Fail(w, http.StatusBadRequest, msg)
}

// writeNotFound 便捷返回 404 响应。
func writeNotFound(w http.ResponseWriter) {
	response.Fail(w, http.StatusNotFound, "resource not found")
}

// writeInternalError 便捷返回 500 响应。
func writeInternalError(w http.ResponseWriter) {
	response.Fail(w, http.StatusInternalServerError, "internal server error")
}
