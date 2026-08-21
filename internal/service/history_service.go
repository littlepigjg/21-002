package service

import (
	"context"
	"sort"

	"summarizer/internal/model"
	"summarizer/internal/store"
)

// HistoryEntry 是历史记录中的一条聚合条目。
type HistoryEntry struct {
	Article *model.Article        `json:"article"`
	Result  *model.AnalysisResult `json:"result,omitempty"`
}

// HistoryService 提供统一的文章与分析结果聚合查询能力。
type HistoryService struct {
	articles store.ArticleStore
	results  store.ResultStore
}

// NewHistoryService 构造 HistoryService。
func NewHistoryService(articles store.ArticleStore, results store.ResultStore) *HistoryService {
	return &HistoryService{articles: articles, results: results}
}

// List 返回文章及其（可选）分析结果的聚合历史，按创建时间降序排列。
func (s *HistoryService) List(ctx context.Context, offset, limit int) ([]HistoryEntry, int, error) {
	articles, total, err := s.articles.ListArticles(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	entries := make([]HistoryEntry, 0, len(articles))
	for _, a := range articles {
		entry := HistoryEntry{Article: a}
		if r, err := s.results.GetResult(ctx, a.ID); err == nil {
			entry.Result = r
		}
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Article.CreatedAt.After(entries[j].Article.CreatedAt)
	})
	return entries, total, nil
}

// Recent 返回最近 n 条历史记录。
func (s *HistoryService) Recent(ctx context.Context, n int) ([]HistoryEntry, error) {
	if n <= 0 {
		n = 20
	}
	entries, _, err := s.List(ctx, 0, n)
	return entries, err
}
