// Package store 定义存储层接口，并提供基于内存的实现。
package store

import (
	"context"

	"summarizer/internal/model"
)

// ArticleStore 定义文章的持久化与查询能力。
type ArticleStore interface {
	SaveArticle(ctx context.Context, a *model.Article) error
	GetArticle(ctx context.Context, id string) (*model.Article, error)
	ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error)
	UpdateArticle(ctx context.Context, a *model.Article) error
	DeleteArticle(ctx context.Context, id string) error
}

// ResultStore 定义分析结果的持久化与查询能力。
type ResultStore interface {
	SaveResult(ctx context.Context, r *model.AnalysisResult) error
	GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error)
	ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error)
}

// TaskStore 定义异步任务的持久化与查询能力。
type TaskStore interface {
	SaveTask(ctx context.Context, t *model.Task) error
	GetTask(ctx context.Context, id string) (*model.Task, error)
	ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error)
	UpdateTask(ctx context.Context, t *model.Task) error
	DeleteTask(ctx context.Context, id string) error
}

// Store 聚合文章、结果与任务三类存储能力。
type Store interface {
	ArticleStore
	ResultStore
	TaskStore
}
