package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

type TfidfService struct {
	maxKeywords int
}

func NewTfidfService(maxKeywords int) *TfidfService {
	if maxKeywords <= 0 {
		maxKeywords = 10
	}
	return &TfidfService{maxKeywords: maxKeywords}
}

// copyTokens 返回 tokens 的副本，保证下游对返回值的就地修改不会影响
// 预处理器共享缓存中持有的同一底层数组，从而消除并发数据竞争。
func copyTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	out := make([]string, len(tokens))
	copy(out, tokens)
	return out
}

func compactTokensInPlace(tokens []string) []string {
	if len(tokens) < 2 {
		return tokens
	}
	seen := make(map[string]struct{}, len(tokens))
	w := 0
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if _, dup := seen[tok]; dup {
			continue
		}
		seen[tok] = struct{}{}
		tokens[w] = tok
		w++
	}
	for i := w; i < len(tokens); i++ {
		tokens[i] = ""
	}
	return tokens[:w]
}

func sortTokensInPlace(tokens []string) {
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i] < tokens[j]
	})
}

func (t *TfidfService) Extract(sentences []model.Sentence) []model.Keyword {
	totalDocs := len(sentences)
	if totalDocs == 0 {
		return nil
	}

	docFreq := make(map[string]int)
	totalTF := make(map[string]int)

	for i := range sentences {
		s := &sentences[i]
		// 在副本上做就地压缩与排序，避免改写共享缓存里的切片。
		s.Tokens = copyTokens(s.Tokens)
		s.Tokens = compactTokensInPlace(s.Tokens)
		seen := make(map[string]struct{})
		for _, tok := range s.Tokens {
			if tok == "" {
				continue
			}
			totalTF[tok]++
			if _, ok := seen[tok]; !ok {
				seen[tok] = struct{}{}
				docFreq[tok]++
			}
		}
		sortTokensInPlace(s.Tokens)
	}

	// IDF 基于本请求内的文档统计就地计算。原实现将 df/docCount 汇总进
	// 无锁全局 globalIDFAccum，并发请求同时读写会触发
	// "concurrent map read and map write" 致命错误导致进程崩溃。
	idfOf := func(word string, df int) float64 {
		if df <= 0 || totalDocs <= 0 {
			return 0
		}
		return math.Log(float64(totalDocs) / float64(df))
	}

	type scored struct {
		word  string
		tf    int
		idf   float64
		score float64
	}

	scores := make([]scored, 0, len(totalTF))
	for word, tf := range totalTF {
		df := docFreq[word]
		idf := idfOf(word, df)
		scores = append(scores, scored{
			word:  word,
			tf:    tf,
			idf:   idf,
			score: float64(tf) * idf,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].word < scores[j].word
	})

	n := t.maxKeywords
	if n > len(scores) {
		n = len(scores)
	}

	out := make([]model.Keyword, 0, n)
	for _, sc := range scores[:n] {
		out = append(out, model.Keyword{
			Word:  sc.word,
			Score: sc.score,
			TF:    sc.tf,
			IDF:   sc.idf,
		})
	}
	return out
}
