package textutil

import "strings"

// TokenizeSentence 对单个句子分词并去除停用词。
func TokenizeSentence(sentence string, stopwords *StopwordSet) []string {
	tokens := Tokenize(sentence)
	return stopwords.Filter(tokens)
}

// TokenizeWithNGrams 返回原始 token 及其二元组拼接结果。
func TokenizeWithNGrams(text string) []string {
	tokens := Tokenize(text)
	ngrams := Bigrams(tokens)
	return append(tokens, ngrams...)
}

// UniqueTokens 返回去重后的 token，保持首次出现顺序。
func UniqueTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// JoinTokens 以指定分隔符拼接 token。
func JoinTokens(tokens []string, sep string) string {
	return strings.Join(tokens, sep)
}
