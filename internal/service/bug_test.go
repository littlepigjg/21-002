package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func TestBugConcur002_BatchZeroWorkerHang(t *testing.T) {
	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	stopwords := textutil.NewStopwordSet()
	prep := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(30, 0.85)
	summ := NewSummarizeService(5)
	analyzer := NewAnalyzer(prep, tfidf, textrank, summ)

	q := taskqueue.NewQueue(256)
	taskSvc := NewTaskService(memStore, memStore, memStore, analyzer, ids, q, 100000)

	mgr := taskqueue.NewManager(q, 0, taskSvc.HandleJob)
	mgr.Start(context.Background())

	req := model.BatchSubmitRequest{
		Articles: []model.SubmitArticleRequest{
			{Title: "test", Content: "hello world test content here"},
		},
	}
	task, err := taskSvc.SubmitBatch(context.Background(), req)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("submit batch failed: %v", err)
	}

	done := make(chan bool, 1)
	go func() {
		deadline := time.After(2 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-deadline:
				tk, err := taskSvc.GetTask(context.Background(), task.ID)
				if err != nil {
					done <- false
					return
				}
				if tk.Status == model.TaskPending || tk.Status == model.TaskRunning {
					done <- false
					return
				}
				done <- true
				return
			case <-ticker.C:
				tk, err := taskSvc.GetTask(context.Background(), task.ID)
				if err != nil {
					continue
				}
				if tk.Status != model.TaskPending && tk.Status != model.TaskRunning {
					done <- true
					return
				}
			}
		}
	}()

	result := <-done
	if !result {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("task %s stuck in pending with zero workers", task.ID)
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
