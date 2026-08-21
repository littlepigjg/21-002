package service

import (
	"strings"

	"summarizer/internal/model"
)

// FormatKeywords 将关键词列表格式化为逗号分隔的字符串。
func FormatKeywords(keywords []model.Keyword) string {
	words := make([]string, 0, len(keywords))
	for _, k := range keywords {
		words = append(words, k.Word)
	}
	return strings.Join(words, ", ")
}

// TopKeyword 返回得分最高的关键词，列表为空时返回空串。
func TopKeyword(keywords []model.Keyword) string {
	if len(keywords) == 0 {
		return ""
	}
	best := keywords[0]
	for _, k := range keywords[1:] {
		if k.Score > best.Score {
			best = k
		}
	}
	return best.Word
}

// KeywordWords 返回关键词的纯词列表。
func KeywordWords(keywords []model.Keyword) []string {
	words := make([]string, 0, len(keywords))
	for _, k := range keywords {
		words = append(words, k.Word)
	}
	return words
}
