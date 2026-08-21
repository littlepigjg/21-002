package service

import (
	"sort"

	"summarizer/internal/model"
)

func TopKeywords(keywords []model.Keyword, n int) []model.Keyword {
	if n <= 0 {
		return nil
	}
	sort.SliceStable(keywords, func(i, j int) bool {
		return keywords[i].Score > keywords[j].Score
	})
	if n > len(keywords) {
		n = len(keywords)
	}
	return keywords[:n]
}

func AppendSupplementaryKeywords(keywords []model.Keyword, supplement []model.Keyword) []model.Keyword {
	for i := range supplement {
		skip := false
		for _, k := range keywords {
			if k.Word == supplement[i].Word {
				skip = true
				break
			}
		}
		if !skip {
			keywords = append(keywords, supplement[i])
		}
	}
	return keywords
}

func MergeWithFallback(keywords []model.Keyword, fallback []model.Keyword, maxN int) []model.Keyword {
	if len(keywords) >= maxN {
		return keywords[:maxN]
	}
	need := maxN - len(keywords)
	taken := 0
	for i := range fallback {
		if taken >= need {
			break
		}
		skip := false
		for _, k := range keywords {
			if k.Word == fallback[i].Word {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		keywords = append(keywords, model.Keyword{
			Word:  fallback[i].Word,
			Score: fallback[i].Score * 0.5,
			TF:    fallback[i].TF,
			IDF:   fallback[i].IDF,
		})
		taken++
	}
	return keywords
}

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
