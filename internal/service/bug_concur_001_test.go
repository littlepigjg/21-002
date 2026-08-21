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

func TestBugConcur001_TaskPointerDataRace(t *testing.T) {
	sw := textutil.NewStopwordSet()
	prep := NewPreprocessor(sw)
	tfidf := NewTfidfService(10)
	tr := NewTextRankService(30, 0.85)
	summ := NewSummarizeService(5)
	analyzer := NewAnalyzer(prep, tfidf, tr, summ)

	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	queue := taskqueue.NewQueue(8)

	svc := NewTaskService(
		memStore, memStore, memStore,
		analyzer, ids, queue, 100000,
	)

	ctx := context.Background()

	mgr := taskqueue.NewManager(queue, 1, func(ctx context.Context, j taskqueue.Job) error {
		if j.Run == nil {
			return nil
		}
		return svc.HandleJob(ctx, j)
	})
	mgr.Start(ctx)
	defer mgr.Stop()

	req := model.BatchSubmitRequest{
		Articles: []model.SubmitArticleRequest{
			{Title: "AI", Content: "人工智能正在改变世界。深度学习是机器学习的一个分支。自然语言处理技术日趋成熟。大数据分析推动了行业变革。机器学习算法不断优化。智能语音助手越来越普及。"},
			{Title: "分布式", Content: "分布式系统面临一致性和可用性的挑战。CAP定理描述了这些权衡。微服务架构越来越流行。容器化部署提高了可维护性。消息队列解耦了系统组件。"},
			{Title: "云原生", Content: "云原生技术栈包括容器化和编排。Kubernetes已成为事实标准。服务网格提供了新的网络抽象。可观测性是运维的基础。无服务器架构降低了运维成本。"},
			{Title: "安全", Content: "网络安全形势日益严峻。零信任架构逐步推广。加密技术保障数据安全。身份认证是第一道防线。安全审计帮助发现潜在风险。漏洞扫描是常规操作。"},
		},
	}

	task, err := svc.SubmitBatch(ctx, req)
	if err != nil {
		t.Fatalf("SubmitBatch failed: %v", err)
	}

	var wg sync.WaitGroup
	var readsDone int64
	readerCount := 32
	readsPerGoroutine := 2000

	for i := 0; i < readerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < readsPerGoroutine; j++ {
				tk, err := svc.GetTask(ctx, task.ID)
				if err != nil {
					continue
				}
				_ = tk.Status
				_ = tk.Progress
				_ = tk.Error
				_ = tk.UpdatedAt
				atomic.AddInt64(&readsDone, 1)
			}
		}()
	}

	var statusCheckWg sync.WaitGroup
	var reachedTerminal atomic.Bool
	statusCheckWg.Add(1)
	go func() {
		defer statusCheckWg.Done()
		deadline := time.After(30 * time.Second)
		for {
			select {
			case <-deadline:
				return
			default:
			}
			tk, err := svc.GetTask(ctx, task.ID)
			if err == nil && tk.Status.IsTerminal() {
				reachedTerminal.Store(true)
				return
			}
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()
	statusCheckWg.Wait()

	totalReads := atomic.LoadInt64(&readsDone)
	_ = fmt.Sprintf("task_id=%s total_reads=%d", task.ID, totalReads)

	if !reachedTerminal.Load() {
		fmt.Println("RED（红灯，缺陷未修复）：任务未达到终态")
		t.Fail()
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）：任务正常完成，无数据竞争")
}
