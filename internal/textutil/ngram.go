package textutil

import "strings"

// Ngrams 从 token 序列生成连续的 n 元组。
// 当 n 小于等于 0，或 token 数量不足 n 时返回 nil。
func Ngrams(tokens []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(tokens) < n {
		return nil
	}
	out := make([]string, 0, len(tokens)-n+1)
	for i := 0; i+n <= len(tokens); i++ {
		out = append(out, joinTokens(tokens[i:i+n]))
	}
	return out
}

// Bigrams 生成二元组（bigram）。
func Bigrams(tokens []string) []string {
	return Ngrams(tokens, 2)
}

// Trigrams 生成三元组（trigram）。
func Trigrams(tokens []string) []string {
	return Ngrams(tokens, 3)
}

// joinTokens 将 token 无分隔地拼接为一个字符串，适用于 CJK 分词场景。
func joinTokens(tokens []string) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t)
	}
	return b.String()
}
