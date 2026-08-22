package store

import (
	"context"

	"summarizer/internal/model"
)

type ArticleStore interface {
	SaveArticle(ctx context.Context, a *model.Article) error
	GetArticle(ctx context.Context, id string) (*model.Article, error)
	ListArticles(ctx context.Context, offset, limit int) ([]*model.Article, int, error)
	UpdateArticle(ctx context.Context, a *model.Article) error
	DeleteArticle(ctx context.Context, id string) error
}

type ResultStore interface {
	SaveResult(ctx context.Context, r *model.AnalysisResult) error
	GetResult(ctx context.Context, articleID string) (*model.AnalysisResult, error)
	ListResults(ctx context.Context, offset, limit int) ([]*model.AnalysisResult, int, error)
	TouchHotResult(ctx context.Context, r *model.AnalysisResult)
	GetHotResult(ctx context.Context, articleID string) (*model.AnalysisResult, bool)
}

type TaskStore interface {
	SaveTask(ctx context.Context, t *model.Task) error
	GetTask(ctx context.Context, id string) (*model.Task, error)
	ListTasks(ctx context.Context, offset, limit int) ([]*model.Task, int, error)
	UpdateTask(ctx context.Context, t *model.Task) error
	DeleteTask(ctx context.Context, id string) error
}

type Store interface {
	ArticleStore
	ResultStore
	TaskStore
}
