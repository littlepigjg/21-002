package model

import "errors"

// 领域层对外暴露的可识别错误，便于上层使用 errors.Is 判断。
var (
	ErrNotFound        = errors.New("resource not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrEmptyContent    = errors.New("article content is empty")
	ErrTooLarge        = errors.New("article content exceeds limit")
	ErrQueueFull       = errors.New("task queue is full")
	ErrConflict        = errors.New("resource already exists")
)
