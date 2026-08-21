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

func buildServices(queueCapacity, workers int) (*TaskService, *store.MemoryStore, *taskqueue.Manager, *taskqueue.Queue, context.CancelFunc) {
	stopwords := textutil.NewStopwordSet()
	stopwords.Add("the")
	stopwords.Add("a")
	stopwords.Add("and")
	stopwords.Add("of")
	stopwords.Add("to")
	stopwords.Add("in")
	stopwords.Add("for")
	stopwords.Add("is")
	stopwords.Add("it")
	stopwords.Add("on")

	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(8)
	textrank := NewTextRankService(20, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	mem := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	queue := taskqueue.NewQueue(queueCapacity)

	svc := NewTaskService(mem, mem, mem, analyzer, ids, queue, 100000)

	manager := taskqueue.NewManager(queue, workers, func(ctx context.Context, j taskqueue.Job) error {
		return svc.HandleJob(ctx, j)
	})

	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	return svc, mem, manager, queue, cancel
}

func makeArticles(n int) []model.SubmitArticleRequest {
	out := make([]model.SubmitArticleRequest, 0, n)
	for i := 0; i < n; i++ {
		title := fmt.Sprintf("title-%d", i)
		content := fmt.Sprintf(
			"The quick brown fox jumps over the lazy dog. This is a sample sentence about nature and science. Technology changes the world every day, and we all need to adapt to new realities quickly. Paragraph number %d with unique content here.",
			i,
		)
		out = append(out, model.SubmitArticleRequest{Title: title, Content: content})
	}
	return out
}

func TestBugProgressConcurrentRace(t *testing.T) {
	workers := 4
	queueCapacity := 128
	svc, _, manager, _, cancel := buildServices(queueCapacity, workers)
	defer cancel()
	defer manager.Stop()

	ctx := context.Background()
	batches := 32
	articlesPerBatch := 12

	var wg sync.WaitGroup
	stopPoll := make(chan struct{})
	var pollWg sync.WaitGroup

	pollWg.Add(1)
	go func() {
		defer pollWg.Done()
		for {
			select {
			case <-stopPoll:
				return
			default:
			}
			_ = svc.AggregateProgressSnapshot()
			_ = SumAllProgress(svc)
			_, _, _ = CountRunningProgress(svc)
			_ = CollectAllCompletedIDs(svc)
		}
	}()

	submitStart := make(chan struct{})
	for i := 0; i < batches; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-submitStart
			req := model.BatchSubmitRequest{
				Articles: makeArticles(articlesPerBatch),
			}
			_, err := svc.SubmitBatch(ctx, req)
			if err != nil {
				t.Logf("submit batch %d err: %v", idx, err)
			}
		}(i)
	}
	close(submitStart)

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		if q := queueCapacity; q > 0 {
			all, _, _ := svc.ListTasks(ctx, 0, 1000)
			finished := 0
			for _, tk := range all {
				if tk.Status == model.TaskSuccess || tk.Status == model.TaskFailed {
					finished++
				}
			}
			if finished >= batches {
				break
			}
		}
	}
	wg.Wait()

	close(stopPoll)
	pollWg.Wait()

	totals := SumAllProgress(svc)
	snap := svc.AggregateProgressSnapshot()
	expectedTotalItems := batches * articlesPerBatch
	sumCompleted := 0
	for _, p := range snap {
		sumCompleted += len(p.Completed)
		if p.Success+p.Fail != len(p.Completed) {
			t.Errorf("mismatch: task %s success=%d fail=%d completed=%d",
				p.TaskID, p.Success, p.Fail, len(p.Completed))
		}
	}

	if totals.TotalItems != expectedTotalItems {
		t.Errorf("totals.TotalItems=%d expected %d (SumAllProgress corrupted due to race?)",
			totals.TotalItems, expectedTotalItems)
	}
	if sumCompleted != totals.SuccItems+totals.FailItems {
		t.Errorf("sumCompleted=%d vs succ+fail=%d (progress snapshot read/write mismatch due to race?)",
			sumCompleted, totals.SuccItems+totals.FailItems)
	}

	if t.Failed() {
		fmt.Println("RED（红灯，缺陷未修复）")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}
