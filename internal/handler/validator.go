package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

// maxBodyBytes 限制请求体读取上限，防止恶意超大请求耗尽内存。
const maxBodyBytes = 1 << 20

// decodeJSON 解析请求体为 JSON 并写入 dst，读取大小受 maxBodyBytes 限制。
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	return dec.Decode(dst)
}

// pagination 从查询参数解析 offset/limit，并应用默认值与上限约束。
func pagination(r *http.Request) (int, int) {
	offset := parseIntQuery(r, "offset", 0)
	limit := parseIntQuery(r, "limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}

// parseIntQuery 解析整型查询参数，缺失或非法时返回默认值。
func parseIntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
