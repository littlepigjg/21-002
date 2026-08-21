package service

import (
	"context"
	"fmt"
	"strings"

	"summarizer/internal/store"
)

// ReportService 生成文章分析报告。
type ReportService struct {
	articles store.ArticleStore
	results  store.ResultStore
}

// NewReportService 构造 ReportService。
func NewReportService(articles store.ArticleStore, results store.ResultStore) *ReportService {
	return &ReportService{articles: articles, results: results}
}

// Generate 生成指定文章的纯文本分析报告。
func (s *ReportService) Generate(ctx context.Context, articleID string) (string, error) {
	article, err := s.articles.GetArticle(ctx, articleID)
	if err != nil {
		return "", err
	}
	result, err := s.results.GetResult(ctx, articleID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("=== 文章分析报告 ===\n")
	b.WriteString(fmt.Sprintf("标题: %s\n", article.Title))
	b.WriteString(fmt.Sprintf("文章ID: %s\n", article.ID))
	b.WriteString(fmt.Sprintf("句子数: %d\n", result.SentenceCount))
	b.WriteString(fmt.Sprintf("耗时: %d ms\n", result.DurationMs))
	b.WriteString("\n【摘要】\n")
	b.WriteString(result.Summary)
	b.WriteString("\n\n【关键词】\n")
	for i, k := range result.Keywords {
		b.WriteString(fmt.Sprintf("%d. %s (score=%.4f)\n", i+1, k.Word, k.Score))
	}
	return b.String(), nil
}

// ListReports 批量生成多篇文章的分析报告。
func (s *ReportService) ListReports(ctx context.Context, ids []string) (map[string]string, error) {
	reports := make(map[string]string, len(ids))
	for _, id := range ids {
		r, err := s.Generate(ctx, id)
		if err != nil {
			continue
		}
		reports[id] = r
	}
	return reports, nil
}
