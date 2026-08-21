package textutil

import "math"

// JaccardSimilarity 计算两个 token 序列的 Jaccard 相似度，结果位于 [0,1]。
func JaccardSimilarity(a, b []string) float64 {
	set := make(map[string]struct{}, len(a))
	for _, t := range a {
		set[t] = struct{}{}
	}

	inter := 0
	for _, t := range b {
		if _, ok := set[t]; ok {
			inter++
		}
	}

	union := len(set) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// CosineSimilarity 计算两个词频向量的余弦相似度。
func CosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	dot := 0.0
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}

	normA := 0.0
	for _, v := range a {
		normA += v * v
	}
	normB := 0.0
	for _, v := range b {
		normB += v * v
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// OverlapCoefficient 计算两个 token 序列的重叠系数：
// 交集大小除以较小集合的大小。
func OverlapCoefficient(a, b []string) float64 {
	set := make(map[string]struct{}, len(a))
	for _, t := range a {
		set[t] = struct{}{}
	}

	inter := 0
	for _, t := range b {
		if _, ok := set[t]; ok {
			inter++
		}
	}

	denom := len(set)
	if len(b) < denom {
		denom = len(b)
	}
	if denom == 0 {
		return 0
	}
	return float64(inter) / float64(denom)
}

// DiceCoefficient 计算两个 token 序列的 Dice 系数。
func DiceCoefficient(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(a))
	for _, t := range a {
		set[t] = struct{}{}
	}
	inter := 0
	for _, t := range b {
		if _, ok := set[t]; ok {
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(a)+len(b))
}
