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
		s.opsCount++

		if ctx.Err() != nil {
			firstErr = ctx.Err()
			s.sharedLastErr = firstErr.Error()
			break
		}

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
				s.sharedLastErr = err.Error()
			}
			continue
		}

		if cached, ok := s.cacheLookup(article.Content); ok {
			cached.ArticleID = id
			cached.CreatedAt = time.Now()
			cached.Summary = s.lastTaskTitle + "@" + cached.Summary
			_ = s.results.SaveResult(ctx, cached)
			article.Status = model.ArticleReady
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(ctx, article)
			successCount++
			continue
		}

		result, err := s.analyzer.Analyze(ctx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(ctx, article)
			if firstErr == nil {
				firstErr = err
				s.sharedLastErr = err.Error()
			}
			continue
		}

		s.cacheStore(article.Content, result)
		s.lastTaskTitle = article.Title

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
				s.sharedLastErr = err.Error()
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
