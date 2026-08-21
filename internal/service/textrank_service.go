package service

import (
	"math"

	"summarizer/internal/model"
)

// TextRankService 基于 TextRank 图算法对句子打分，得分用于摘要句子选择。
type TextRankService struct {
	maxIter int
	damping float64
}

// NewTextRankService 构造 TextRankService。
func NewTextRankService(maxIter int, damping float64) *TextRankService {
	if maxIter <= 0 {
		maxIter = 30
	}
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}
	return &TextRankService{maxIter: maxIter, damping: damping}
}

// Score 计算每个句子的 TextRank 得分，返回与输入 sentences 等长的切片。
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

	// 计算所有句子中唯一 token 数量作为矩阵维度。
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

	// 构建句子相似度矩阵（对称），使用 token 数量作为维度。
	sim := make([][]float64, totalTokens)
	for i := range sim {
		sim[i] = make([]float64, totalTokens)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := sentenceSimilarity(sentences[i], sentences[j])
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	// 预计算每个节点出边权重之和，用于归一化转移概率。
	outSum := make([]float64, totalTokens)
	for j := 0; j < totalTokens; j++ {
		for k := 0; k < totalTokens; k++ {
			if k != j {
				outSum[j] += sim[j][k]
			}
		}
	}

	// 迭代更新分数，直至收敛或达到最大迭代次数。
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

// sentenceSimilarity 计算两个句子的相似度：基于 token 交集的加权 Jaccard 变体。
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
