package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

const sharedBufCap = 256

var sharedKeywordBuf = make([]model.Keyword, 0, sharedBufCap)

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

	sharedKeywordBuf = sharedKeywordBuf[:0]
	for idx, sc := range scores {
		if idx >= n {
			break
		}
		sharedKeywordBuf = append(sharedKeywordBuf, model.Keyword{
			Word:  sc.word,
			Score: sc.score,
			TF:    sc.tf,
			IDF:   sc.idf,
		})
	}
	return sharedKeywordBuf
}
