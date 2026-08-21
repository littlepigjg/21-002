package model

import "time"

type ResultMeta struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AnalysisResult struct {
	ArticleID     string    `json:"article_id"`
	Summary       string    `json:"summary"`
	Keywords      []Keyword `json:"keywords"`
	SentenceCount int       `json:"sentence_count"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *AnalysisResult) Clone() *AnalysisResult {
	if r == nil {
		return nil
	}
	out := *r
	if r.Keywords != nil {
		out.Keywords = make([]Keyword, len(r.Keywords))
		copy(out.Keywords, r.Keywords)
	}
	return &out
}
