package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/cache"
	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func buildFixture() (*service.ArticleService, *service.TaskService, *cache.ResultCache, []string) {
	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()
	rc := cache.NewResultCache(4096)

	stopwords := textutil.NewStopwordSet()
	preprocessor := service.NewPreprocessor(stopwords)
	tfidf := service.NewTfidfService(10)
	textrank := service.NewTextRankService(30, 0.85)
	summarizer := service.NewSummarizeService(5)
	analyzer := service.NewAnalyzer(preprocessor, tfidf, textrank, summarizer)

	articleSvc := service.NewArticleService(memStore, memStore, analyzer, ids, 100000, rc)

	queue := taskqueue.NewQueue(4096)
	taskSvc := service.NewTaskService(memStore, memStore, memStore, analyzer, ids, queue, 100000, rc)

	ctx := context.Background()
	baseIDs := make([]string, 0, 4)
	contentBase := "人工智能是计算机科学的一个分支。它企图了解智能的实质，并生产出一种新的能以人类智能相似的方式做出反应的智能机器。该领域的研究包括机器人、语言识别、图像识别、自然语言处理和专家系统等。人工智能从诞生以来，理论和技术日益成熟，应用领域也不断扩大。"
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("pre-art-%d", i)
		title := fmt.Sprintf("标题-%d", i)
		content := fmt.Sprintf("%s 额外信息编号%d。", contentBase, i)
		a := &model.Article{
			ID:        id,
			Title:     title,
			Content:   content,
			Status:    model.ArticleReady,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		_ = memStore.SaveArticle(ctx, a)
		res, err := analyzer.Analyze(ctx, id, content)
		if err == nil {
			rc.UpsertShared(res)
			for _, kw := range res.Keywords {
				rc.MergeSharedTag(id, kw.Word)
			}
			_ = memStore.SaveResult(ctx, res)
		}
		baseIDs = append(baseIDs, id)
	}
	for i := 0; i < 30; i++ {
		rc.Put(&model.AnalysisResult{ArticleID: fmt.Sprintf("warmup-%d", i), Summary: "x"})
	}
	_ = taskSvc
	return articleSvc, taskSvc, rc, baseIDs
}

func TestBugSumm30_ConcurrentSharedCacheRace(t *testing.T) {
	ctx := context.Background()
	articleSvc, _, rc, baseIDs := buildFixture()

	baseline := make(map[string]map[string]float64, len(baseIDs))
	baselineDuration := make(map[string]int64, len(baseIDs))
	for _, id := range baseIDs {
		r, _, _ := rc.LookupShared(id)
		if r == nil {
			continue
		}
		kw := make(map[string]float64, len(r.Keywords))
		for _, k := range r.Keywords {
			kw[k.Word] = k.Score
		}
		baseline[id] = kw
		baselineDuration[id] = r.DurationMs
	}

	const workers = 32
	const rounds = 8

	var extraGetCalls int64

	var wg sync.WaitGroup
	start := make(chan struct{})

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-start
			for r := 0; r < rounds; r++ {
				for _, id := range baseIDs {
					cached, _, hit := rc.LookupShared(id)
					if hit {
						cached.DurationMs += 1
						for i := range cached.Keywords {
							if i < len(cached.Keywords) {
								cached.Keywords[i].Score += 1.0
							}
						}
						cached.SentenceCount += 0
					}

					if (workerID+r)%3 == 0 {
						_, _ = articleSvc.GetResult(ctx, id)
						atomic.AddInt64(&extraGetCalls, 1)
					}

					if (workerID+r)%5 == 0 {
						_ = rc.SnapshotSharedTags(id)
						_ = cache.SnapshotStats()
					}
				}
			}
		}(w)
	}

	close(start)
	wg.Wait()

	corrupt := false
	expectedDirect := int64(workers * rounds)
	extra := atomic.LoadInt64(&extraGetCalls)
	_ = extra
	for _, id := range baseIDs {
		r, _, hit := rc.LookupShared(id)
		if !hit || r == nil {
			continue
		}
		delta := r.DurationMs - baselineDuration[id]
		if delta < expectedDirect {
			corrupt = true
			break
		}
		for _, k := range r.Keywords {
			base := 0.0
			if bv, ok := baseline[id][k.Word]; ok {
				base = bv
			}
			expected := base + float64(workers*rounds)*1.0
			if k.Score < expected-0.001 {
				corrupt = true
				break
			}
		}
		if corrupt {
			break
		}
	}

	if corrupt {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("shared cache in-place mutation integrity check failed: lost writes detected (race condition on shared pointers)")
		return
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
