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

func TestBugNilArticle010_DeletedBatchArticlePanic(t *testing.T) {
	ms := store.NewMemoryStore()
	sw := textutil.NewStopwordSet()
	prep := NewPreprocessor(sw)
	tfidf := NewTfidfService(10)
	tr := NewTextRankService(30, 0.85)
	summ := NewSummarizeService(5)
	analyzer := NewAnalyzer(prep, tfidf, tr, summ)
	ids := store.NewIDGenerator()
	q := taskqueue.NewQueue(16)
	taskSvc := NewTaskService(ms, ms, ms, analyzer, ids, q, 100000)
	ctx := context.Background()

	contentA := "Go语言是一门开源编程语言，由Google开发。Go语言具有简洁、高效、安全等特点，广泛应用于云计算和微服务领域。Go语言的并发编程模型基于goroutine和channel，使得编写并发程序变得简单。"
	contentB := "Rust语言是一门系统编程语言，由Mozilla开发。Rust语言以内存安全和零成本抽象著称，广泛应用于系统编程和WebAssembly领域。Rust语言的所有权系统确保了内存安全。"

	req := model.BatchSubmitRequest{
		Articles: []model.SubmitArticleRequest{
			{Title: "Go语言", Content: contentA},
			{Title: "Rust语言", Content: contentB},
		},
	}

	task, err := taskSvc.SubmitBatch(ctx, req)
	if err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}

	for _, aid := range task.ArticleIDs {
		_ = ms.DeleteArticle(ctx, aid)
	}

	panicDetected := make(chan struct{})
	normalDone := make(chan struct{})

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RED（红灯，缺陷未修复）")
				fmt.Printf("panic recovered: %v\n", r)
				close(panicDetected)
			}
		}()
		j := <-q.Jobs()
		_ = taskSvc.HandleJob(ctx, j)
		close(normalDone)
	}()

	select {
	case <-panicDetected:
		return
	case <-normalDone:
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	case <-time.After(10 * time.Second):
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		t.Fatal("timeout waiting for batch processing")
	}
}
