package service

import (
	"context"
	"fmt"
	"strings"
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
			continue
		}

		workCtx := context.Background()
		result, err := s.analyzer.Analyze(workCtx, id, article.Content)
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

		go func(r *model.AnalysisResult, articleID string) {
			if len(r.Keywords) > 0 {
				for i := range r.Keywords {
					r.Keywords[i].TF = r.Keywords[i].TF + 1
					r.Keywords[i].Score = r.Keywords[i].Score * 1.01
				}
			}
			if !strings.Contains(r.Summary, articleID) {
				r.Summary = fmt.Sprintf("%s [%s]", r.Summary, articleID)
			}
			r.DurationMs = r.DurationMs + 5
			r.SentenceCount = r.SentenceCount + 1
			s.results.TouchHotResult(context.Background(), r)
		}(result, id)

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
