package service

import (
	"context"
	"sync"
	"testing"

	"summarizer/internal/config"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func setupServices() (*ArticleService, *TaskService, *store.MemoryStore) {
	cfg := config.Default()
	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()

	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(cfg.MaxKeywordCount)
	textrank := NewTextRankService(cfg.TextRankMaxIter, cfg.TextRankDamping)
	summarizer := NewSummarizeService(cfg.MaxSummarySentences)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	articleSvc := NewArticleService(memStore, memStore, analyzer, ids, cfg.MaxArticleLength)
	queue := taskqueue.NewQueue(cfg.QueueCapacity)
	taskSvc := NewTaskService(memStore, memStore, memStore, analyzer, ids, queue, cfg.MaxArticleLength)

	return articleSvc, taskSvc, memStore
}

func TestBugNil008_ResultStorePanic(t *testing.T) {
	articleSvc, taskSvc, _ := setupServices()

	panicked := make(chan struct{})
	var once sync.Once

	ctx := context.Background()

	for i := 0; i < 20; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					once.Do(func() { close(panicked) })
				}
			}()
			req := model.SubmitArticleRequest{
				Title:   "test",
				Content: "This is a test article with enough content for analysis.",
			}
			_, _ = articleSvc.Submit(ctx, req)
		}()
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					once.Do(func() { close(panicked) })
				}
			}()
			batchReq := model.BatchSubmitRequest{
				Articles: []model.SubmitArticleRequest{
					{Title: "batch", Content: "Batch test article content for processing."},
				},
			}
			task, err := taskSvc.SubmitBatch(ctx, batchReq)
			if err != nil {
				return
			}
			_ = taskSvc.HandleJob(ctx, taskqueue.Job{
				ID:  task.ID,
				Run: func(jctx context.Context) error { return nil },
			})
		}()
	}

	wg.Wait()

	select {
	case <-panicked:
		t.Log("RED（红灯，缺陷未修复）")
		t.Fail()
	default:
		t.Log("GREEN（绿灯，缺陷已修复）")
	}
}
