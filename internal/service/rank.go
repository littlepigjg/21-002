package service

import (
	"sort"

	"summarizer/internal/model"
)

// TopKeywords 按得分降序返回前 n 个关键词；n 超过数量时返回全部。
// 该函数会复制输入切片，不修改调用方数据。
func TopKeywords(keywords []model.Keyword, n int) []model.Keyword {
	if n <= 0 {
		return nil
	}
	cp := make([]model.Keyword, len(keywords))
	copy(cp, keywords)
	sort.SliceStable(cp, func(i, j int) bool {
		return cp[i].Score > cp[j].Score
	})
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

// TopIndices 返回 scores 中得分最高的 n 个下标，按得分降序排列。
func TopIndices(scores []float64, n int) []int {
	if n <= 0 {
		return nil
	}
	type item struct {
		idx   int
		score float64
	}
	items := make([]item, len(scores))
	for i, s := range scores {
		items[i] = item{idx: i, score: s}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})
	if n > len(items) {
		n = len(items)
	}
	out := make([]int, 0, n)
	for _, it := range items[:n] {
		out = append(out, it.idx)
	}
	return out
}

// NormalizeScores 将分数向量线性归一化到 [0,1] 区间。
// 当所有分数相同时统一归一化为 1。
func NormalizeScores(scores []float64) []float64 {
	if len(scores) == 0 {
		return nil
	}
	min, max := scores[0], scores[0]
	for _, s := range scores {
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}
	out := make([]float64, len(scores))
	if max == min {
		for i := range out {
			out[i] = 1.0
		}
		return out
	}
	for i, s := range scores {
		out[i] = (s - min) / (max - min)
	}
	return out
}
