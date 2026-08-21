package model

// Keyword 表示从文章中提取出的一个关键词及其统计信息。
type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
	TF    int     `json:"tf"`
	IDF   float64 `json:"idf"`
}
