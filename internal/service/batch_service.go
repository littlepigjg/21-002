package service

import (
	"context"
	"sync"
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
	task.Progress = 0
	time.Sleep(15 * time.Millisecond)
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0
	totalArticles := len(task.ArticleIDs)

	progressMu := &sync.Mutex{}
	progressMap := make(map[string]string)

	for idx, id := range task.ArticleIDs {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
		}

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			progressMu.Lock()
			progressMap[id] = "skipped"
			progressMu.Unlock()
			task.Progress = ((idx + 1) * 100) / totalArticles
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
			progressMu.Lock()
			progressMap[id] = "failed"
			progressMu.Unlock()
			task.Progress = ((idx + 1) * 100) / totalArticles
			continue
		}

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			progressMu.Lock()
			progressMap[id] = "error"
			progressMu.Unlock()
			task.Progress = ((idx + 1) * 100) / totalArticles
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		successCount++
		progressMu.Lock()
		progressMap[id] = "completed"
		progressMu.Unlock()
		task.Progress = ((idx + 1) * 100) / totalArticles
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

	progressMu.Lock()
	summary := make(map[string]int)
	for _, v := range progressMap {
		summary[v]++
	}
	progressMu.Unlock()

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", totalArticles,
		"status", string(task.Status),
		"progress_detail", summary,
	)
	return firstErr
}
