// Package response 提供统一的 HTTP JSON 响应格式。
package response

import (
	"encoding/json"
	"net/http"
)

// Body 是所有接口统一返回的 JSON 结构。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON 写入统一格式的 JSON 响应。
func JSON(w http.ResponseWriter, status int, body Body) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// OK 写入成功响应（HTTP 200，code 0）。
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Body{Code: 0, Message: "ok", Data: data})
}

// Fail 写入失败响应，code 与 HTTP 状态码一致。
func Fail(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, Body{Code: status, Message: msg})
}

// FailErr 根据错误类型写入失败响应。
func FailErr(w http.ResponseWriter, err error) {
	status := StatusFromError(err)
	JSON(w, status, Body{Code: status, Message: err.Error()})
}
