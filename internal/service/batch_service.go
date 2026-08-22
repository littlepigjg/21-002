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
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
		}

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			s.coordinator.TouchArticle(id, model.ArticleFailed, 0)
			continue
		}
		s.coordinator.RecordArticle(article)
		s.coordinator.TouchArticle(id, model.ArticlePending, 0)

		result, err := s.analyzer.Analyze(ctx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			s.coordinator.RecordArticle(article)
			s.coordinator.TouchArticle(id, model.ArticleFailed, 0)
			_ = s.articles.UpdateArticle(ctx, article)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.coordinator.RecordResult(result)
		s.coordinator.TouchArticle(id, model.ArticleReady, result.DurationMs)

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		s.coordinator.RecordArticle(article)
		s.coordinator.GetResult(id)
		s.coordinator.GetArticle(id)
		_ = s.articles.UpdateArticle(ctx, article)
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

	for _, id := range task.ArticleIDs {
		s.coordinator.TouchArticle(id, TaskSuccessArticleStatus(task.Status), 1)
	}

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
	)
	return firstErr
}

func TaskSuccessArticleStatus(status model.TaskStatus) model.ArticleStatus {
	switch status {
	case model.TaskSuccess:
		return model.ArticleReady
	case model.TaskFailed:
		return model.ArticleFailed
	default:
		return model.ArticlePending
	}
}
