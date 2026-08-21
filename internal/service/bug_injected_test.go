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

func buildBugInjectionDeps(workers int) (*store.MemoryStore, *store.IDGenerator, *Analyzer, *taskqueue.Queue, *taskqueue.Manager, *TaskService, *ArticleService) {
	mem := store.NewMemoryStore()
	ids := store.NewIDGenerator()

	sw := textutil.NewStopwordSet()
	sw.Add("the")
	sw.Add("a")
	sw.Add("an")
	sw.Add("is")
	sw.Add("are")
	sw.Add("of")
	sw.Add("to")
	sw.Add("in")
	sw.Add("for")
	sw.Add("and")
	sw.Add("or")
	sw.Add("but")

	pre := NewPreprocessor(sw)
	tfidf := NewTfidfService(5)
	tr := NewTextRankService(20, 0.85)
	summ := NewSummarizeService(3)
	analyzer := NewAnalyzer(pre, tfidf, tr, summ)

	q := taskqueue.NewQueue(1024)
	ts := NewTaskService(mem, mem, mem, analyzer, ids, q, 100000)
	mgr := taskqueue.NewManager(q, workers, ts.HandleJob)
	art := NewArticleService(mem, mem, analyzer, ids, 100000)
	return mem, ids, analyzer, q, mgr, ts, art
}

func sampleArticles(n int) []model.SubmitArticleRequest {
	out := make([]model.SubmitArticleRequest, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, model.SubmitArticleRequest{
			Title:   fmt.Sprintf("title-%d", i),
			Content: fmt.Sprintf("The quick brown fox jumps over the lazy dog. This is sentence number %d about testing concurrency behavior. Another paragraph that contains more words for analysis. We want to ensure enough tokens are produced.", i),
		})
	}
	return out
}

func TestBugConcur003_RunningTaskMapRace(t *testing.T) {
	const (
		batchSubmitters = 4
		batchesEach     = 5
		articlesPerBatch = 3
		singleSubmitters = 4
		singlesEach      = 10
		observers        = 3
		duration         = 2 * time.Second
	)

	_, ids, _, q, mgr, ts, art := buildBugInjectionDeps(4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	_ = ids

	var wg sync.WaitGroup
	panicCh := make(chan any, 256)
	errCh := make(chan string, 256)

	safeGo := func(name string, fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case panicCh <- fmt.Sprintf("%s panic: %v", name, r):
					default:
					}
				}
			}()
			fn()
		}()
	}

	for s := 0; s < batchSubmitters; s++ {
		sid := s
		safeGo(fmt.Sprintf("batch-sub-%d", sid), func() {
			for i := 0; i < batchesEach; i++ {
				req := model.BatchSubmitRequest{Articles: sampleArticles(articlesPerBatch)}
				task, err := ts.SubmitBatch(ctx, req)
				if err != nil {
					select {
					case errCh <- fmt.Sprintf("submit-batch err: %v", err):
					default:
					}
					continue
				}
				deadline := time.Now().Add(2 * time.Second)
				for time.Now().Before(deadline) {
					cur, err := ts.GetTask(ctx, task.ID)
					if err != nil {
						break
					}
					if cur.Status.IsTerminal() {
						_ = store.GetRunningTaskCount()
						_ = store.ListRunningTasks()
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
		})
	}

	for s := 0; s < singleSubmitters; s++ {
		sid := s
		safeGo(fmt.Sprintf("single-sub-%d", sid), func() {
			for i := 0; i < singlesEach; i++ {
				_ = store.GetRunningTaskCount()
				_ = store.SnapshotWorkerStats()
				req := model.SubmitArticleRequest{
					Title:   fmt.Sprintf("single-%d-%d", sid, i),
					Content: fmt.Sprintf("Single article body %d. We want enough content to produce meaningful sentences. Foxes, dogs, and jumps are all part of the testing story.", i),
				}
				_, err := art.Submit(ctx, req)
				if err != nil {
					select {
					case errCh <- fmt.Sprintf("submit-single err: %v", err):
					default:
					}
				}
			}
		})
	}

	for o := 0; o < observers; o++ {
		oid := o
		safeGo(fmt.Sprintf("observer-%d", oid), func() {
			deadline := time.Now().Add(duration)
			for time.Now().Before(deadline) {
				_ = store.GetRunningTaskCount()
				_ = store.ListRunningTasks()
				_ = store.SnapshotWorkerStats()
				for w := 0; w < 8; w++ {
					_ = store.GetWorkerStat(w)
				}
				store.TrackRunningTask(fmt.Sprintf("observer-%d-tmp", oid))
				store.UntrackRunningTask(fmt.Sprintf("observer-%d-tmp", oid))
				store.IncWorkerStat(oid)
				time.Sleep(1 * time.Millisecond)
			}
		})
	}

	wg.Wait()
	_ = q

	hasPanic := len(panicCh) > 0
	hasRaceIndicator := false

	close(panicCh)
	seenPanics := make([]any, 0, len(panicCh))
	for p := range panicCh {
		seenPanics = append(seenPanics, p)
	}

	close(errCh)
	errs := make([]string, 0, len(errCh))
	for e := range errCh {
		errs = append(errs, e)
	}

	t.Logf("panics=%d race_indicator=%v errors=%d", len(seenPanics), hasRaceIndicator, len(errs))
	for i, p := range seenPanics {
		t.Logf("  panic[%d]: %v", i, p)
	}
	for i, e := range errs {
		if i < 5 {
			t.Logf("  err[%d]: %s", i, e)
		}
	}

	if hasPanic || hasRaceIndicator {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED: detected panics=%d race_like=%v", len(seenPanics), hasRaceIndicator)
	}

	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer verifyCancel()

	verifyMem, verifyIds, _, verifyQ, verifyMgr, verifyTs, _ := buildBugInjectionDeps(6)
	verifyMgr.Start(verifyCtx)
	defer verifyMgr.Stop()

	const verifyBatches = 20
	verifyArt := sampleArticles(3)
	var vwg sync.WaitGroup
	var vmu sync.Mutex
	badStates := 0
	totalTerminal := 0

	for i := 0; i < verifyBatches; i++ {
		vwg.Add(1)
		go func(idx int) {
			defer vwg.Done()
			defer func() {
				if r := recover(); r != nil {
					vmu.Lock()
					badStates++
					vmu.Unlock()
				}
			}()
			req := model.BatchSubmitRequest{Articles: verifyArt}
			task, err := verifyTs.SubmitBatch(verifyCtx, req)
			if err != nil {
				vmu.Lock()
				badStates++
				vmu.Unlock()
				return
			}
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				cur, err := verifyTs.GetTask(verifyCtx, task.ID)
				if err != nil {
					return
				}
				if cur.Status.IsTerminal() {
					vmu.Lock()
					totalTerminal++
					vmu.Unlock()
					return
				}
				time.Sleep(3 * time.Millisecond)
			}
		}(i)
	}

	for o := 0; o < 4; o++ {
		vwg.Add(1)
		go func() {
			defer vwg.Done()
			defer func() { _ = recover() }()
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				_ = store.GetRunningTaskCount()
				_ = store.ListRunningTasks()
				_ = store.SnapshotWorkerStats()
				store.TrackRunningTask(fmt.Sprintf("v-%s", verifyIds.Next("x")))
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	vwg.Wait()
	_ = verifyMem
	_ = verifyQ

	if badStates > 0 || totalTerminal != verifyBatches {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED: badStates=%d terminalTasks=%d expected=%d", badStates, totalTerminal, verifyBatches)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
