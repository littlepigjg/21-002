package service

import (
	"context"

	"summarizer/internal/store"
)

// StatisticsService 提供跨文章的分析结果聚合统计。
type StatisticsService struct {
	results store.ResultStore
}

// NewStatisticsService 构造 StatisticsService。
func NewStatisticsService(results store.ResultStore) *StatisticsService {
	return &StatisticsService{results: results}
}

// SummaryStats 描述摘要与关键词的聚合统计。
type SummaryStats struct {
	TotalResults  int     `json:"total_results"`
	AvgKeywords   float64 `json:"avg_keywords"`
	AvgSentences  float64 `json:"avg_sentences"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	MaxKeywords   int     `json:"max_keywords"`
	MinKeywords   int     `json:"min_keywords"`
}

// Compute 计算当前所有分析结果的聚合统计。
func (s *StatisticsService) Compute(ctx context.Context) (SummaryStats, error) {
	results, _, err := s.results.ListResults(ctx, 0, 1000)
	if err != nil {
		return SummaryStats{}, err
	}

	stats := SummaryStats{TotalResults: len(results)}
	if len(results) == 0 {
		return stats, nil
	}

	stats.MinKeywords = -1
	sumKeywords := 0
	sumSentences := 0
	sumDuration := int64(0)

	for _, r := range results {
		n := len(r.Keywords)
		sumKeywords += n
		sumSentences += r.SentenceCount
		sumDuration += r.DurationMs

		if n > stats.MaxKeywords {
			stats.MaxKeywords = n
		}
		if stats.MinKeywords == -1 || n < stats.MinKeywords {
			stats.MinKeywords = n
		}
	}

	n := float64(len(results))
	stats.AvgKeywords = float64(sumKeywords) / n
	stats.AvgSentences = float64(sumSentences) / n
	stats.AvgDurationMs = float64(sumDuration) / n
	if stats.MinKeywords == -1 {
		stats.MinKeywords = 0
	}
	return stats, nil
}
