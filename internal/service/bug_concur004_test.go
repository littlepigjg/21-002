package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func TestBugConcur004_IDConflict(t *testing.T) {
	const iterations = 50
	totalDups := 0

	for iter := 0; iter < iterations; iter++ {
		ids := store.NewIDGenerator()
		const goroutines = 300
		seenIDs := make([]string, goroutines)

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				seenIDs[idx] = ids.Next("art")
			}(i)
		}
		wg.Wait()

		unique := make(map[string]struct{})
		for _, id := range seenIDs {
			if _, exists := unique[id]; exists {
				totalDups++
			}
			unique[id] = struct{}{}
		}
		runtime.Gosched()
	}

	if totalDups > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）: detected %d duplicate IDs across %d iterations\n", totalDups, iterations)
		t.Fatalf("RED: %d duplicate IDs detected", totalDups)
	}
	fmt.Printf("GREEN（绿灯，缺陷已修复）: no duplicates across %d iterations\n", iterations)
}

func TestBugConcur004_DataOverwrite(t *testing.T) {
	const goroutines = 200
	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()
	preprocessor := NewPreprocessor(textutil.NewStopwordSet())
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(20, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)
	articleSvc := NewArticleService(memStore, memStore, analyzer, ids, 100000)

	var errCount atomic.Int32
	var idStore sync.Map

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			title := fmt.Sprintf("Article %d", idx)
			content := fmt.Sprintf("Content of article %d for concurrent testing with enough text for analysis.", idx)
			req := model.SubmitArticleRequest{Title: title, Content: content}
			resp, err := articleSvc.Submit(context.Background(), req)
			if err != nil {
				errCount.Add(1)
				return
			}
			idStore.Store(resp.ArticleID, title)
		}(i)
	}
	wg.Wait()

	idCount := 0
	idStore.Range(func(key, value any) bool {
		idCount++
		return true
	})

	errs := int(errCount.Load())
	_, total, _ := memStore.ListArticles(context.Background(), 0, 10000)

	if idCount < goroutines || errs > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）: submitted=%d stored_unique=%d errors=%d total_entries=%d\n",
			goroutines, idCount, errs, total)
		t.Fatalf("RED: data overwrite or errors detected")
	}
	fmt.Printf("GREEN（绿灯，缺陷已修复）: all %d articles stored correctly\n", goroutines)
}

func TestBugConcur004_StressOverwrite(t *testing.T) {
	const rounds = 20
	const perRound = 50
	totalSubmitted := 0
	totalErrors := 0

	ids := store.NewIDGenerator()
	memStore := store.NewMemoryStore()
	preprocessor := NewPreprocessor(textutil.NewStopwordSet())
	tfidf := NewTfidfService(10)
	textrank := NewTextRankService(20, 0.85)
	summarizer := NewSummarizeService(3)
	analyzer := NewAnalyzer(preprocessor, tfidf, textrank, summarizer)
	articleSvc := NewArticleService(memStore, memStore, analyzer, ids, 100000)

	for r := 0; r < rounds; r++ {
		var wg sync.WaitGroup
		var errCount atomic.Int32
		wg.Add(perRound)
		for i := 0; i < perRound; i++ {
			go func(idx int) {
				defer wg.Done()
				title := fmt.Sprintf("R%d-Article %d", r, idx)
				content := fmt.Sprintf("Content for round %d article %d with sufficient text.", r, idx)
				req := model.SubmitArticleRequest{Title: title, Content: content}
				_, err := articleSvc.Submit(context.Background(), req)
				if err != nil {
					errCount.Add(1)
				}
			}(i)
		}
		wg.Wait()
		totalSubmitted += perRound
		totalErrors += int(errCount.Load())
		runtime.Gosched()
	}

	_, total, _ := memStore.ListArticles(context.Background(), 0, 100000)
	expected := totalSubmitted - totalErrors

	if total != expected || totalErrors > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）: expected=%d actual=%d errors=%d\n",
			expected, total, totalErrors)
		t.Fatalf("RED: data inconsistency detected")
	}
	fmt.Printf("GREEN（绿灯，缺陷已修复）: all %d articles correct across %d rounds\n", total, rounds)
}
