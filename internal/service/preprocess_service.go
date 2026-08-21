package service

import (
	"summarizer/internal/model"
	"summarizer/internal/textutil"
)

// Preprocessor 负责文本预处理：分句、分词与停用词过滤。
type Preprocessor struct {
	stopwords *textutil.StopwordSet
}

// NewPreprocessor 构造 Preprocessor。
func NewPreprocessor(stopwords *textutil.StopwordSet) *Preprocessor {
	return &Preprocessor{stopwords: stopwords}
}

// Prepare 将原文切分为句子，并对每个句子完成分词与停用词过滤。
// 返回的句子保留其在原文中的相对顺序。
func (p *Preprocessor) Prepare(text string) []model.Sentence {
	raw := textutil.SplitSentences(text)
	if len(raw) == 0 {
		return nil
	}

	sentences := make([]model.Sentence, 0, len(raw))

	for i, s := range raw {
		tokens := textutil.Tokenize(s)
		tokens = p.stopwords.Filter(tokens)
		sentences = append(sentences, model.Sentence{
			Index:  i,
			Text:   s,
			Tokens: tokens,
		})
	}
	return sentences
}
