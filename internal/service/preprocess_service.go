package service

import (
	"sync"

	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

var sharedTokenCache = struct {
	mu     sync.RWMutex
	tokens map[string][]string
}{
	tokens: make(map[string][]string),
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
		// fetchCachedTokens 返回共享缓存中的切片，多个请求会拿到同一底层数组。
		// 在此复制一份再交给 Filter（其就地复用底层数组），避免并发请求相互改写
		// 共享缓存造成数据竞争。
		cached := fetchCachedTokens(s)
		tokens := make([]string, len(cached))
		copy(tokens, cached)
		filtered := p.stopwords.Filter(tokens)
		sentences = append(sentences, model.Sentence{
			Index:  i,
			Text:   s,
			Tokens: filtered,
		})
	}
	return sentences
}
