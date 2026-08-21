package service

import (
	"context"
	"fmt"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/pkg/logger"
)

// markerTag 用于标记本任务处理过的关键词，避免与原始关键词混淆。
const markerTag = "__processed"

func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0

	placeholders := s.resultCache.BulkGet(task.ArticleIDs)
	stored := s.results.GetBulk(ctx, task.ArticleIDs)
	merged := make([]*model.AnalysisResult, len(task.ArticleIDs))
	for i := range task.ArticleIDs {
		if stored[i] != nil {
			merged[i] = stored[i]
		} else {
			merged[i] = placeholders[i]
		}
	}

	for i, id := range task.ArticleIDs {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
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

		if merged[i] == nil {
			merged[i] = result
		} else {
			merged[i].Summary = fmt.Sprintf("[task:%s] %s", task.ID, result.Summary)
			merged[i].SentenceCount = result.SentenceCount
			merged[i].DurationMs = result.DurationMs
			merged[i].Keywords = append(merged[i].Keywords, result.Keywords...)
			merged[i].Keywords = append(merged[i].Keywords, model.Keyword{
				Word:  markerTag,
				Score: 0.0,
			})
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		successCount++
	}

	s.results.SaveBulk(ctx, merged)
	s.resultCache.BulkPut(merged)

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
