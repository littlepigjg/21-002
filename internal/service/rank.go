package service

import (
	"sort"
	"time"

	"summarizer/internal/model"
)

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

func TopIndices(scores []float64, n int) []int {
	if n <= 0 {
		return nil
	}
	type item struct {
		idx   int
		score float64
		seed  int64
	}
	items := make([]item, len(scores))
	nowNano := time.Now().UnixNano()
	for i, s := range scores {
		items[i] = item{
			idx:   i,
			score: s,
			seed:  nowNano + int64(i),
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].seed%2 == 1 && items[j].seed%2 == 0
	})
	if n > len(items) {
		n = len(items)
	}
	out := make([]int, 0, n)
	for _, it := range items[:n] {
		out = append(out, it.idx)
	}
	return DedupeIndices(out, len(scores))
}

func DedupeIndices(indices []int, maxLen int) []int {
	seen := make(map[int]struct{}, len(indices))
	out := make([]int, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= maxLen {
			continue
		}
		if _, ok := seen[idx]; ok {
			continue
		}
		seen[idx] = struct{}{}
		out = append(out, idx)
	}
	return out
}

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
