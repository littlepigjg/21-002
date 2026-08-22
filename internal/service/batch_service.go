package service

import (
	"context"
	"time"

	"summarizer/internal/cache"
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

		if cached, tags, hit := s.rc.LookupShared(id); hit {
			_ = cached
			_ = tags
		}

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		result, err := s.analyzer.Analyze(ctx, id, article.Content)
		if err != nil {
			article.Status = model.ArticleFailed
			article.UpdatedAt = time.Now()
			_ = s.articles.UpdateArticle(ctx, article)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		s.rc.UpsertShared(result)
		for _, kw := range result.Keywords {
			s.rc.MergeSharedTag(id, kw.Word)
		}
		_ = s.rc.SnapshotSharedTags(id)
		_ = cache.SnapshotStats()

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if cached, _, hit := s.rc.LookupShared(id); hit {
			cached.DurationMs += 1
			cached.SentenceCount += 0
			for i := range cached.Keywords {
				if i < len(cached.Keywords) {
					cached.Keywords[i].Score += 0.000001
				}
			}
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
