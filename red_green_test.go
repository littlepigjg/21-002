package summarizer

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func buildServicesForRaceTest() (*service.ArticleService, *service.HistoryService, *service.TaskService) {
	ids := store.NewIDGenerator()
	mem := store.NewMemoryStore()
	stopwords := textutil.NewStopwordSet()
	pre := service.NewPreprocessor(stopwords)
	tfidf := service.NewTfidfService(10)
	tr := service.NewTextRankService(30, 0.85)
	sum := service.NewSummarizeService(3)
	analyzer := service.NewAnalyzer(pre, tfidf, tr, sum)
	coord := service.NewAnalysisCoordinator(2048)
	articleSvc := service.NewArticleService(mem, mem, analyzer, ids, coord, 100000)
	historySvc := service.NewHistoryService(mem, mem, coord)
	taskSvc := service.NewTaskService(mem, mem, mem, analyzer, ids, nil, coord, 100000)
	return articleSvc, historySvc, taskSvc
}

func raceSampleContent(i int) string {
	return fmt.Sprintf("这是第 %d 篇用于测试并发读写竞态的文章正文。"+
		"系统需要同时提交文章、写入结果并反复查询历史列表。"+
		"每个句子都应该被正确切分、分词、打分并参与最终的关键词统计。"+
		"我们希望在线程安全的前提下，高并发下依然能稳定工作。"+
		"另外每篇文章尽量保持长度不一，以便触发更多边界情况。"+
		"关键词、摘要、时长字段的可见性必须是线程安全的。文章序号 %d。", i, i)
}

func TestBugConcurSumm28_CoordinatorSharedStateRace(t *testing.T) {
	articleSvc, historySvc, _ := buildServicesForRaceTest()

	ctx := context.Background()

	for i := 0; i < 8; i++ {
		req := model.SubmitArticleRequest{
			Title:   fmt.Sprintf("seed-title-%d", i),
			Content: raceSampleContent(i),
		}
		if _, err := articleSvc.Submit(ctx, req); err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("RED（红灯，缺陷未修复）：seed submit 失败，err=%v", err)
		}
	}

	var wg sync.WaitGroup
	workers := 16
	loops := 80
	var opErrs int64
	var panicked int64

	racePanicHandler := func() {
		if r := recover(); r != nil {
			atomic.AddInt64(&panicked, 1)
			atomic.AddInt64(&opErrs, 1)
			fmt.Fprintf(os.Stderr, "panic caught during concurrent ops: %v\n", r)
		}
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			defer racePanicHandler()
			for k := 0; k < loops; k++ {
				func() {
					defer racePanicHandler()
					req := model.SubmitArticleRequest{
						Title:   fmt.Sprintf("w%d-l%d", wid, k),
						Content: raceSampleContent(wid*10000 + k),
					}
					resp, err := articleSvc.Submit(ctx, req)
					if err != nil {
						atomic.AddInt64(&opErrs, 1)
						return
					}
					if resp == nil {
						atomic.AddInt64(&opErrs, 1)
						return
					}
					_, _ = articleSvc.Get(ctx, resp.ArticleID)
					_, _ = articleSvc.GetResult(ctx, resp.ArticleID)
				}()
			}
		}(w)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer racePanicHandler()
			for k := 0; k < loops; k++ {
				func() {
					defer racePanicHandler()
					entries, _, err := historySvc.List(ctx, 0, 60)
					if err != nil {
						atomic.AddInt64(&opErrs, 1)
						return
					}
					for _, e := range entries {
						if e.Article == nil {
							continue
						}
						_ = e.Article.Status
						if e.Result != nil {
							_ = e.Result.DurationMs
							_ = e.Result.SentenceCount
						}
					}
				}()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer racePanicHandler()
			for k := 0; k < loops; k++ {
				func() {
					defer racePanicHandler()
					_, _, _ = articleSvc.List(ctx, 0, 40)
					_, _, _ = articleSvc.ListResults(ctx, 0, 40)
					_, _ = historySvc.Recent(ctx, 12)
				}()
			}
		}()
	}

	wg.Wait()

	fmt.Println()
	if atomic.LoadInt64(&opErrs) > 0 || atomic.LoadInt64(&panicked) > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("RED（红灯，缺陷未修复）：并发运行期间出现错误 %d 次、panic %d 次（共享 Coordinator 无锁共享状态导致并发写 map / 指针字段竞态，-race 可捕获数据竞争）。",
			atomic.LoadInt64(&opErrs), atomic.LoadInt64(&panicked))
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
