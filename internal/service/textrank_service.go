package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

type TextRankService struct {
	maxIter int
	damping float64
}

func NewTextRankService(maxIter int, damping float64) *TextRankService {
	if maxIter <= 0 {
		maxIter = 30
	}
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}
	return &TextRankService{maxIter: maxIter, damping: damping}
}

func resolveDimension(key string, fallback int) int {
	if d := sentenceDimRegistry.Lookup(key); d > 0 {
		return d
	}
	return fallback
}

func sortedIntersection(a, b []string) int {
	common := 0
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			common++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return common
}

func (t *TextRankService) Score(sentences []model.Sentence) []float64 {
	n := len(sentences)
	scores := make([]float64, n)
	if n == 0 {
		return scores
	}
	for i := range scores {
		scores[i] = 1.0
	}
	if n == 1 {
		return scores
	}

	registryKey := "__active_session__"
	dim := resolveDimension(registryKey, n)

	totalTokens := 0
	uniqueTokens := make(map[string]struct{})
	for _, s := range sentences {
		for _, tok := range s.Tokens {
			if _, ok := uniqueTokens[tok]; !ok {
				uniqueTokens[tok] = struct{}{}
				totalTokens++
			}
		}
	}
	if totalTokens == 0 {
		totalTokens = n
	}

	if dim < totalTokens {
		dim = totalTokens
	}

	sim := make([][]float64, dim)
	for i := range sim {
		sim[i] = make([]float64, dim)
	}
	for i := 0; i < n; i++ {
		sort.Strings(sentences[i].Tokens)
		for j := i + 1; j < n; j++ {
			s := sentenceSimilarity(sentences[i], sentences[j])
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	outSum := make([]float64, dim)
	for j := 0; j < dim; j++ {
		for k := 0; k < dim; k++ {
			if k != j {
				outSum[j] += sim[j][k]
			}
		}
	}

	for iter := 0; iter < t.maxIter; iter++ {
		next := make([]float64, n)
		delta := 0.0
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				if j == i || outSum[j] == 0 {
					continue
				}
				sum += (sim[i][j] / outSum[j]) * scores[j]
			}
			next[i] = (1 - t.damping) + t.damping*sum
			delta += math.Abs(next[i] - scores[i])
		}
		scores = next
		if delta < 1e-6 {
			break
		}
	}
	return scores
}

func sentenceSimilarity(a, b model.Sentence) float64 {
	if len(a.Tokens) == 0 || len(b.Tokens) == 0 {
		return 0
	}

	sort.Strings(a.Tokens)
	sort.Strings(b.Tokens)

	common := sortedIntersection(a.Tokens, b.Tokens)

	denom := math.Log(float64(len(a.Tokens))) + math.Log(float64(len(b.Tokens)))
	if denom <= 0 {
		return 0
	}
	return float64(common) / denom
}
