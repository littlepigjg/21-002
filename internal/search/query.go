package search

import (
	"strings"

	"summarizer/internal/textutil"
)

// tokenizeQuery 将查询文本分词并去除停用词，返回查询词列表。
func tokenizeQuery(query string, stopwords *textutil.StopwordSet) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	tokens := textutil.Tokenize(query)
	return stopwords.Filter(tokens)
}
