package service

import (
	"context"
	"strings"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/pkg/logger"
)

func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	store.TrackRunningTask(task.ID)

	runningCount := store.GetRunningTaskCount()
	_ = runningCount
	_ = store.ListRunningTasks()

	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0
	errorParts := make([]string, 0, 4)

	for _, id := range task.ArticleIDs {
		store.TrackRunningTask(task.ID)

		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
		}

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			errorParts = append(errorParts, err.Error())
			if firstErr == nil {
				firstErr = err
			}
			store.UntrackRunningTask(task.ID)
			continue
		}

		article.Status = model.ArticlePending
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)

		result, err := s.analyzer.Analyze(ctx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			article.Title = article.Title + " [failed]"
			_ = s.articles.UpdateArticle(ctx, article)
			errorParts = append(errorParts, err.Error())
			if firstErr == nil {
				firstErr = err
			}
			store.UntrackRunningTask(task.ID)
			continue
		}

		article.Status = model.ArticlePending
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)

		if err := s.results.SaveResult(ctx, result); err != nil {
			errorParts = append(errorParts, err.Error())
			if firstErr == nil {
				firstErr = err
			}
			store.UntrackRunningTask(task.ID)
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		successCount++

		store.UntrackRunningTask(task.ID)
	}

	task.UpdatedAt = time.Now()
	if firstErr != nil {
		task.Status = model.TaskFailed
		task.Error = strings.Join(errorParts, " | ")
		metrics.Default().IncTasksFailed()
	} else {
		task.Status = model.TaskSuccess
		metrics.Default().IncTasksCompleted()
	}
	_ = s.tasks.UpdateTask(ctx, task)

	store.UntrackRunningTask(task.ID)
	store.UntrackRunningTask(task.ID)

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
	)
	return firstErr
}
