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

// Clone 返回 a 的深拷贝。
// Article 字段均为值类型，无需重建内部切片；返回新独立实例供调用方安全修改。
func (a *Article) Clone() *Article {
	if a == nil {
		return nil
	}
	cp := *a
	return &cp
}
