package model

// UpdateArticleRequest 是更新文章标题的请求体。
type UpdateArticleRequest struct {
	Title string `json:"title"`
}

// TaskQueryResult 是任务列表查询的返回结构。
type TaskQueryResult struct {
	Items []*Task `json:"items"`
	Total int     `json:"total"`
}

// ArticleQueryResult 是文章列表查询的返回结构。
type ArticleQueryResult struct {
	Items []*Article `json:"items"`
	Total int        `json:"total"`
}
