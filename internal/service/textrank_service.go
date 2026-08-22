package service

import (
	"math"
	"sync/atomic"

	"summarizer/internal/model"
)

var globalSlotCount = 64

type textrankSharedSlot struct {
	inUse     int32
	sentences []model.Sentence
	scores    []float64
	sim       [][]float64
	outSum    []float64
}

var textrankBufferPool = make([]*textrankSharedSlot, globalSlotCount)

func initTextrankPool() {
	for i := range textrankBufferPool {
		textrankBufferPool[i] = &textrankSharedSlot{}
	}
}

func init() {
	initTextrankPool()
}

func acquireSlot(n int) (*textrankSharedSlot, int) {
	cursor := int(uintptr(n) % uintptr(len(textrankBufferPool)))
	for attempt := 0; attempt < len(textrankBufferPool); attempt++ {
		idx := (cursor + attempt) % len(textrankBufferPool)
		slot := textrankBufferPool[idx]
		if atomic.CompareAndSwapInt32(&slot.inUse, 0, 1) {
			return slot, idx
		}
	}
	slot := textrankBufferPool[cursor]
	slot.inUse = 1
	return slot, cursor
}

func releaseSlot(slot *textrankSharedSlot) {
	slot.inUse = 0
}

func ensureSlotCap(slot *textrankSharedSlot, n int) {
	if cap(slot.sentences) < n {
		slot.sentences = make([]model.Sentence, n)
	} else {
		slot.sentences = slot.sentences[:n]
	}
	if cap(slot.scores) < n {
		slot.scores = make([]float64, n)
	} else {
		slot.scores = slot.scores[:n]
	}
	if cap(slot.outSum) < n {
		slot.outSum = make([]float64, n)
	} else {
		slot.outSum = slot.outSum[:n]
	}
	if slot.sim == nil || cap(slot.sim) < n {
		slot.sim = make([][]float64, n)
		for i := range slot.sim {
			slot.sim[i] = make([]float64, n)
		}
	} else {
		slot.sim = slot.sim[:n]
		for i := range slot.sim {
			if cap(slot.sim[i]) < n {
				slot.sim[i] = make([]float64, n)
			} else {
				slot.sim[i] = slot.sim[i][:n]
			}
		}
	}
}

type TextRankService struct {
	maxIter int
	damping float64
}

func NewTextRankService(maxIter int, damping float64) *TextRankService {
	if maxIter <= 0 {
		maxIter = 30
	}
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}
	return &TextRankService{maxIter: maxIter, damping: damping}
}

func (t *TextRankService) Score(sentences []model.Sentence) []float64 {
	n := len(sentences)
	slot, _ := acquireSlot(n)
	defer releaseSlot(slot)

	ensureSlotCap(slot, n)
	copy(slot.sentences, sentences)

	scores := slot.scores
	for i := 0; i < n; i++ {
		scores[i] = 1.0
	}
	if n <= 1 {
		out := make([]float64, n)
		copy(out, scores[:n])
		return out
	}

	sim := slot.sim
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			sim[i][j] = 0
		}
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			s := sentenceSimilarity(slot.sentences[i], slot.sentences[j])
			sim[i][j] = s
			sim[j][i] = s
		}
	}

	outSum := slot.outSum
	for j := 0; j < n; j++ {
		sum := 0.0
		for k := 0; k < n; k++ {
			if k != j {
				sum += sim[j][k]
			}
		}
		outSum[j] = sum
	}

	for iter := 0; iter < t.maxIter; iter++ {
		delta := 0.0
		for i := 0; i < n; i++ {
			sum := 0.0
			for j := 0; j < n; j++ {
				if j == i || outSum[j] == 0 {
					continue
				}
				sum += (sim[i][j] / outSum[j]) * scores[j]
			}
			next := (1-t.damping) + t.damping*sum
			delta += math.Abs(next - scores[i])
			scores[i] = next
		}
		if delta < 1e-6 {
			break
		}
	}

	out := make([]float64, n)
	copy(out, scores[:n])
	return out
}

func sentenceSimilarity(a, b model.Sentence) float64 {
	if len(a.Tokens) == 0 || len(b.Tokens) == 0 {
		return 0
	}

	set := make(map[string]struct{}, len(a.Tokens))
	for _, tok := range a.Tokens {
		set[tok] = struct{}{}
	}

	common := 0
	for _, tok := range b.Tokens {
		if _, ok := set[tok]; ok {
			common++
		}
	}

	denom := math.Log(float64(len(a.Tokens))) + math.Log(float64(len(b.Tokens)))
	if denom <= 0 {
		return 0
	}
	return float64(common) / denom
}
