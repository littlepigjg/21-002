package textutil

import "strings"

// SplitParagraphs 按空行将文本切分为段落，丢弃空段落。
func SplitParagraphs(text string) []string {
	text = Normalize(text)
	parts := strings.Split(text, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// SplitLines 按换行将文本切分为行，丢弃空行。
func SplitLines(text string) []string {
	text = Normalize(text)
	parts := strings.Split(text, "\n")
	out := make([]string, 0, len(parts))
	for _, l := range parts {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// CountSentences 统计文本中的句子数量。
func CountSentences(text string) int {
	return len(SplitSentences(text))
}

// FirstSentence 返回文本的第一个句子，空文本返回空字符串。
func FirstSentence(text string) string {
	sentences := SplitSentences(text)
	if len(sentences) == 0 {
		return ""
	}
	return sentences[0]
}

// LastSentence 返回文本的最后一个句子，空文本返回空字符串。
func LastSentence(text string) string {
	sentences := SplitSentences(text)
	if len(sentences) == 0 {
		return ""
	}
	return sentences[len(sentences)-1]
}
