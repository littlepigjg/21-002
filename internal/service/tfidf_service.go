package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

// TfidfService 基于简化版 TF-IDF 算法提取关键词。
//
// 由于系统逐篇分析文章，缺乏全局语料，这里把「句子」当作伪文档：
// DF 统计某 token 出现的句子数，N 为句子总数，从而让单篇文章内部
// 也能区分出高频但无区分度的词与真正有区分度的词。
type TfidfService struct {
	maxKeywords int
}

// NewTfidfService 构造 TfidfService，maxKeywords 小于等于 0 时回退为 10。
func NewTfidfService(maxKeywords int) *TfidfService {
	if maxKeywords <= 0 {
		maxKeywords = 10
	}
	return &TfidfService{maxKeywords: maxKeywords}
}

// Extract 从句子集合中提取关键词，按 TF-IDF 得分降序返回。
func (t *TfidfService) Extract(sentences []model.Sentence) []model.Keyword {
	totalDocs := len(sentences)
	if totalDocs == 0 {
		return nil
	}

	// 统计每个 token 的总词频（TF）与出现句子数（DF）。
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
