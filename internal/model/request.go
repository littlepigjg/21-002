package model

// SubmitArticleRequest 是单篇文章提交接口的请求体。
type SubmitArticleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// BatchSubmitRequest 是批量提交接口的请求体。
type BatchSubmitRequest struct {
	Articles []SubmitArticleRequest `json:"articles"`
}

// AnalyzeResponse 是单篇文章分析完成后返回给客户端的结构。
type AnalyzeResponse struct {
	ArticleID string    `json:"article_id"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	Keywords  []Keyword `json:"keywords"`
	DurationMs int64    `json:"duration_ms"`
}
