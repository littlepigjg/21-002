package summ27_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/handler"
	"summarizer/internal/model"
	"summarizer/internal/service"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func buildService() (*service.ArticleService, *store.MemoryStore) {
	ids := store.NewIDGenerator()
	mem := store.NewMemoryStore()
	stopwords := textutil.NewStopwordSet()
	pre := service.NewPreprocessor(stopwords)
	tfidf := service.NewTfidfService(10)
	tr := service.NewTextRankService(20, 0.85)
	sum := service.NewSummarizeService(3)
	analyzer := service.NewAnalyzer(pre, tfidf, tr, sum)
	svc := service.NewArticleService(mem, mem, analyzer, ids, 100000)
	return svc, mem
}

func buildHandler() (*handler.ArticleHandler, *store.MemoryStore) {
	svc, mem := buildService()
	return handler.NewArticleHandler(svc), mem
}

type submitResult struct {
	resp *model.AnalyzeResponse
	err  error
}

func runSubmitWithTimeout(svc *service.ArticleService, ctx context.Context, title, content string, timeout time.Duration) (*submitResult, bool) {
	done := make(chan submitResult, 1)
	go func() {
		r, e := svc.Submit(ctx, model.SubmitArticleRequest{Title: title, Content: content})
		done <- submitResult{r, e}
	}()
	select {
	case res := <-done:
		return &res, true
	case <-time.After(timeout):
		return nil, false
	}
}

func TestBugSumm27_DeferResourceLeakAndRace(t *testing.T) {
	bugDetected := false
	var reasons []string

	t.Run("token_exhaustion_via_save_result_fail", func(t *testing.T) {
		svc, mem := buildService()
		ctx := context.Background()

		for i := 0; i < 9; i++ {
			content := fmt.Sprintf("abcde%d", i%10)
			if len([]rune(content)) != 6 {
				extra := 6 - len([]rune(content))
				for j := 0; j < extra; j++ {
					content += "z"
				}
				content = string([]rune(content)[:6])
			}
			_, ok := runSubmitWithTimeout(svc, ctx, "t", content, 900*time.Millisecond)
			if !ok {
				bugDetected = true
				reasons = append(reasons, fmt.Sprintf("[token_exhaustion] iter=%d Submit blocked >900ms, writeSem leaked 8 tokens and exhausted", i))
				break
			}
		}
		if leaks := mem.TokenLeaks(); leaks > 0 {
			bugDetected = true
			reasons = append(reasons, fmt.Sprintf("[token_leak_counter] TokenLeaks=%d > 0 after save-result-fail path submissions", leaks))
		}
	})

	t.Run("analyze_fail_triggers_update_deadlock", func(t *testing.T) {
		svc, _ := buildService()
		ctx := context.Background()

		content := "1234567"
		_, ok := runSubmitWithTimeout(svc, ctx, "t", content, 1500*time.Millisecond)
		if !ok {
			bugDetected = true
			reasons = append(reasons, "[analyze_fail_deadlock] Submit len(rune)=7 blocked >1.5s: UpdateArticle re-locks per-article Mutex held since SaveArticle (recursive lock deadlock)")
		}
	})

	t.Run("success_path_update_deadlock", func(t *testing.T) {
		svc, _ := buildService()
		ctx := context.Background()

		content := "This is a longer article content with many runes beyond ten runes indeed."
		_, ok := runSubmitWithTimeout(svc, ctx, "hello", content, 1500*time.Millisecond)
		if !ok {
			bugDetected = true
			reasons = append(reasons, "[success_path_deadlock] Submit normal content blocked >1.5s: success UpdateArticle re-acquires per-article Mutex held by SaveArticle → deadlock")
		}
	})

	t.Run("service_internal_concurrent_race", func(t *testing.T) {
		svc, _ := buildService()
		ctx := context.Background()

		var wg sync.WaitGroup
		var panicCount int32
		n := 30
		body := "this rune length is clearly more than ten runes for concurrency testing"
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt32(&panicCount, 1)
					}
				}()
				title := fmt.Sprintf("t%d", idx)
				c := fmt.Sprintf("%s v%d", body, idx)
				done := make(chan struct{})
				go func() {
					_, _ = svc.Submit(ctx, model.SubmitArticleRequest{Title: title, Content: c})
					close(done)
				}()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
				}
				_, _, _, _, _ = svc.Stats()
			}(i)
		}
		doneAll := make(chan struct{})
		go func() { wg.Wait(); close(doneAll) }()
		select {
		case <-doneAll:
		case <-time.After(6 * time.Second):
			bugDetected = true
			reasons = append(reasons, "[service_race_stall] 30 concurrent Submit did not finish within 6s: deadlock + resource contention")
		}
		if atomic.LoadInt32(&panicCount) > 0 {
			bugDetected = true
			reasons = append(reasons, fmt.Sprintf("[service_race_panic] concurrent goroutines panicked %d times during Submit+Stats (unsynchronized submitCount/failCount/lastSubmitIDs)", atomic.LoadInt32(&panicCount)))
		}
	})

	t.Run("handler_concurrent_map_race_and_stall", func(t *testing.T) {
		h, _ := buildHandler()

		n := 40
		var wg sync.WaitGroup
		var panicCount int32
		var okCount int32

		start := make(chan struct{})
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt32(&panicCount, 1)
					}
				}()
				<-start
				body := fmt.Sprintf(`{"title":"title-%d","content":"content value rune length more than ten %d"}`, idx, idx)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewBufferString(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				h.Submit(w, req)
				atomic.AddInt32(&okCount, 1)
				if idx%2 == 0 {
					_, _, _ = h.HandlerStats()
				}
				if idx%7 == 0 {
					failBody := `{"title":"dup","content":"shared-fp-content"}`
					req2 := httptest.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewBufferString(failBody))
					req2.Header.Set("Content-Type", "application/json")
					w2 := httptest.NewRecorder()
					h.Submit(w2, req2)
				}
			}(i)
		}
		close(start)

		doneAll := make(chan struct{})
		go func() { wg.Wait(); close(doneAll) }()

		stalled := false
		select {
		case <-doneAll:
		case <-time.After(8 * time.Second):
			stalled = true
		}
		if stalled {
			bugDetected = true
			reasons = append(reasons, fmt.Sprintf("[handler_map_stall] 40 concurrent Submit handler calls did not finish within 8s: deadlock/map write panic blocked progress (finished=%d panic=%d)", atomic.LoadInt32(&okCount), atomic.LoadInt32(&panicCount)))
		}
		if atomic.LoadInt32(&panicCount) > 0 {
			bugDetected = true
			reasons = append(reasons, fmt.Sprintf("[handler_map_panic] concurrent map read/write triggered %d panic(s) (failureLookup / seenIDs are package-level maps without mutex protection)", atomic.LoadInt32(&panicCount)))
		}
	})

	fmt.Println()
	if bugDetected {
		fmt.Println("==================================================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("------------------------------------------------------------------")
		for _, r := range reasons {
			fmt.Println("  - " + r)
		}
		fmt.Println("==================================================================")
		t.Errorf("RED（红灯，缺陷未修复）—— %d issue(s) detected", len(reasons))
	} else {
		fmt.Println("==================================================================")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("==================================================================")
	}
}
