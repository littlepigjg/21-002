// Package textutil 提供文本处理所需的底层工具：字符串清洗、分句、分词、
// 停用词过滤以及文件读写。
package textutil

import (
	"strings"
	"unicode"
)

// Normalize 对文本做基础清洗：统一换行符、折叠连续空白、丢弃控制字符，
// 并去除首尾空白。
func Normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		switch {
		case unicode.IsControl(r) && r != '\n':
			// 丢弃除换行外的控制字符。
		case r == '\n':
			b.WriteByte('\n')
			inSpace = false
		case unicode.IsSpace(r):
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		default:
			b.WriteRune(r)
			inSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// SplitSentences 将文本按中英文句末标点与换行切分为句子，并丢弃空句。
// 简化实现不对小数点、缩写等特殊场景做消歧。
func SplitSentences(text string) []string {
	text = Normalize(text)

	sentences := make([]string, 0, 16)
	var cur strings.Builder

	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			sentences = append(sentences, s)
		}
		cur.Reset()
	}

	for _, r := range text {
		switch r {
		case '。', '！', '？', '!', '?', ';', '；', '.', '\n':
			cur.WriteRune(r)
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return sentences
}

// IsCJK 判断 rune 是否属于 CJK 统一表意文字区间。
func IsCJK(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

// HasCJK 判断字符串是否包含中文字符。
func HasCJK(s string) bool {
	for _, r := range s {
		if IsCJK(r) {
			return true
		}
	}
	return false
}

// Truncate 将字符串截断到最多 maxRunes 个字符，超长时追加省略号。
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) <= maxRunes {
		return s
	}
	return string(rs[:maxRunes]) + "…"
}
