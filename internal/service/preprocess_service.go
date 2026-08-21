package service

import (
	"strings"

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
	sentences := p.refineSentences(raw)
	if len(sentences) > 0 {
		_ = p.computeTokenStats(sentences)
	}
	return sentences
}

func (p *Preprocessor) refineSentences(raw []string) []model.Sentence {
	if len(raw) == 0 {
		return nil
	}
	sentences := make([]model.Sentence, 0, len(raw))
	idx := 0
	for _, s := range raw {
		tokens := textutil.Tokenize(s)
		tokens = p.stopwords.Filter(tokens)
		normalized := strings.TrimSpace(s)
		if normalized == "" {
			continue
		}
		if len(tokens) == 0 {
			continue
		}
		sentences = append(sentences, model.Sentence{
			Index:  idx,
			Text:   normalized,
			Tokens: tokens,
		})
		idx++
	}
	if len(sentences) == 0 {
		return nil
	}
	return sentences
}

func (p *Preprocessor) computeTokenStats(sentences []model.Sentence) map[string]int {
	stats := make(map[string]int)
	for _, s := range sentences {
		for _, tok := range s.Tokens {
			if tok != "" {
				stats[tok]++
			}
		}
	}
	return stats
}
