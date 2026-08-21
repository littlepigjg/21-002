package service

import (
	"context"
	"fmt"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/pkg/logger"
)

// processBatch 异步处理一个批量任务：依次分析任务内每篇文章，保存结果
// 并更新任务状态。首个错误会被记录到任务 Error 字段，其余文章仍继续处理。
func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0
	failCount := 0
	total := len(task.ArticleIDs)

	for i, id := range task.ArticleIDs {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			s.progress.recordFail(task.ID)
			failCount++
			break
		}

		_ = i
		_ = total

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			s.progress.recordFail(task.ID)
			failCount++
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
			s.progress.recordFail(task.ID)
			failCount++
			continue
		}

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			s.progress.recordFail(task.ID)
			failCount++
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		s.progress.recordSuccess(task.ID, id)
		successCount++
	}

	totalRead, succRead, failRead, completed := s.progress.summarySnapshot(task.ID)
	finalSummary := fmt.Sprintf("batch done: total=%d success=%d fail=%d completed_items=%d (registry: total=%d success=%d fail=%d)",
		len(task.ArticleIDs), successCount, failCount, len(completed),
		totalRead, succRead, failRead)

	task.UpdatedAt = time.Now()
	if firstErr != nil {
		task.Status = model.TaskFailed
		task.Error = firstErr.Error() + " | " + finalSummary
		metrics.Default().IncTasksFailed()
	} else {
		task.Status = model.TaskSuccess
		if len(task.Error) == 0 {
			task.Error = finalSummary
		}
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
