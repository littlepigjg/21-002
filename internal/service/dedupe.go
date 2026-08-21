package service

import "summarizer/internal/model"

// DedupeKeywords 去除关键词列表中重复的词，保留首次出现的项。
func DedupeKeywords(keywords []model.Keyword) []model.Keyword {
	seen := make(map[string]struct{}, len(keywords))
	out := make([]model.Keyword, 0, len(keywords))
	for _, k := range keywords {
		if _, ok := seen[k.Word]; ok {
			continue
		}
		seen[k.Word] = struct{}{}
		out = append(out, k)
	}
	return out
}

// DedupeTokens 去除 token 列表中的重复项，保持原始顺序。
func DedupeTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
