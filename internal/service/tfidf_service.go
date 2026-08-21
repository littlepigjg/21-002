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

func (t *TfidfService) Extract(sentences []model.Sentence) []model.Keyword {
	totalDocs := len(sentences)
	if totalDocs == 0 {
		return nil
	}

	docFreq := make(map[string]int)
	totalTF := make(map[string]int)

	for _, s := range sentences {
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
	}

	type scored struct {
		word  string
		tf    int
		idf   float64
		score float64
	}

	scores := make([]scored, 0, len(totalTF))
	for word, tf := range totalTF {
		idf := 0.0
		if df := docFreq[word]; df > 0 {
			idf = math.Log(float64(totalDocs) / float64(df))
		}
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

	// 每次调用都分配独立切片，绝不复用全局缓冲。
	// 之前用包级 sharedKeywordBuf 复用同一块底层数组，导致每次 Extract
	// 都把前一次结果的 Keywords 原地清空并改写：后一篇文章的关键词会
	// 覆盖前面已保存结果的关键词（TF/score 全部串台）。让每个结果独占
	// 自己的切片，才能保证已保存的 Keywords 不被后续分析改动。
	keywords := make([]model.Keyword, 0, n)
	for idx, sc := range scores {
		if idx >= n {
			break
		}
		keywords = append(keywords, model.Keyword{
			Word:  sc.word,
			Score: sc.score,
			TF:    sc.tf,
			IDF:   sc.idf,
		})
	}
	return keywords
}
