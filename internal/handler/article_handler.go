package handler

import (
	"net/http"
	"strings"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/pkg/response"
)

// ArticleHandler 处理文章相关的 HTTP 请求。
type ArticleHandler struct {
	articles *service.ArticleService
}

// NewArticleHandler 构造 ArticleHandler。
func NewArticleHandler(articles *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articles: articles}
}

// Submit 处理 POST /api/v1/articles，提交单篇文章并同步分析。
func (h *ArticleHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req model.SubmitArticleRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.articles.Submit(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "article content is empty") {
			response.Fail(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.Contains(err.Error(), "article content exceeds limit") {
			response.Fail(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.Contains(err.Error(), "resource not found") {
			response.Fail(w, http.StatusNotFound, err.Error())
			return
		}
		response.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.OK(w, resp)
}

// Get 处理 GET /api/v1/articles/{id}，查询文章详情。
func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	article, err := h.articles.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, article)
}

// List 处理 GET /api/v1/articles，分页查询文章历史。
func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.articles.List(r.Context(), offset, limit)
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

// GetResult 处理 GET /api/v1/articles/{id}/result，查询文章分析结果。
func (h *ArticleHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.articles.GetResult(r.Context(), id)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, result)
}

// ListResults 处理 GET /api/v1/results，分页查询分析结果历史。
func (h *ArticleHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.articles.ListResults(r.Context(), offset, limit)
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
