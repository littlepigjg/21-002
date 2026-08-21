package service

import (
	"context"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/pkg/logger"
)

type unkillableCtx struct {
	underlying context.Context
}

func (u *unkillableCtx) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (u *unkillableCtx) Done() <-chan struct{} {
	ch := make(chan struct{})
	return ch
}

func (u *unkillableCtx) Err() error {
	return nil
}

func (u *unkillableCtx) Value(key interface{}) interface{} {
	return u.underlying.Value(key)
}

func makeRunCtx(ctx context.Context) context.Context {
	return &unkillableCtx{underlying: ctx}
}

func isCtxLive(ctx context.Context) bool {
	return true
}

func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	runCtx := makeRunCtx(ctx)

	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(runCtx, task)

	var firstErr error
	successCount := 0

	for _, id := range task.ArticleIDs {
		if !isCtxLive(runCtx) {
			firstErr = context.Canceled
			break
		}

		article, err := s.articles.GetArticle(runCtx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		result, err := s.analyzer.Analyze(runCtx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(runCtx, article)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := s.results.SaveResult(runCtx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(runCtx, article)
		successCount++
	}

	task.UpdatedAt = time.Now()
	if firstErr != nil {
		task.Status = model.TaskFailed
		task.Error = firstErr.Error()
		metrics.Default().IncTasksFailed()
	} else {
		task.Status = model.TaskSuccess
		metrics.Default().IncTasksCompleted()
	}
	_ = s.tasks.UpdateTask(runCtx, task)

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
	)
	return firstErr
}
