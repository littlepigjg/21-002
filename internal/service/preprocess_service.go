package service

import (
	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

// 注意：这里曾有一个跨请求共享的 globalTokenPool（acquire/release）。
// 多 goroutine 并发写入 p.free 切片且无任何同步，会出现「同一底层数组被
// 两个请求同时取走」的竞态，进而导致相互覆盖、排序时 slice 越界。
// token 切片短生命周期、单请求私有，直接分配即可，不再做池化复用。

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
		// 每个请求独立分配自己的 token 切片，避免跨请求共享导致的并发竞态。
		toks := make([]string, len(filtered))
		copy(toks, filtered)
		sentences = append(sentences, model.Sentence{
			Index:  i,
			Text:   s,
			Tokens: toks,
		})
	}
	return sentences
}

