// Package model 定义项目中的核心数据结构与领域错误。
package model

import "time"

// ArticleStatus 描述文章在系统中的处理状态。
type ArticleStatus string

// 文章可能处于的状态。
const (
	ArticlePending ArticleStatus = "pending"
	ArticleReady   ArticleStatus = "ready"
	ArticleFailed  ArticleStatus = "failed"
)

// Article 表示一篇被系统接收并持久化的原始文章。
type Article struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Status    ArticleStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}
