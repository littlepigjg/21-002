package handler

import (
	"net/http"
	"strings"
)

// StaticHandler 提供带缓存控制的前端静态资源服务。
type StaticHandler struct {
	fs http.Handler
}

// NewStaticHandler 构造 StaticHandler，dir 为静态资源根目录。
func NewStaticHandler(dir string) *StaticHandler {
	return &StaticHandler{fs: http.FileServer(http.Dir(dir))}
}

// ServeHTTP 为 HTML 页面设置 no-cache，为其余资源设置较长的缓存时间。
func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html") {
		w.Header().Set("Cache-Control", "no-cache")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	h.fs.ServeHTTP(w, r)
}
