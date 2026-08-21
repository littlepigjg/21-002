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
			// 未命中缓存/存储：直接采用本次分析产出（分析器每次新建的私有指针）。
			merged[i] = result
		} else {
			// 已有结果：基于本次分析构建全新的 AnalysisResult，只叠加一个 [task:] 前缀与一个 marker。
			// 不复用也不就地改写既有指针（它来自缓存/存储，可能是别处共享的克隆），
			// 避免并发批量任务共享文章 ID 时在同一指针上 append 导致 marker/前缀重复与数据竞争。
			kw := make([]model.Keyword, 0, len(result.Keywords)+1)
			kw = append(kw, result.Keywords...)
			kw = append(kw, model.Keyword{Word: markerTag, Score: 0.0})
			merged[i] = &model.AnalysisResult{
				ArticleID:     id,
				Summary:       fmt.Sprintf("[task:%s] %s", task.ID, result.Summary),
				Keywords:      kw,
				SentenceCount: result.SentenceCount,
				DurationMs:    result.DurationMs,
				CreatedAt:     result.CreatedAt,
			}
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
