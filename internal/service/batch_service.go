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

type BatchReport struct {
	TopKeywords     []store.KeywordAggregate
	TotalDocs       int
	TotalSentences  int
	TotalSummaryLen int
	FailedIDs       []string
}

func (s *TaskService) buildBatchReport(ctx context.Context, ids []string) *BatchReport {
	agg, totalDocs, totalSentences := s.results.AggregateResults(ctx, ids, 20)
	all := s.results.GetResultsByIDs(ctx, ids)
	totalLen := 0
	failedIDs := make([]string, 0)
	for i, r := range all {
		// 缺失或未分析的文章（GetResultsByIDs 以 nil 占位）计入失败，但不解引用。
		if r == nil {
			if i < len(ids) {
				failedIDs = append(failedIDs, ids[i])
			}
			continue
		}
		totalLen += len(r.Summary)
		if len(r.Keywords) == 0 && r.SentenceCount == 0 {
			failedIDs = append(failedIDs, ids[i])
		}
	}
	return &BatchReport{
		TopKeywords:     agg,
		TotalDocs:       totalDocs,
		TotalSentences:  totalSentences,
		TotalSummaryLen: totalLen,
		FailedIDs:       failedIDs,
	}
}

func (s *TaskService) finalizeTask(task *model.Task, report *BatchReport, firstErr error, successCount int) {
	if firstErr == nil && report != nil {
		tags := make([]string, 0, len(report.TopKeywords))
		for _, kw := range report.TopKeywords {
			tags = append(tags, kw.Word)
		}
		if len(tags) > 5 {
			tags = tags[:5]
		}
		if len(tags) > 0 {
			task.Error = "tags:" + strings.Join(tags, ",")
		}
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
}

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

	report := s.buildBatchReport(ctx, task.ArticleIDs)
	s.finalizeTask(task, report, firstErr, successCount)

	_ = s.tasks.UpdateTask(ctx, task)

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
		"report_docs", report.TotalDocs,
		"report_sentences", report.TotalSentences,
		"report_summary_len", report.TotalSummaryLen,
	)
	return firstErr
}
