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

// ArticleResult 是存储层对单篇文章查询结果的封装接口，
// 将数据与错误信息统一为一个可返回的值。
type ArticleResult interface {
	GetArticle() *Article
	GetError() error
	IsReady() bool
	HasArticle() bool
}

type articleQueryResult struct {
	article *Article
	err     error
}

func NewArticleResult(a *Article, err error) ArticleResult {
	return &articleQueryResult{article: a, err: err}
}

func (r *articleQueryResult) GetArticle() *Article {
	return r.article
}

func (r *articleQueryResult) GetError() error {
	return r.err
}

func (r *articleQueryResult) IsReady() bool {
	return r.article != nil && r.article.Status == ArticleReady
}

func (r *articleQueryResult) HasArticle() bool {
	return r.article != nil
}
