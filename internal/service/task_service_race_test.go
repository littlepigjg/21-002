package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func TestBugConcur003_ServicePanic(t *testing.T) {
	panicCount := 0
	var mu sync.Mutex

	for round := 0; round < 10; round++ {
		roundPanicked := runServiceRaceTest()
		if roundPanicked {
			mu.Lock()
			panicCount++
			mu.Unlock()
		}
	}

	if panicCount > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）：%d/10 轮次出现 send on closed channel panic\n", panicCount)
		t.Fatalf("RED: %d rounds panicked", panicCount)
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）：10 轮次均无 panic")
}

func runServiceRaceTest() (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()

	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(5, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)
	queue := taskqueue.NewQueue(32)
	svc := NewTaskService(memStore, memStore, memStore, analyzer, ids, queue, 100000)

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				req := model.BatchSubmitRequest{
					Articles: []model.SubmitArticleRequest{
						{Title: "test", Content: "hello world content"},
					},
				}
				_, _ = svc.SubmitBatch(ctx, req)
			}
		}()
	}

	time.Sleep(100 * time.Microsecond)

	queue.Close()

	wg.Wait()
	return false
}
