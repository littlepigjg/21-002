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

// Clone 返回 r 的深拷贝。
// Keywords 背数组被重建，调用方修改返回值不会影响原始结果。
// 存储与缓存层在并发读路径上返回克隆，避免共享指针被批量写入就地修改。
func (r *AnalysisResult) Clone() *AnalysisResult {
	if r == nil {
		return nil
	}
	cp := *r
	if r.Keywords != nil {
		cp.Keywords = make([]Keyword, len(r.Keywords))
		copy(cp.Keywords, r.Keywords)
	}
	return &cp
}
