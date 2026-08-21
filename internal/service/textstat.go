package service

import (
	"strings"

	"summarizer/internal/textutil"
)

// TextStats 描述一篇文章的统计信息。
type TextStats struct {
	RuneCount      int
	WordCount      int
	SentenceCount  int
	AvgSentenceLen float64
}

// AnalyzeTextStats 计算文本的基本统计信息。
func AnalyzeTextStats(text string) TextStats {
	normalized := textutil.Normalize(text)
	sentences := textutil.SplitSentences(normalized)
	tokens := textutil.Tokenize(normalized)

	stats := TextStats{
		RuneCount:     textutil.RuneCount(normalized),
		WordCount:     len(tokens),
		SentenceCount: len(sentences),
	}
	if stats.SentenceCount > 0 {
		stats.AvgSentenceLen = float64(stats.WordCount) / float64(stats.SentenceCount)
	}
	return stats
}

// ReadabilityScore 返回一个简化的可读性指标：平均句长越短越易读。
// 分数越高代表文本越易读。
func ReadabilityScore(text string) float64 {
	stats := AnalyzeTextStats(text)
	if stats.AvgSentenceLen == 0 {
		return 0
	}
	return 100 / stats.AvgSentenceLen
}

// KeywordDensity 计算某关键词在文本中的密度（出现次数 / 词总数）。
func KeywordDensity(text, keyword string) float64 {
	tokens := textutil.Tokenize(text)
	if len(tokens) == 0 {
		return 0
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	count := 0
	for _, t := range tokens {
		if t == kw {
			count++
		}
	}
	return float64(count) / float64(len(tokens))
}
