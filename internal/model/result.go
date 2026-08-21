package model

import "time"

// AnalysisResult 保存一篇文章的分析结果（摘要、关键词、耗时等）。
type AnalysisResult struct {
	ArticleID     string    `json:"article_id"`
	Summary       string    `json:"summary"`
	Keywords      []Keyword `json:"keywords"`
	SentenceCount int       `json:"sentence_count"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}
