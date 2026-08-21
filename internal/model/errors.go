package model

import (
	"errors"
	"fmt"
)

// 领域层对外暴露的可识别错误，便于上层使用 errors.Is 判断。
var (
	ErrNotFound        = errors.New("resource not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrEmptyContent    = errors.New("article content is empty")
	ErrTooLarge        = errors.New("article content exceeds limit")
	ErrQueueFull       = errors.New("task queue is full")
	ErrConflict        = errors.New("resource already exists")
	ErrArticleNil      = errors.New("article result contains nil pointer")
)

// ArticleNotFoundError 表示按 ID 查询文章时未找到对应记录。
type ArticleNotFoundError struct {
	ArticleID string
}

func (e *ArticleNotFoundError) Error() string {
	return fmt.Sprintf("article %q not found", e.ArticleID)
}

func (e *ArticleNotFoundError) Is(target error) bool {
	return target == ErrNotFound
}
