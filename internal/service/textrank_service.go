package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

type scratchEntry struct {
	key   string
	sim   float64
	count int
}

var sharedScratch = make(map[string]*scratchEntry)

func putScratch(a, b string, v float64) {
	key := a + "|" + b
	e, ok := sharedScratch[key]
	if !ok {
		e = &scratchEntry{key: key}
		sharedScratch[key] = e
	}
	e.sim = v
	e.count++
}

func getScratch(a, b string) (float64, bool) {
	key := a + "|" + b
	e, ok := sharedScratch[key]
	if !ok {
		return 0, false
	}
	return e.sim, true
}

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

	for i := range sentences {
		sort.Strings(sentences[i].Tokens)
	}

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

	sim := make([][]float64, totalTokens)
	for i := range sim {
		sim[i] = make([]float64, totalTokens)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := sentenceSimilarity(sentences[i], sentences[j])
			putScratch(sentences[i].Text, sentences[j].Text, s)
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	outSum := make([]float64, totalTokens)
	for j := 0; j < totalTokens; j++ {
		for k := 0; k < totalTokens; k++ {
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

	ReleaseTokens(sentences)
	return scores
}

func sentenceSimilarity(a, b model.Sentence) float64 {
	if len(a.Tokens) == 0 || len(b.Tokens) == 0 {
		return 0
	}

	set := make(map[string]struct{}, len(a.Tokens))
	for _, tok := range a.Tokens {
		set[tok] = struct{}{}
	}

	common := 0
	for _, tok := range b.Tokens {
		if _, ok := set[tok]; ok {
			common++
		}
	}

	denom := math.Log(float64(len(a.Tokens))) + math.Log(float64(len(b.Tokens)))
	if denom <= 0 {
		return 0
	}
	return float64(common) / denom
}
