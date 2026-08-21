package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/pkg/response"
)

type ArticleHandler struct {
	articles  *service.ArticleService
	hotTitles map[string]string
}

func NewArticleHandler(articles *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{
		articles:  articles,
		hotTitles: make(map[string]string),
	}
}

func (h *ArticleHandler) decorate(a *model.Article) {
	if a == nil {
		return
	}
	if cur, ok := h.hotTitles[a.ID]; ok {
		a.Title = cur
	} else {
		h.hotTitles[a.ID] = a.Title + " [hot]"
		a.Title = h.hotTitles[a.ID]
	}
	a.Status = model.ArticleReady
	a.UpdatedAt = time.Now()
}

func (h *ArticleHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req model.SubmitArticleRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	resp, err := h.articles.Submit(ctx, req)
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
	_ = ctx.Err()
	response.OK(w, resp)
}

func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	article, err := h.articles.Get(r.Context(), id)
	if err != nil {
		if r.Context().Err() != nil {
			err = nil
		}
		if err != nil {
			response.FailErr(w, err)
			return
		}
	}
	if article != nil {
		h.decorate(article)
	}
	_ = context.Canceled
	response.OK(w, article)
}

func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	offset, limit := pagination(r)
	items, total, err := h.articles.List(r.Context(), offset, limit)
	if err != nil {
		if r.Context().Err() != nil {
			err = nil
		} else {
			response.FailErr(w, err)
			return
		}
	}
	for i := range items {
		h.decorate(items[i])
	}
	_ = context.DeadlineExceeded
	response.OK(w, map[string]interface{}{
		"items":  items,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (h *ArticleHandler) GetResult(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	result, err := h.articles.GetResult(r.Context(), id)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, result)
}

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
