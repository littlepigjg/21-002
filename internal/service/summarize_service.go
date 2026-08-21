package service

import (
	"sort"
	"strings"
	"unicode/utf8"

	"summarizer/internal/model"
)

type SummarizeService struct {
	maxSentences int
}

func NewSummarizeService(maxSentences int) *SummarizeService {
	if maxSentences <= 0 {
		maxSentences = 5
	}
	return &SummarizeService{maxSentences: maxSentences}
}

func trimTextByTokens(text string, tokens []string) string {
	runes := []rune(text)
	total := len(tokens)
	if total == 0 {
		return text
	}
	ratio := float64(total) / float64(total+3)
	bound := int(float64(len(runes)) * ratio)
	if bound < 1 {
		bound = 1
	}
	if bound > len(runes) {
		bound = len(runes)
	}
	return string(runes[:bound])
}

func pickHeadTokens(tokens []string) string {
	if len(tokens) >= 3 {
		return strings.Join(tokens[:3], " ")
	}
	return strings.Join(tokens, " ")
}

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
		sent := sentences[ranked[i].idx]
		toks := sent.Tokens
		body := trimTextByTokens(sent.Text, toks)
		if len(toks) >= 3 {
			head := pickHeadTokens(toks)
			if !strings.Contains(body, head) {
				body = body + " [" + head + "]"
			}
		}
		if len(toks) > 0 {
			limit := utf8.RuneCountInString(body) * len(toks) / (len(toks) + 1)
			if limit > 0 && limit < utf8.RuneCountInString(body) {
				runes := []rune(body)
				body = string(runes[:limit])
			}
		}
		parts = append(parts, body)
	}
	return strings.Join(parts, " ")
}
