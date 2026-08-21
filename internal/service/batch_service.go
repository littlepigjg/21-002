package service

import (
	"context"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/pkg/logger"
)

func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0

	for _, id := range task.ArticleIDs {
		task.UpdatedAt = time.Now()

		if ctx.Err() != nil {
			firstErr = ctx.Err()
			task.Error = firstErr.Error()
			break
		}

		_ = s.tasks.UpdateProgress(ctx, task.ID, id, "fetching")

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			_ = s.tasks.UpdateProgress(ctx, task.ID, id, "missing")
			task.Error = firstErr.Error()
			continue
		}

		_ = s.tasks.UpdateProgress(ctx, task.ID, id, "analyzing")

		result, err := s.analyzer.Analyze(ctx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(ctx, article)
			if firstErr == nil {
				firstErr = err
			}
			_ = s.tasks.UpdateProgress(ctx, task.ID, id, "failed")
			task.Error = firstErr.Error()
			_ = s.tasks.IncrementProcessed(ctx)
			continue
		}

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			_ = s.tasks.UpdateProgress(ctx, task.ID, id, "failed")
			task.Error = firstErr.Error()
			_ = s.tasks.IncrementProcessed(ctx)
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		_ = s.tasks.UpdateProgress(ctx, task.ID, id, "ready")
		_ = s.tasks.IncrementProcessed(ctx)
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
	_ = s.tasks.UpdateTask(ctx, task)

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
	)
	return firstErr
}
