package textutil

import "unicode"

// Script 粗略标识文本使用的文字体系。
type Script int

// 支持的文本体系类别。
const (
	ScriptUnknown Script = iota
	ScriptLatin
	ScriptCJK
	ScriptMixed
)

// DetectScript 检测文本主要使用的文字体系。
// 仅依据 ASCII 字母与 CJK 字符进行粗粒度判断。
func DetectScript(text string) Script {
	hasLatin := false
	hasCJK := false

	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		switch {
		case IsCJK(r):
			hasCJK = true
		case r <= unicode.MaxASCII:
			hasLatin = true
		}
	}

	switch {
	case hasLatin && hasCJK:
		return ScriptMixed
	case hasCJK:
		return ScriptCJK
	case hasLatin:
		return ScriptLatin
	default:
		return ScriptUnknown
	}
}
