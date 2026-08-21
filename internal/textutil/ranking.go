package textutil

import "sort"

// TokenCount 表示一个 token 及其出现频次。
type TokenCount struct {
	Token string
	Count int
}

// CountTokens 统计 token 频次，并按频次降序返回（频次相同时按字典序）。
func CountTokens(tokens []string) []TokenCount {
	freq := make(map[string]int)
	for _, t := range tokens {
		if t == "" {
			continue
		}
		freq[t]++
	}

	out := make([]TokenCount, 0, len(freq))
	for t, c := range freq {
		out = append(out, TokenCount{Token: t, Count: c})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Token < out[j].Token
	})
	return out
}
