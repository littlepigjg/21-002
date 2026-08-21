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

func buildTaskServiceForRace(workers, queueCap, maxLen int) (*TaskService, *taskqueue.Manager, context.CancelFunc) {
	ids := store.NewIDGenerator()
	mem := store.NewMemoryStore()

	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(8)
	textrank := NewTextRankService(20, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	queue := taskqueue.NewQueue(queueCap)
	taskSvc := NewTaskService(mem, mem, mem, analyzer, ids, queue, maxLen)

	ctx, cancel := context.WithCancel(context.Background())
	manager := taskqueue.NewManager(queue, workers, taskSvc.HandleJob)
	manager.Start(ctx)

	return taskSvc, manager, cancel
}

func makeBatchRequest(articles int) model.BatchSubmitRequest {
	req := model.BatchSubmitRequest{
		Articles: make([]model.SubmitArticleRequest, 0, articles),
	}
	for i := 0; i < articles; i++ {
		body := "这是第 " + fmt.Sprintf("%d", i) + " 段测试正文内容。" +
			"今天的天气非常好，阳光明媚，微风不燥。" +
			"我们正在进行一项并发编程的可靠性测试，用来检验系统在高并发下的行为稳定性。" +
			"每一段文字都需要被正确分句、分词，完成关键词抽取和摘要生成。" +
			"希望所有的 goroutine 都能安全地协作，而不会出现任何意料之外的崩溃。"
		req.Articles = append(req.Articles, model.SubmitArticleRequest{
			Title:   fmt.Sprintf("测试文章标题-%d", i),
			Content: body,
		})
	}
	return req
}

func TestBugConcur024_ConcurrentTaskProgress(t *testing.T) {
	const submitGoroutines = 16
	const articlesPerTask = 40
	const readerGoroutines = 20
	const iterationsPerReader = 120
	totalExpectedArticles := submitGoroutines * articlesPerTask

	svc, manager, cancel := buildTaskServiceForRace(4, 256, 10000)
	defer cancel()
	defer func() {
		cancel()
		manager.Stop()
	}()

	var panicCount int64
	var submittedOK int64
	var taskIDs sync.Map

	submit := func(wg *sync.WaitGroup) {
		defer wg.Done()
		ctx := context.Background()
		req := makeBatchRequest(articlesPerTask)
		task, err := svc.SubmitBatch(ctx, req)
		if err != nil {
			return
		}
		atomic.AddInt64(&submittedOK, 1)
		taskIDs.Store(task.ID, struct{}{})
	}

	reader := func(wg *sync.WaitGroup) {
		defer wg.Done()
		ctx := context.Background()
		defer func() {
			if r := recover(); r != nil {
				atomic.AddInt64(&panicCount, 1)
				fmt.Printf("[panic] %v\n", r)
			}
		}()
		for i := 0; i < iterationsPerReader; i++ {
			taskIDs.Range(func(key, _ interface{}) bool {
				id, _ := key.(string)
				progress, tsk, perr := svc.GetTaskProgress(ctx, id)
				if perr == nil {
					_ = progress
					if tsk != nil {
						_ = tsk.Status
						_ = tsk.UpdatedAt
						_ = tsk.Error
						_ = len(tsk.ArticleIDs)
					}
				}
				_ = svc.TouchTask(ctx, id)
				_, _ = svc.GetProcessedCount(ctx)
				time.Sleep(time.Microsecond * 30)
				return true
			})
			time.Sleep(time.Microsecond * 10)
		}
	}

	var wgSubmit sync.WaitGroup
	for i := 0; i < submitGoroutines; i++ {
		wgSubmit.Add(1)
		go submit(&wgSubmit)
	}

	time.Sleep(time.Millisecond * 10)

	var wgReaders sync.WaitGroup
	for i := 0; i < readerGoroutines; i++ {
		wgReaders.Add(1)
		go reader(&wgReaders)
	}

	wgSubmit.Wait()
	wgReaders.Wait()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		allDone := true
		taskIDs.Range(func(key, _ interface{}) bool {
			id, _ := key.(string)
			tsk, err := svc.GetTask(context.Background(), id)
			if err != nil {
				return true
			}
			if tsk.Status == model.TaskPending || tsk.Status == model.TaskRunning {
				allDone = false
				return false
			}
			return true
		})
		if allDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	panics := atomic.LoadInt64(&panicCount)
	if panics > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("concurrent access triggered %d panic(s), e.g. concurrent map read and map write", panics)
	}

	actualProcessed, cerr := svc.GetProcessedCount(context.Background())
	if cerr != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("failed to read processed count: %v", cerr)
	}

	if actualProcessed != int64(totalExpectedArticles) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("processed count mismatch due to lost non-atomic increments: expected %d, got %d",
			totalExpectedArticles, actualProcessed)
	}

	totalProgressEntries := 0
	totalNonReady := 0
	taskStatusErrors := 0
	taskIDs.Range(func(key, _ interface{}) bool {
		id, _ := key.(string)
		prog, _, perr := svc.GetTaskProgress(context.Background(), id)
		if perr == nil {
			totalProgressEntries += len(prog)
			for _, v := range prog {
				if v != "ready" {
					totalNonReady++
				}
			}
		}
		tsk, gerr := svc.GetTask(context.Background(), id)
		if gerr == nil {
			if tsk.Status != model.TaskSuccess {
				taskStatusErrors++
			}
		}
		return true
	})

	submitted := atomic.LoadInt64(&submittedOK)
	expectedProgress := int(submitted) * articlesPerTask
	if totalProgressEntries != expectedProgress {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("progress entries lost due to concurrent map writes: expected %d across %d tasks, got %d (%d non-ready)",
			expectedProgress, submitted, totalProgressEntries, totalNonReady)
	}

	if taskStatusErrors != 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("%d tasks ended with wrong status (shared task pointer fields overwritten by TouchTask race)",
			taskStatusErrors)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
