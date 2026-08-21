package httputil

import "net/http"

// SetJSON 设置 JSON 响应的 Content-Type 头。
func SetJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// SetNoCache 设置禁止缓存的响应头。
func SetNoCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// SetCORS 设置允许跨域的响应头。
func SetCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
