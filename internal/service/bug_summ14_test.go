package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"summarizer/internal/cache"
	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/taskqueue"
	"summarizer/internal/textutil"
)

func buildBuggyService() (*TaskService, *store.MemoryStore, *cache.ResultCache) {
	memStore := store.NewMemoryStore()
	ids := store.NewIDGenerator()
	stopwords := textutil.NewStopwordSet()
	preprocessor := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(30, 0.85)
	summarizer := NewSummarizeService(5)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)
	queue := taskqueue.NewQueue(64)
	resultCache := cache.NewResultCache(1024)
	svc := NewTaskService(memStore, memStore, memStore, analyzer, ids, queue, 100000, resultCache)
	return svc, memStore, resultCache
}

func seedSharedArticles(t *testing.T, memStore *store.MemoryStore, ids *store.IDGenerator, count int) []string {
	ctx := context.Background()
	articleIDs := make([]string, count)
	for i := 0; i < count; i++ {
		id := ids.Next("art")
		a := &model.Article{
			ID:        id,
			Title:     fmt.Sprintf("title-%d", i),
			Content:   "Hello world. This is a test. The quick brown fox jumps over the lazy dog. Machine learning is interesting. Natural language processing is powerful.",
			Status:    model.ArticlePending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := memStore.SaveArticle(ctx, a); err != nil {
			t.Fatalf("seed article: %v", err)
		}
		articleIDs[i] = id
	}
	return articleIDs
}

func TestBugSumm14_SharedPointerMutationRace(t *testing.T) {
	totalFailures := 0
	var mu sync.Mutex
	report := func(format string, args ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		totalFailures++
		t.Logf(format, args...)
	}

	const rounds = 50
	for round := 0; round < rounds; round++ {
		svc, memStore, rc := buildBuggyService()
		ctx := context.Background()

		ids := store.NewIDGenerator()
		sharedIDs := seedSharedArticles(t, memStore, ids, 3)

		rc.Warmup(sharedIDs, time.Now())

		task1 := &model.Task{
			ID:         ids.Next("task"),
			Type:       model.TaskBatch,
			Status:     model.TaskPending,
			ArticleIDs: append([]string{}, sharedIDs...),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		task2 := &model.Task{
			ID:         ids.Next("task"),
			Type:       model.TaskBatch,
			Status:     model.TaskPending,
			ArticleIDs: append([]string{}, sharedIDs...),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := memStore.SaveTask(ctx, task1); err != nil {
			t.Fatalf("save task1: %v", err)
		}
		if err := memStore.SaveTask(ctx, task2); err != nil {
			t.Fatalf("save task2: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer func() {
				if rr := recover(); rr != nil {
					report("round=%d processBatch(task1) panicked: %v", round, rr)
				}
			}()
			_ = svc.processBatch(ctx, task1)
		}()
		go func() {
			defer wg.Done()
			defer func() {
				if rr := recover(); rr != nil {
					report("round=%d processBatch(task2) panicked: %v", round, rr)
				}
			}()
			_ = svc.processBatch(ctx, task2)
		}()
		wg.Wait()

		for _, aid := range sharedIDs {
			r, err := memStore.GetResult(ctx, aid)
			if err != nil {
				report("round=%d missing result for article %s: %v", round, aid, err)
				continue
			}
			if r == nil {
				report("round=%d nil result for article %s", round, aid)
				continue
			}

			markerCount := 0
			for _, kw := range r.Keywords {
				if kw.Word == "__processed" {
					markerCount++
				}
			}
			if markerCount > 1 {
				report("round=%d article %s got %d __processed markers (expected <=1), cache/shared pointer append race confirmed",
					round, aid, markerCount)
			}

			taggedSummary := r.Summary
			tagCount := strings.Count(taggedSummary, "[task:")
			if tagCount > 1 {
				report("round=%d article %s summary has %d [task:] prefixes (expected 1 or 0): %q",
					round, aid, tagCount, taggedSummary)
			}
		}
	}

	if totalFailures > 0 {
		fmt.Printf("RED（红灯，缺陷未修复） - %d corruption/race cases observed over %d rounds\n", totalFailures, rounds)
		t.Fatalf("RED（红灯，缺陷未修复）: detected %d data-corruption / shared-pointer race failures across %d rounds", totalFailures, rounds)
	}
	fmt.Printf("GREEN（绿灯，缺陷已修复） - no corruption observed over %d rounds\n", rounds)
}
