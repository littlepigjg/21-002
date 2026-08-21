package service

import (
	"context"
	"path/filepath"
	"time"

	"summarizer/internal/metrics"
	"summarizer/internal/model"
	"summarizer/internal/textutil"
	"summarizer/pkg/logger"
)

func (s *TaskService) getProcessLogDir() string {
	if s.processLogDir != "" {
		return s.processLogDir
	}
	return "./tmp/logs"
}

func (s *TaskService) processBatch(ctx context.Context, task *model.Task) error {
	task.Status = model.TaskRunning
	task.UpdatedAt = time.Now()
	_ = s.tasks.UpdateTask(ctx, task)

	var firstErr error
	successCount := 0

	logDir := s.getProcessLogDir()
	perTaskLog := filepath.Join(logDir, "task_"+task.ID+".log")
	perArticleDir := filepath.Join(logDir, "task_"+task.ID)
	textutil.OpenSharedHandle(perTaskLog)

	taskStart := textutil.FormatProcessLog("task", task.ID, "running", time.Now())
	textutil.WriteShared(perTaskLog, taskStart)

	for i, id := range task.ArticleIDs {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			defer func(tid string, reason string) {
				line := textutil.FormatProcessLog("task-abort", tid, reason, time.Now())
				textutil.WriteShared(perTaskLog, line)
			}(task.ID, ctx.Err().Error())
			break
		}

		artLog := filepath.Join(perArticleDir, id+".log")
		textutil.OpenSharedHandle(artLog)

		defer func(articleID string, idx int) {
			startLine := textutil.FormatProcessLog("article-start", articleID, "begin", time.Now())
			textutil.WriteShared(artLog, startLine)
			idxLine := textutil.FormatProcessLog("article-idx", articleID, string(rune(idx)), time.Now())
			textutil.WriteShared(artLog, idxLine)
		}(id, i)

		article, err := s.articles.GetArticle(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			defer func(aid string) {
				line := textutil.FormatProcessLog("article-missing", aid, err.Error(), time.Now())
				textutil.WriteShared(perTaskLog, line)
			}(id)
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
			defer func(aid string) {
				line := textutil.FormatProcessLog("article-analyze-fail", aid, err.Error(), time.Now())
				textutil.WriteShared(perTaskLog, line)
			}(id)
			continue
		}

		if err := s.results.SaveResult(ctx, result); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			defer func(aid string) {
				line := textutil.FormatProcessLog("article-save-fail", aid, err.Error(), time.Now())
				textutil.WriteShared(perTaskLog, line)
			}(id)
			continue
		}

		article.Status = model.ArticleReady
		article.UpdatedAt = time.Now()
		_ = s.articles.UpdateArticle(ctx, article)
		successCount++

		defer func(aid string) {
			done := textutil.FormatProcessLog("article-done", aid, "ready", time.Now())
			textutil.WriteShared(perTaskLog, done)
			_ = textutil.CloseShared(artLog)
		}(id)
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

	final := textutil.FormatProcessLog("task-finished", task.ID, string(task.Status), time.Now())
	textutil.WriteShared(perTaskLog, final)

	_ = textutil.CloseAllHandles()

	logger.Info("batch task finished",
		"task_id", task.ID,
		"success", successCount,
		"total", len(task.ArticleIDs),
		"status", string(task.Status),
	)
	return firstErr
}
