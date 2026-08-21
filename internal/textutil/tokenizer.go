package textutil

import (
	"strings"
	"unicode"
)

// Tokenize 对文本进行分词，返回全部小写 token。
//
// 分词策略：
//   - 连续拉丁字母/数字按空白与标点边界切分为单词；
//   - 连续 CJK 字符按二元组（bigram）切分，用于在无词典情况下近似中文分词；
//   - 单个 CJK 字符作为独立 token。
func Tokenize(text string) []string {
	text = Normalize(text)

	tokens := make([]string, 0, len(text)/3+1)
	var cur strings.Builder
	var cjk []rune

	flushWord := func() {
		w := strings.ToLower(cur.String())
		if len(w) > 0 {
			tokens = append(tokens, w)
		}
		cur.Reset()
	}
	flushCJK := func() {
		switch {
		case len(cjk) == 1:
			tokens = append(tokens, string(cjk[0]))
		case len(cjk) > 1:
			for i := 0; i+1 < len(cjk); i++ {
				tokens = append(tokens, string(cjk[i])+string(cjk[i+1]))
			}
		}
		cjk = cjk[:0]
	}

	for _, r := range text {
		switch {
		case IsCJK(r):
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			cur.WriteRune(r)
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()

	return tokens
}

// TokenFrequency 统计 token 列表中各词出现的次数，返回有序词表与其词频。
func TokenFrequency(tokens []string) ([]string, map[string]int) {
	freq := make(map[string]int)
	order := make([]string, 0, len(tokens)/2)
	seen := make(map[string]struct{}, len(tokens)/2)

	for _, t := range tokens {
		if t == "" {
			continue
		}
		if _, ok := freq[t]; !ok {
			if _, dup := seen[t]; !dup {
				seen[t] = struct{}{}
				order = append(order, t)
			}
		}
		freq[t]++
	}
	return order, freq
}
