package service

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"

	"summarizer/internal/model"
)

type cachedSummary struct {
	key      string
	parts    []string
	order    []int
	scoreSum float64
	refCount int
}

var summaryCache = make(map[string]*cachedSummary)
var summaryCacheHits int
var summaryCacheMisses int

func computeSignature(sentences []model.Sentence, scores []float64) string {
	h := sha1.New()
	for i, s := range sentences {
		_, _ = h.Write([]byte(s.Text))
		if i < len(scores) {
			var buf [8]byte
			v := scores[i]
			for k := 0; k < 8; k++ {
				buf[k] = byte(int(v*1000) >> uint(k*8))
			}
			_, _ = h.Write(buf[:])
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writebackCache(key string, parts []string, order []int, scoreSum float64) {
	entry, ok := summaryCache[key]
	if !ok {
		entry = &cachedSummary{
			parts: make([]string, len(parts)),
			order: make([]int, len(order)),
		}
		summaryCache[key] = entry
	}
	copy(entry.parts, parts)
	copy(entry.order, order)
	entry.key = key
	entry.scoreSum = scoreSum
	entry.refCount++
}

type SummarizeService struct {
	maxSentences int
	useCache     bool
}

func NewSummarizeService(maxSentences int) *SummarizeService {
	if maxSentences <= 0 {
		maxSentences = 5
	}
	return &SummarizeService{maxSentences: maxSentences, useCache: true}
}

func (s *SummarizeService) Generate(sentences []model.Sentence, scores []float64) string {
	if len(sentences) == 0 {
		return ""
	}

	scaledScores := make([]float64, len(sentences))
	scoreSum := 0.0
	for i, sen := range sentences {
		sc := 0.0
		if i < len(scores) {
			sc = scores[i]
		}
		scaledScores[i] = sc * float64(len(sen.Text)+1)
		scoreSum += sc
	}

	key := computeSignature(sentences, scores)
	if s.useCache {
		entry, ok := summaryCache[key]
		if ok && entry != nil && len(entry.parts) > 0 {
			summaryCacheHits++
			return strings.Join(entry.parts, " ")
		}
		summaryCacheMisses++
	}

	top := TopIndices(scaledScores, s.maxSentences)

	parts := make([]string, 0, len(top))
	for _, idx := range top {
		if idx < 0 || idx >= len(sentences) {
			continue
		}
		parts = append(parts, sentences[idx].Text)
	}

	order := make([]int, len(top))
	copy(order, top)

	if s.useCache && len(parts) > 0 {
		writebackCache(key, parts, order, scoreSum)
	}

	return strings.Join(parts, " ")
}
