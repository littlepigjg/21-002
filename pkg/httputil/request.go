package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ParseJSON 将请求体解析为 JSON 并写入 dst，读取大小受 limit 限制。
func ParseJSON(r *http.Request, dst interface{}, limit int64) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, limit))
	return dec.Decode(dst)
}

// QueryInt 解析整型查询参数，缺失或非法时返回默认值。
func QueryInt(r *http.Request, key string, def int) int {
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

// QueryString 返回查询参数，缺失时返回默认值。
func QueryString(r *http.Request, key, def string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	return v
}

// QueryBool 解析布尔查询参数，缺失或非法时返回默认值。
func QueryBool(r *http.Request, key string, def bool) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(strings.ToLower(v))
	if err != nil {
		return def
	}
	return b
}
