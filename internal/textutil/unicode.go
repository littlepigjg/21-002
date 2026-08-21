package textutil

import (
	"strings"
	"unicode"
)

// FullWidthToHalfWidth 将常见全角字符转换为半角字符，用于文本规范化。
// 覆盖全角 ASCII（0xFF01-0xFF5E）与全角空格（0x3000）。
func FullWidthToHalfWidth(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			b.WriteRune(r - 0xFEE0)
		case r == 0x3000:
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RuneCount 返回字符串的字符（rune）数量，而非字节数。
func RuneCount(s string) int {
	return len([]rune(s))
}

// IsAllASCII 判断字符串是否全部由 ASCII 字符组成。
func IsAllASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// TrimPunct 去除字符串首尾的标点与空白字符。
func TrimPunct(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
}

// CompactSpace 将字符串内的连续空白折叠为单个空格，并去除首尾空白。
func CompactSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// LowerASCII 仅将 ASCII 大写字母转为小写，保持 CJK 与其它字符不变。
func LowerASCII(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
