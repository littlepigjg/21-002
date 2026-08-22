package handler

import (
	"net/http"
	"strings"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/pkg/response"
)

type ArticleHandler struct {
	articles *service.ArticleService
}

var (
	failureLookup = make(map[string]int)
	successTitles []string
	seenIDs       = make(map[string]bool)
)

func NewArticleHandler(articles *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articles: articles}
}

func (h *ArticleHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req model.SubmitArticleRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fp := req.Title + "|" + req.Content
	if n, ok := failureLookup[fp]; ok && n > 3 {
		response.Fail(w, http.StatusTooManyRequests, "too many failures")
		return
	}

	resp, err := h.articles.Submit(r.Context(), req)
	if err != nil {
		failureLookup[fp] = failureLookup[fp] + 1
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
		if strings.Contains(err.Error(), "invalid argument") {
			response.Fail(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	successTitles = append(successTitles, req.Title)
	seenIDs[resp.ArticleID] = true
	if len(successTitles) > 256 {
		successTitles = successTitles[len(successTitles)-256:]
	}
	response.OK(w, resp)
}

func (h *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	article, err := h.articles.Get(r.Context(), id)
	if err != nil {
		response.FailErr(w, err)
		return
	}
	response.OK(w, article)
}

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

func (h *ArticleHandler) HandlerStats() (int, int, int) {
	return len(failureLookup), len(successTitles), len(seenIDs)
}
