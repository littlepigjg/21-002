package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func buildTestServices() (*ArticleService, *TaskService, *store.MemoryStore) {
	st := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	stopwords := textutil.NewStopwordSet()
	pre := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(20)
	textrank := NewTextRankService(20, 0.85)
	summ := NewSummarizeService(3)
	analyzer := NewAnalyzer(pre, tfidf, textrank, summ)
	queue := taskqueue.NewQueue(1024)
	articleSvc := NewArticleService(st, st, analyzer, ids, 1000000)
	taskSvc := NewTaskService(st, st, st, analyzer, ids, queue, 1000000)
	return articleSvc, taskSvc, st
}

func sampleContent(i int) string {
	return fmt.Sprintf(`这是第 %d 篇测试文章。人工智能正在改变世界。
机器学习是人工智能的核心技术。深度学习在计算机视觉领域取得了巨大成功。
自然语言处理让机器可以理解人类语言。数据驱动的方法在各个行业广泛应用。
研究者不断提出新的算法和模型。开源社区推动了技术的快速发展。
未来人工智能将更加深入地融入我们的生活。技术创新永无止境。我们需要关注技术伦理。`, i)
}

func snapshotKeywords(r *model.AnalysisResult) []model.Keyword {
	if r == nil {
		return nil
	}
	out := make([]model.Keyword, len(r.Keywords))
	copy(out, r.Keywords)
	return out
}

func keywordsSnapshotEqual(a, b []model.Keyword) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Word != b[i].Word || a[i].Score != b[i].Score || a[i].TF != b[i].TF || a[i].IDF != b[i].IDF {
			return false
		}
	}
	return true
}

func findResultByID(snap store.Snapshot, id string) *model.AnalysisResult {
	for _, r := range snap.Results {
		if r != nil && r.ArticleID == id {
			return r
		}
	}
	return nil
}

func TestBugConcur026_HotResultRaceAndPointerMutation(t *testing.T) {
	articleSvc, taskSvc, st := buildTestServices()
	ctx := context.Background()

	totalArticles := 80
	createdIDs := make([]string, 0, totalArticles)

	for i := 0; i < totalArticles; i++ {
		req := model.SubmitArticleRequest{
			Title:   fmt.Sprintf("标题-%d", i),
			Content: sampleContent(i),
		}
		resp, err := articleSvc.Submit(ctx, req)
		if err == nil && resp != nil {
			createdIDs = append(createdIDs, resp.ArticleID)
		}
	}
	if len(createdIDs) < 10 {
		t.Fatalf("准备阶段失败，仅创建 %d 篇文章", len(createdIDs))
	}

	type baseline struct {
		summary       string
		keywords      []model.Keyword
		sentenceCount int
		durationMs    int64
		createdAt     time.Time
	}
	baselines := make(map[string]baseline, len(createdIDs))
	{
		snap := st.Snapshot(ctx)
		for _, id := range createdIDs {
			r := findResultByID(snap, id)
			if r != nil {
				baselines[id] = baseline{
					summary:       r.Summary,
					keywords:      snapshotKeywords(r),
					sentenceCount: r.SentenceCount,
					durationMs:    r.DurationMs,
					createdAt:     r.CreatedAt,
				}
			}
		}
	}

	batchArticles := make([]model.SubmitArticleRequest, 0, 1)
	for k := 0; k < 1; k++ {
		batchArticles = append(batchArticles, model.SubmitArticleRequest{
			Title:   fmt.Sprintf("batch-%d", k),
			Content: sampleContent(9000 + k),
		})
	}
	batchTask, _ := taskSvc.SubmitBatch(ctx, model.BatchSubmitRequest{Articles: batchArticles})

	if batchTask != nil {
		deadline := time.Now().Add(2000 * time.Millisecond)
		for time.Now().Before(deadline) {
			tcur, _ := taskSvc.GetTask(ctx, batchTask.ID)
			if tcur != nil && (tcur.Status == model.TaskSuccess || tcur.Status == model.TaskFailed) {
				break
			}
			time.Sleep(30 * time.Millisecond)
		}
	}

	var panicCount int64
	var readerWg sync.WaitGroup
	readerGoroutines := 5

	for r := 0; r < readerGoroutines; r++ {
		readerWg.Add(1)
		go func(rank int) {
			defer readerWg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					atomic.AddInt64(&panicCount, 1)
				}
			}()
			shuffled := make([]string, len(createdIDs))
			copy(shuffled, createdIDs)
			sort.SliceStable(shuffled, func(i, j int) bool {
				return (i+rank)%len(shuffled) < (j+rank)%len(shuffled)
			})
			for it := 0; it < 40; it++ {
				snap := st.Snapshot(ctx)
				for _, id := range shuffled {
					r := findResultByID(snap, id)
					if r != nil {
						_ = len(r.Keywords)
						_ = r.Summary
						_ = r.DurationMs
					}
				}
			}
		}(r)
	}

	var touchWg sync.WaitGroup
	touchWg.Add(1)
	go func() {
		defer touchWg.Done()
		defer func() {
			if rec := recover(); rec != nil {
				atomic.AddInt64(&panicCount, 1)
			}
		}()
		iters := 80
		shuffled := make([]string, len(createdIDs))
		copy(shuffled, createdIDs)
		for it := 0; it < iters; it++ {
			sort.SliceStable(shuffled, func(i, j int) bool {
				return (i+it)%len(shuffled) < (j+it)%len(shuffled)
			})
			for _, id := range shuffled {
				_, _ = articleSvc.GetResult(ctx, id)
			}
		}
	}()

	touchWg.Wait()
	readerWg.Wait()
	time.Sleep(400 * time.Millisecond)

	baselineMismatches := 0
	consecutiveInconsistent := 0

	snap1 := st.Snapshot(ctx)
	time.Sleep(20 * time.Millisecond)
	snap2 := st.Snapshot(ctx)

	for _, id := range createdIDs {
		base, ok := baselines[id]
		if !ok {
			continue
		}
		r1 := findResultByID(snap1, id)
		r2 := findResultByID(snap2, id)
		if r1 == nil || r2 == nil {
			continue
		}

		if r1.SentenceCount != base.sentenceCount {
			baselineMismatches++
		}
		if r1.DurationMs != base.durationMs {
			baselineMismatches++
		}
		if len(r1.Keywords) != len(base.keywords) {
			baselineMismatches++
		}
		if !keywordsSnapshotEqual(r1.Keywords, base.keywords) {
			baselineMismatches++
		}
		if r1.Summary != base.summary {
			baselineMismatches++
		}

		if r1.Summary != r2.Summary {
			consecutiveInconsistent++
		}
		if !keywordsSnapshotEqual(r1.Keywords, r2.Keywords) {
			consecutiveInconsistent++
		}
		if r1.DurationMs != r2.DurationMs {
			consecutiveInconsistent++
		}
		if r1.SentenceCount != r2.SentenceCount {
			consecutiveInconsistent++
		}
	}

	totalPanic := atomic.LoadInt64(&panicCount)

	t.Logf("baseline_mismatches=%d consecutive_inconsistent=%d panics=%d created=%d batchTask=%v",
		baselineMismatches, consecutiveInconsistent, totalPanic, len(createdIDs), batchTask != nil)

	defectDetected := false
	reason := ""
	if totalPanic > 0 {
		defectDetected = true
		reason = fmt.Sprintf("检测到 recoverable panic=%d 次，无锁缓存访问或共享指针并发写触发运行时异常", totalPanic)
	} else if baselineMismatches > 0 {
		defectDetected = true
		reason = fmt.Sprintf("基准数据(Submit之后立即快照)被污染 %d 处（Summary/Keywords/SentenceCount/DurationMs 任意字段相对基线漂移）。原因：Submit 保存结果后就地修改同一共享指针，后续 GetResult 再对该指针做排序/去重/裁剪前缀，后台 batch 任务也在修改关键词TF/Score、DurationMs、SentenceCount 与 Summary 追加标记，三处写入同时污染同一份 AnalysisResult 内存", baselineMismatches)
	} else if consecutiveInconsistent > 0 {
		defectDetected = true
		reason = fmt.Sprintf("短时间内连续两次快照同一文章返回不一致 %d 处（Summary/Keywords/DurationMs/SentenceCount），说明共享 AnalysisResult 指针仍被后台或 GetResult 就地操作持续修改，底层状态随调用漂移", consecutiveInconsistent)
	}

	if defectDetected {
		fmt.Printf("RED（红灯，缺陷未修复）—— %s\n", reason)
		t.Errorf("RED（红灯，缺陷未修复）: %s", reason)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）—— 并发场景无 panic、基准数据稳定未被污染、连续两次读取完全一致，缓存路径与指针处理均安全")
	}
}
