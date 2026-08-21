package service

import (
	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

// SentenceTokenOverlap 计算两个句子之间的 token 重叠数量。
func SentenceTokenOverlap(a, b model.Sentence) int {
	set := make(map[string]struct{}, len(a.Tokens))
	for _, t := range a.Tokens {
		set[t] = struct{}{}
	}
	overlap := 0
	for _, t := range b.Tokens {
		if _, ok := set[t]; ok {
			overlap++
		}
	}
	return overlap
}

// SentencesJaccard 计算两个句子 token 集合的 Jaccard 相似度。
func SentencesJaccard(a, b model.Sentence) float64 {
	return textutil.JaccardSimilarity(a.Tokens, b.Tokens)
}

// DocumentSimilarity 计算两篇文章预处理后 token 集合的余弦相似度。
func DocumentSimilarity(tokensA, tokensB []string) float64 {
	return textutil.CosineSimilarity(tokenFreq(tokensA), tokenFreq(tokensB))
}

// tokenFreq 将 token 列表转换为词频向量。
func tokenFreq(tokens []string) map[string]float64 {
	freq := make(map[string]float64)
	for _, t := range tokens {
		freq[t]++
	}
	return freq
}
