package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func buildTestServices() (*ArticleService, *TaskService, *Analyzer, *taskqueue.Queue) {
	ids := store.NewIDGenerator()
	mem := store.NewMemoryStore()
	stopwords := textutil.NewStopwordSet()
	prep := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(10)
	tr := NewTextRankService(30, 0.85)
	sum := NewSummarizeService(5)
	analyzer := NewAnalyzer(prep, tfidf, tr, sum)
	q := taskqueue.NewQueue(8192)
	as := NewArticleService(mem, mem, analyzer, ids, 1_000_000)
	ts := NewTaskService(mem, mem, mem, analyzer, ids, q, 1_000_000)
	return as, ts, analyzer, q
}

func TestBugCtxReuse022_ConcurrentSubmitWithCancelledContext(t *testing.T) {
	as, ts, analyzerObj, q := buildTestServices()
	rootCtx := context.Background()

	manager := taskqueue.NewManager(q, 16, ts.HandleJob)
	manager.Start(rootCtx)
	defer manager.Stop()

	const cancelledArticleGW = 80
	const validArticleGW = 80
	const validBatchGW = 40
	const repeatsPerGoroutine = 80

	var panicObserved atomic.Bool
	var spuriousErrors atomic.Int64
	var cancelledExpected atomic.Int64
	var successOps atomic.Int64
	var totalOps atomic.Int64

	totalWorkers := cancelledArticleGW + validArticleGW + validBatchGW
	var ready sync.WaitGroup
	ready.Add(totalWorkers)
	start := make(chan struct{})
	var done sync.WaitGroup
	done.Add(totalWorkers)

	sampleContents := []string{
		"The quick brown fox jumps over the lazy dog near the riverbank on a sunny morning. The fox was quick and clever. It avoided every trap set by the farmer. The dog just watched lazily from the porch.",
		"Machine learning is a subset of artificial intelligence. It enables systems to learn from data without explicit programming. Supervised learning uses labeled data. Unsupervised learning finds patterns in unlabeled data. Both approaches are widely used today.",
		"Go is a statically typed compiled language designed at Google. It emphasizes simplicity and concurrency. Goroutines are lightweight threads. Channels enable safe communication between goroutines. The standard library is comprehensive.",
		"Climate change affects ecosystems worldwide. Rising temperatures cause ice caps to melt. Sea levels increase every year. Extreme weather events become more common. Scientists urge immediate action to reduce greenhouse gas emissions.",
		"Reading books expands vocabulary and improves focus. Fiction encourages empathy. Non-fiction teaches practical skills. Libraries provide free access to knowledge. Every page offers a new perspective on the world.",
		"The solar system contains eight planets. Mercury is the closest to the sun. Venus is the hottest planet. Earth supports life. Mars has the largest volcano. Jupiter is the biggest planet. Saturn has beautiful rings.",
		"Nutrition is fundamental to good health. Proteins build muscle tissue. Carbohydrates provide energy. Fats support hormone production. Vitamins and minerals regulate cellular functions. A balanced diet includes all these components.",
		"Architecture combines art and engineering to shape environments. Classical styles emphasize symmetry. Modern designs explore new materials. Sustainable buildings reduce environmental impact. Every structure reflects its cultural context.",
	}

	cancelledArticleWorker := func(id int) {
		defer func() {
			if r := recover(); r != nil {
				panicObserved.Store(true)
				fmt.Printf("[cancelled-art-%d] panic: %v\n", id, r)
			}
		}()
		ready.Done()
		<-start
		defer done.Done()
		for i := 0; i < repeatsPerGoroutine; i++ {
			totalOps.Add(1)
			ctx, cancel := context.WithCancel(rootCtx)
			cancel()
			content := sampleContents[(id*3+i)%len(sampleContents)]
			req := model.SubmitArticleRequest{
				Title:   fmt.Sprintf("cancel-art-%d-%d", id, i),
				Content: content,
			}
			_, err := as.Submit(ctx, req)
			if err != nil {
				cancelledExpected.Add(1)
			} else {
				successOps.Add(1)
			}
		}
	}

	validArticleWorker := func(id int) {
		defer func() {
			if r := recover(); r != nil {
				panicObserved.Store(true)
				fmt.Printf("[valid-art-%d] panic: %v\n", id, r)
			}
		}()
		ready.Done()
		<-start
		defer done.Done()
		for i := 0; i < repeatsPerGoroutine; i++ {
			totalOps.Add(1)
			func() {
				ctx, cancel := context.WithCancel(rootCtx)
				defer cancel()
				content := sampleContents[(id*5+i)%len(sampleContents)]
				req := model.SubmitArticleRequest{
					Title:   fmt.Sprintf("valid-art-%d-%d", id, i),
					Content: content,
				}
				resp, err := as.Submit(ctx, req)
				if err != nil {
					spuriousErrors.Add(1)
					return
				}
				if resp != nil && resp.ArticleID != "" {
					successOps.Add(1)
				}
			}()
		}
	}

	validBatchWorker := func(id int) {
		defer func() {
			if r := recover(); r != nil {
				panicObserved.Store(true)
				fmt.Printf("[valid-batch-%d] panic: %v\n", id, r)
			}
		}()
		ready.Done()
		<-start
		defer done.Done()
		for i := 0; i < repeatsPerGoroutine/2; i++ {
			totalOps.Add(1)
			func() {
				ctx, cancel := context.WithCancel(rootCtx)
				defer cancel()
				articleCount := 3 + (i % 4)
				articles := make([]model.SubmitArticleRequest, 0, articleCount)
				for k := 0; k < articleCount; k++ {
					content := sampleContents[(id*7+i*2+k)%len(sampleContents)]
					articles = append(articles, model.SubmitArticleRequest{
						Title:   fmt.Sprintf("vbatch-%d-%d-%d", id, i, k),
						Content: content,
					})
				}
				req := model.BatchSubmitRequest{Articles: articles}
				task, err := ts.SubmitBatch(ctx, req)
				if err != nil {
					spuriousErrors.Add(1)
					return
				}
				if task != nil && task.ID != "" {
					successOps.Add(1)
				}
			}()
		}
	}

	for g := 0; g < cancelledArticleGW; g++ {
		go cancelledArticleWorker(g)
	}
	for g := 0; g < validArticleGW; g++ {
		go validArticleWorker(g)
	}
	for g := 0; g < validBatchGW; g++ {
		go validBatchWorker(g)
	}
	ready.Wait()
	close(start)
	done.Wait()

	time.Sleep(3 * time.Second)

	red := false
	reason := ""
	if panicObserved.Load() {
		red = true
		reason = "panic observed during concurrent execution (concurrent map write or unsafe shared state mutation)"
	}
	if spuriousErrors.Load() > 0 {
		red = true
		reason = fmt.Sprintf("valid-ctx requests returned errors unexpectedly: %d times; cancelledExpected=%d, successOps=%d, totalOps=%d",
			spuriousErrors.Load(), cancelledExpected.Load(), successOps.Load(), totalOps.Load())
	}
	if successOps.Load() == 0 {
		red = true
		reason = "no successful submission completed"
	}

	var asCounter int64
	var tsCounter int64
	var anCounter int64
	func() {
		defer func() { _ = recover() }()
		asCounter = as.opsCount
		tsCounter = ts.opsCount
		anCounter = analyzerObj.runCounter
	}()

	fmt.Println()
	fmt.Printf("总请求数: %d\n", totalOps.Load())
	fmt.Printf("  取消请求期望错误数: %d\n", cancelledExpected.Load())
	fmt.Printf("  成功完成数: %d\n", successOps.Load())
	fmt.Printf("  伪错误数（有效请求却返回错误）: %d\n", spuriousErrors.Load())
	fmt.Printf("  as.opsCount=%d, ts.opsCount=%d, analyzer.runCounter=%d\n", asCounter, tsCounter, anCounter)
	fmt.Printf("  panicObserved=%v\n", panicObserved.Load())

	if red {
		fmt.Printf("异常原因: %s\n", reason)
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf(reason)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
