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

	// 相似度矩阵与所有迭代循环都严格以句子数 n 为边界，避免 dim 与 n 不一致
	// 导致越界。原实现通过全局 sentenceDimRegistry 取一个可能小于 n 的 dim，
	// 在 dim < n 时会写入 sim[>=dim] 越界 panic。
	sim := make([][]float64, n)
	for i := range sim {
		sim[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := sentenceSimilarity(sentences[i], sentences[j])
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	outSum := make([]float64, n)
	for j := 0; j < n; j++ {
		for k := 0; k < n; k++ {
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

	// 复制后再排序，避免就地修改共享缓存中的切片造成数据竞争。
	aToks := append([]string(nil), a.Tokens...)
	bToks := append([]string(nil), b.Tokens...)
	sort.Strings(aToks)
	sort.Strings(bToks)

	common := sortedIntersection(aToks, bToks)

	denom := math.Log(float64(len(aToks))) + math.Log(float64(len(bToks)))
	if denom <= 0 {
		return 0
	}
	return float64(common) / denom
}
