package model

import "strings"

// MaxTitleLength 是标题允许的最大字符数。
const MaxTitleLength = 200

// ValidateTitle 校验标题长度是否合法。
func ValidateTitle(title string) error {
	if len([]rune(strings.TrimSpace(title))) > MaxTitleLength {
		return ErrInvalidArgument
	}
	return nil
}

// IsBlank 判断字符串是否为空或仅包含空白字符。
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
