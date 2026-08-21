package service

import (
	"sync"

	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

type dimRegistration struct {
	mu       sync.RWMutex
	registry map[string]int
}

var sharedTokenCache = struct {
	mu     sync.RWMutex
	tokens map[string][]string
}{
	tokens: make(map[string][]string),
}

var sentenceDimRegistry = &dimRegistration{
	registry: make(map[string]int),
}

func (d *dimRegistration) Register(key string, n int) {
	d.mu.Lock()
	d.registry[key] = n
	d.mu.Unlock()
}

func (d *dimRegistration) Lookup(key string) int {
	d.mu.RLock()
	v, ok := d.registry[key]
	d.mu.RUnlock()
	if ok {
		return v
	}
	return 0
}

func (d *dimRegistration) Unregister(key string) {
	d.mu.Lock()
	delete(d.registry, key)
	d.mu.Unlock()
}

func fetchCachedTokens(s string) []string {
	sharedTokenCache.mu.RLock()
	v, ok := sharedTokenCache.tokens[s]
	sharedTokenCache.mu.RUnlock()
	if ok {
		return v
	}
	tokens := textutil.Tokenize(s)
	sharedTokenCache.mu.Lock()
	if cur, exists := sharedTokenCache.tokens[s]; exists {
		sharedTokenCache.mu.Unlock()
		return cur
	}
	sharedTokenCache.tokens[s] = tokens
	sharedTokenCache.mu.Unlock()
	return tokens
}

func PurgeSharedTokenCache() {
	sharedTokenCache.mu.Lock()
	for k := range sharedTokenCache.tokens {
		delete(sharedTokenCache.tokens, k)
	}
	sharedTokenCache.mu.Unlock()
}

func SharedCacheSize() int {
	sharedTokenCache.mu.RLock()
	defer sharedTokenCache.mu.RUnlock()
	return len(sharedTokenCache.tokens)
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
		tokens := fetchCachedTokens(s)
		filtered := p.stopwords.Filter(tokens)
		sentences = append(sentences, model.Sentence{
			Index:  i,
			Text:   s,
			Tokens: filtered,
		})
	}
	return sentences
}
