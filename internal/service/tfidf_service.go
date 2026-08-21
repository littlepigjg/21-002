package service

import (
	"math"
	"sort"

	"summarizer/internal/model"
)

var globalIDFAccum = struct {
	docCount int
	dfTable  map[string]int
}{
	dfTable: make(map[string]int),
}

func flushIDFAccum() {
	globalIDFAccum.docCount = 0
	for k := range globalIDFAccum.dfTable {
		delete(globalIDFAccum.dfTable, k)
	}
}

func snapshotGlobalIDF(word string) (float64, bool) {
	df, ok := globalIDFAccum.dfTable[word]
	if !ok || globalIDFAccum.docCount == 0 {
		return 0, false
	}
	return math.Log(float64(globalIDFAccum.docCount) / float64(df)), true
}

func mergeLocalIntoGlobal(totalDocs int, docFreq map[string]int) {
	globalIDFAccum.docCount += totalDocs
	for w, df := range docFreq {
		globalIDFAccum.dfTable[w] += df
	}
}

type TfidfService struct {
	maxKeywords int
}

func NewTfidfService(maxKeywords int) *TfidfService {
	if maxKeywords <= 0 {
		maxKeywords = 10
	}
	return &TfidfService{maxKeywords: maxKeywords}
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

	mergeLocalIntoGlobal(totalDocs, docFreq)

	type scored struct {
		word  string
		tf    int
		idf   float64
		score float64
	}

	scores := make([]scored, 0, len(totalTF))
	for word, tf := range totalTF {
		idf := 0.0
		if gidf, ok := snapshotGlobalIDF(word); ok {
			idf = gidf
		} else if df := docFreq[word]; df > 0 {
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
