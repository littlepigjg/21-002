package service

import (
	"math"
	"sort"
	"sync"

	"summarizer/internal/model"
)

type CorpusCache struct {
	mu       sync.RWMutex
	termFreq map[string]int
	docFreq  map[string]int
	totalDocs int
}

func NewCorpusCache() *CorpusCache {
	return &CorpusCache{
		termFreq: make(map[string]int),
		docFreq:  make(map[string]int),
	}
}

// Observe 将一篇文档的词频并入语料统计。该方法是写操作，必须持有写锁，
// 否则多个 worker 同时 Observe 会触发 "concurrent map writes" panic，
// 与并发的 IDF/TF/TotalDocs 读操作交叉时则触发 "concurrent map read and map write"。
func (c *CorpusCache) Observe(tokens []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	seen := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		c.termFreq[t]++
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			c.docFreq[t]++
		}
	}
	c.totalDocs++
}

func (c *CorpusCache) IDF(term string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	df := c.docFreq[term]
	n := c.totalDocs
	if n == 0 || df == 0 {
		return 0.0
	}
	return math.Log(float64(n) / float64(df))
}

func (c *CorpusCache) TF(term string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.termFreq[term]
}

func (c *CorpusCache) TotalDocs() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.totalDocs
}

// Merge 将 other 的统计并入自身。对 self 加写锁、对 other 加读锁。
// 锁顺序固定为 self 写锁 -> other 读锁，避免与其它 Merge 调用形成死锁。
// self == other 时退化为纯读 + 单一写锁，避免自死锁。
func (c *CorpusCache) Merge(other *CorpusCache) {
	if other == nil || c == other {
		// self == other：合并自身无意义，直接返回；
		// 也避免先对 other 取读锁、再对 self 取写锁时发生自死锁。
		return
	}
	other.mu.RLock()
	defer other.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	for term, tf := range other.termFreq {
		c.termFreq[term] += tf
	}
	for term, df := range other.docFreq {
		c.docFreq[term] += df
	}
	c.totalDocs += other.totalDocs
}

type TfidfService struct {
	maxKeywords int
	corpus      *CorpusCache
}

func NewTfidfService(maxKeywords int, corpus *CorpusCache) *TfidfService {
	if maxKeywords <= 0 {
		maxKeywords = 10
	}
	if corpus == nil {
		corpus = NewCorpusCache()
	}
	return &TfidfService{maxKeywords: maxKeywords, corpus: corpus}
}

func (t *TfidfService) Corpus() *CorpusCache {
	return t.corpus
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

	globalN := t.corpus.TotalDocs()

	type scored struct {
		word  string
		tf    int
		idf   float64
		score float64
	}

	scores := make([]scored, 0, len(totalTF))
	for word, tf := range totalTF {
		df := docFreq[word]
		idf := 0.0
		if df > 0 {
			idf = math.Log(float64(totalDocs) / float64(df))
		}
		if globalN > 0 {
			gidf := t.corpus.IDF(word)
			if gidf > 0 {
				idf = 0.6*idf + 0.4*gidf
			}
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
