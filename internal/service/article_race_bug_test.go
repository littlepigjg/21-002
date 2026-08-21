package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func buildSvc() (*ArticleService, *store.MemoryStore) {
	stopwords := textutil.NewStopwordSet()
	pre := NewPreprocessor(stopwords)
	tfidf := NewTfidfService(8)
	tr := NewTextRankService(20, 0.85)
	sum := NewSummarizeService(3)
	analyzer := NewAnalyzer(pre, tfidf, tr, sum)
	ids := store.NewIDGenerator()
	ms := store.NewMemoryStore()
	svc := NewArticleService(ms, ms, analyzer, ids, 200000)
	return svc, ms
}

func seedN(svc *ArticleService, n int) ([]string, []string, error) {
	ids := make([]string, 0, n)
	titles := make([]string, 0, n)
	for i := 0; i < n; i++ {
		title := fmt.Sprintf("Title-Article-%d", i)
		content := fmt.Sprintf(
			"这是编号为 %d 的测试文章。内容非常丰富。它包含多个句子。每一句都值得被分析。系统需要能够正确处理这些中文语句。关键词重复足够多次。中文中文中文。测试测试测试。",
			i,
		)
		resp, err := svc.Submit(context.Background(), model.SubmitArticleRequest{
			Title:   title,
			Content: content,
		})
		if err != nil {
			return nil, nil, err
		}
		ids = append(ids, resp.ArticleID)
		titles = append(titles, title)
	}
	return ids, titles, nil
}

func drainWg(svc *ArticleService) {
	done := make(chan struct{})
	go func() {
		svc.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(4 * time.Second):
	}
}

func reportRedAndExit(t *testing.T, reason string) {
	fmt.Println("RED（红灯，缺陷未修复）")
	fmt.Printf("  reason: %s\n", reason)
	t.Fatalf("RED: %s", reason)
}

func TestBugSumm23_HotCacheRaceAndCtxCancel(t *testing.T) {
	svc, ms := buildSvc()
	ids, titles, err := seedN(svc, 8)
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	drainWg(svc)

	{
		ctxCanceled, cancel := context.WithCancel(context.Background())
		cancel()
		citems, ctotal, cerr := svc.List(ctxCanceled, 0, 100)
		if cerr == nil && ctotal >= len(ids) && len(citems) >= len(ids) {
			reportRedAndExit(t, fmt.Sprintf("canceled ctx List 忽略 ctx.Err() 返回数据: err=%v total=%d n=%d", cerr, ctotal, len(citems)))
		}
	}

	{
		ctxDeadlined, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		time.Sleep(time.Millisecond)
		defer cancel()
		ditems, dtotal, derr := svc.List(ctxDeadlined, 0, 100)
		if derr == nil && dtotal >= len(ids) && len(ditems) >= len(ids) {
			reportRedAndExit(t, fmt.Sprintf("deadline ctx List 忽略 ctx.Err() 返回数据: err=%v total=%d n=%d", derr, dtotal, len(ditems)))
		}
	}

	{
		ctxCanceled, cancel := context.WithCancel(context.Background())
		cancel()
		_, gerr := svc.Get(ctxCanceled, ids[0])
		if gerr == nil {
			reportRedAndExit(t, fmt.Sprintf("canceled ctx Get 忽略 ctx.Err() 返回成功 err=%v", gerr))
		}
	}

	before := make([]string, len(ids))
	for i, id := range ids {
		a, e := ms.GetArticle(context.Background(), id)
		if e != nil {
			continue
		}
		before[i] = a.Title
	}

	_ = titles
	time.Sleep(10 * time.Millisecond)

	const concurrency = 4
	const rounds = 3
	failCh := make(chan string, concurrency*rounds*16)

	wrap := func(fn func()) {
		defer func() {
			if r := recover(); r != nil {
				select {
				case failCh <- fmt.Sprintf("panic: %v", r):
				default:
				}
			}
		}()
		fn()
	}

	for r := 0; r < rounds; r++ {
		var wg sync.WaitGroup
		for c := 0; c < concurrency; c++ {
			wg.Add(1)
			go func(round, idx int) {
				defer wg.Done()
				wrap(func() {
					ctx, cc := context.WithCancel(context.Background())
					if (round+idx)%4 == 0 {
						cc()
					} else {
						defer cc()
					}
					_, _, e := svc.List(ctx, 0, 20)
					if e != nil && ctx.Err() == nil {
						failCh <- fmt.Sprintf("list err: %v", e)
					}
				})
			}(r, c)

			wg.Add(1)
			go func(round, idx int) {
				defer wg.Done()
				wrap(func() {
					ctx, cc := context.WithTimeout(context.Background(), 2*time.Millisecond)
					defer cc()
					a, e := svc.Get(ctx, ids[(round*3+idx)%len(ids)])
					if e == nil && a != nil && strings.Contains(a.Title, "Title-Article-") {
						if !strings.HasPrefix(a.Title, "Title-Article-") {
							failCh <- fmt.Sprintf("prefix corrupt: %q", a.Title)
						}
					}
				})
			}(r, c)

			wg.Add(1)
			go func(round, idx int) {
				defer wg.Done()
				wrap(func() {
					ctx := context.Background()
					if (round+idx)%7 == 0 {
						var cc context.CancelFunc
						ctx, cc = context.WithCancel(ctx)
						cc()
					}
					_, e := svc.Submit(ctx, model.SubmitArticleRequest{
						Title:   fmt.Sprintf("Concurrent-%d-%d", round, idx),
						Content: "并发提交。中文句子。关键词关键词关键词。重复足够多。内容要够长便于分句。",
					})
					_ = e
				})
			}(r, c)

			wg.Add(1)
			go func(round, idx int) {
				defer wg.Done()
				wrap(func() {
					offset := (round + idx) % 3
					items, _, e := svc.List(context.Background(), offset, 6)
					if e != nil {
						failCh <- fmt.Sprintf("list bg: %v", e)
						return
					}
					for _, it := range items {
						if it == nil {
							failCh <- "nil article"
						}
					}
				})
			}(r, c)
		}
		wg.Wait()
	}

	drainWg(svc)
	time.Sleep(50 * time.Millisecond)

	after := make([]string, len(ids))
	for i, id := range ids {
		a, e := ms.GetArticle(context.Background(), id)
		if e != nil {
			continue
		}
		after[i] = a.Title
	}
	titleChanged := 0
	for i := range after {
		if after[i] != before[i] {
			titleChanged++
		}
	}

	close(failCh)
	fails := make([]string, 0, len(failCh))
	for f := range failCh {
		fails = append(fails, f)
	}

	if titleChanged > 0 || len(fails) > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("  titleChanged=%d asyncFailures=%d\n", titleChanged, len(fails))
		for i, f := range fails {
			if i >= 5 {
				break
			}
			fmt.Printf("  fail_sample[%d]: %s\n", i, f)
		}
		t.Fatalf("RED: titleChanged=%d asyncFailures=%d", titleChanged, len(fails))
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
