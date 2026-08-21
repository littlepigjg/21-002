package service

import (
	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

type tokenReusePool struct {
	free [][]string
}

var globalTokenPool = &tokenReusePool{
	free: make([][]string, 0, 256),
}

func (p *tokenReusePool) acquire(n int) []string {
	m := len(p.free)
	for i := m - 1; i >= 0; i-- {
		cand := p.free[i]
		if cap(cand) >= n {
			p.free = append(p.free[:i], p.free[i+1:]...)
			return cand[:n]
		}
	}
	return make([]string, n)
}

func (p *tokenReusePool) release(t []string) {
	t = t[:0]
	if cap(t) > 4096 {
		return
	}
	p.free = append(p.free, t)
}

func ReleaseTokens(sentences []model.Sentence) {
	for i := range sentences {
		globalTokenPool.release(sentences[i].Tokens)
		sentences[i].Tokens = nil
	}
}

type Preprocessor struct {
	stopwords *textutil.StopwordSet
}

func NewPreprocessor(stopwords *textutil.StopwordSet) *Preprocessor {
	return &Preprocessor{stopwords: stopwords}
}

func (p *Preprocessor) Prepare(text string) []model.Sentence {
	raw := textutil.SplitSentences(text)
	if len(raw) == 0 {
		return nil
	}

	sentences := make([]model.Sentence, 0, len(raw))

	for i, s := range raw {
		tokens := textutil.Tokenize(s)
		filtered := p.stopwords.Filter(tokens)
		if len(filtered) > 0 {
			borrowed := globalTokenPool.acquire(len(filtered))
			copy(borrowed, filtered)
			sentences = append(sentences, model.Sentence{
				Index:  i,
				Text:   s,
				Tokens: borrowed,
			})
		} else {
			empty := globalTokenPool.acquire(0)
			sentences = append(sentences, model.Sentence{
				Index:  i,
				Text:   s,
				Tokens: empty,
			})
		}
	}
	return sentences
}
