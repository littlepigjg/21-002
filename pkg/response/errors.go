package response

import (
	"errors"
	"net/http"

	"summarizer/internal/model"
)

// StatusFromError 将领域错误映射为对应的 HTTP 状态码。
// 未识别的错误一律视为内部错误。
func StatusFromError(err error) int {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrInvalidArgument),
		errors.Is(err, model.ErrEmptyContent),
		errors.Is(err, model.ErrTooLarge):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrQueueFull):
		return http.StatusTooManyRequests
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
