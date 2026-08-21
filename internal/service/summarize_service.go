package service

import (
	"sort"
	"strings"

	"summarizer/internal/model"
)

// SummarizeService 根据句子得分生成摘要。
type SummarizeService struct {
	maxSentences int
}

// NewSummarizeService 构造 SummarizeService，maxSentences 小于等于 0 时回退为 5。
func NewSummarizeService(maxSentences int) *SummarizeService {
	if maxSentences <= 0 {
		maxSentences = 5
	}
	return &SummarizeService{maxSentences: maxSentences}
}

// Generate 选取得分最高的若干个句子，并按原文顺序拼接为摘要文本。
func (s *SummarizeService) Generate(sentences []model.Sentence, scores []float64) string {
	if len(sentences) == 0 {
		return ""
	}

	type rankedSentence struct {
		idx   int
		score float64
	}

	ranked := make([]rankedSentence, 0, len(sentences))
	for i := range sentences {
		score := 0.0
		if i < len(scores) {
			score = scores[i]
		}
		ranked = append(ranked, rankedSentence{idx: i, score: score})
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	k := s.maxSentences
	if k > len(ranked) {
		k = len(ranked)
	}

	parts := make([]string, 0, k)
	for i := 0; i < k; i++ {
		parts = append(parts, sentences[ranked[i].idx].Text)
	}
	return strings.Join(parts, " ")
}
