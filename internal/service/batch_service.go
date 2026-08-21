package service

import (
	"context"
	"fmt"
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

	articles, fetchErr := s.articles.GetArticles(ctx, task.ArticleIDs)
	if fetchErr != nil {
		task.Status = model.TaskFailed
		task.Error = fetchErr.Error()
		task.UpdatedAt = time.Now()
		_ = s.tasks.UpdateTask(ctx, task)
		metrics.Default().IncTasksFailed()
		return fetchErr
	}

	for i, article := range articles {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
		}

		articleID := task.ArticleIDs[i]

		result, err := s.analyzer.Analyze(ctx, articleID, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(ctx, article)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		successCount++
	}

	task.UpdatedAt = time.Now()
	if firstErr != nil {
		task.Status = model.TaskFailed
		task.Error = fmt.Sprintf("batch partially failed: %v (processed %d/%d)", firstErr, successCount, len(articles))
		metrics.Default().IncTasksFailed()
	} else {
		task.Status = model.TaskSuccess
		metrics.Default().IncTasksCompleted()
	}
	_ = s.tasks.UpdateTask(ctx, task)

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(articles),
		"status", string(task.Status),
	)
	return firstErr
}
