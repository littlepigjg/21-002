package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/model"
	"summarizer/internal/store"
	"summarizer/internal/textutil"
)

func setupBugStore(ctx context.Context, n int) (*store.MemoryStore, []string) {
	ms := store.NewMemoryStore()
	ids := make([]string, 0, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("art-bug-%d-%d", i, time.Now().UnixNano())
		a := &model.Article{
			ID:        id,
			Title:     fmt.Sprintf("Title %d", i),
			Content:   fmt.Sprintf("content for article %d with several words to make it non empty. "+
				"Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt.", i),
			Status:    model.ArticleReady,
			CreatedAt: now,
			UpdatedAt: now,
		}
		_ = ms.SaveArticle(ctx, a)
		ids = append(ids, id)
	}
	return ms, ids
}

type bugTestResult struct {
	firstPanic         string
	panics             int
	leakedAfterStress  int
	leakedAfterSolo    int
	leakedAfterBatch   int
	totalLeak          int
}

func runBugDefRaceInner(t *testing.T) bugTestResult {
	defer func() {
		_ = textutil.ForceCloseAllHandles()
		textutil.ResetWriteCount()
	}()
	_ = textutil.ForceCloseAllHandles()
	textutil.ResetWriteCount()

	ctx := context.Background()
	baseDir := t.TempDir()

	ms, ids := setupBugStore(ctx, 40)
	expSvc := NewExportService(ms)
	expSvc.SetProgressLog(baseDir + "/runner.log")

	var panicCount int32
	var mu sync.Mutex
	firstPanicLocal := ""

	concurrency := 10
	rounds := 3
	var wg sync.WaitGroup
	for r := 0; r < rounds; r++ {
		for c := 0; c < concurrency; c++ {
			wg.Add(1)
			go func(round, idx int) {
				defer wg.Done()
				defer func() {
					if rec := recover(); rec != nil {
						atomic.AddInt32(&panicCount, 1)
						mu.Lock()
						if firstPanicLocal == "" {
							firstPanicLocal = fmt.Sprintf("%v", rec)
						}
						mu.Unlock()
					}
				}()

				subDir := fmt.Sprintf("%s/r%d_c%d", baseDir, round, idx)
				_ = os.MkdirAll(subDir, 0o755)

				start := (round*concurrency + idx) % len(ids)
				size := 10
				pick := make([]string, 0, size)
				for k := 0; k < size; k++ {
					pick = append(pick, ids[(start+k)%len(ids)])
				}
				pick = append(pick, fmt.Sprintf("missing-%d-%d", round, idx))

				_, _ = expSvc.ExportPerArticle(ctx, PerArticleExportRequest{
					ArticleIDs: pick,
					OutputDir:  subDir,
					Format:     "json",
				})

				_, _ = expSvc.BatchExportWithProgress(ctx, subDir,
					[]string{"json", "csv", "txt"}, 2)

				extraPath := subDir + "/extra.log"
				for w := 0; w < 8; w++ {
					_ = textutil.WriteShared(extraPath, fmt.Sprintf("tick %d\n", w))
				}
			}(r, c)
		}
	}
	wg.Wait()

	leakedAfterStress := textutil.SharedHandleCount()
	_ = textutil.ForceCloseAllHandles()

	soloDir := baseDir + "/solo"
	_ = os.MkdirAll(soloDir, 0o755)
	soloIds := ids[:18]
	soloIds = append(soloIds, "solo-missing-1", "solo-missing-2")

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				atomic.AddInt32(&panicCount, 1)
				mu.Lock()
				if firstPanicLocal == "" {
					firstPanicLocal = fmt.Sprintf("%v", rec)
				}
				mu.Unlock()
			}
		}()
		_, _ = expSvc.ExportPerArticle(ctx, PerArticleExportRequest{
			ArticleIDs: soloIds,
			OutputDir:  soloDir,
			Format:     "txt",
		})
	}()

	leakedAfterSolo := textutil.SharedHandleCount()
	_ = textutil.ForceCloseAllHandles()

	batchDir := baseDir + "/batch_solo"
	_ = os.MkdirAll(batchDir, 0o755)
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				atomic.AddInt32(&panicCount, 1)
				mu.Lock()
				if firstPanicLocal == "" {
					firstPanicLocal = fmt.Sprintf("%v", rec)
				}
				mu.Unlock()
			}
		}()
		_, _ = expSvc.BatchExportWithProgress(ctx, batchDir,
			[]string{"json", "csv", "txt"}, 3)
	}()

	leakedAfterBatch := textutil.SharedHandleCount()
	_ = textutil.ForceCloseAllHandles()

	totalLeak := leakedAfterStress + leakedAfterSolo + leakedAfterBatch
	panics := int(atomic.LoadInt32(&panicCount))
	return bugTestResult{
		firstPanic:        firstPanicLocal,
		panics:            panics,
		leakedAfterStress: leakedAfterStress,
		leakedAfterSolo:   leakedAfterSolo,
		leakedAfterBatch:  leakedAfterBatch,
		totalLeak:         totalLeak,
	}
}

func TestBugDefRaceSharedHandleLeakExportBatch(t *testing.T) {
	oldMax := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(oldMax)

	var res bugTestResult
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				if res.panics == 0 {
					res.panics = 1
					res.firstPanic = fmt.Sprintf("%v", rec)
				}
			}
		}()
		res = runBugDefRaceInner(t)
	}()

	fmt.Printf("=== BUG TEST SUMMARY ===\n")
	fmt.Printf("panic count       = %d\n", res.panics)
	fmt.Printf("first panic       = %q\n", res.firstPanic)
	fmt.Printf("leakedAfterStress = %d\n", res.leakedAfterStress)
	fmt.Printf("leakedAfterSolo   = %d\n", res.leakedAfterSolo)
	fmt.Printf("leakedAfterBatch  = %d\n", res.leakedAfterBatch)
	fmt.Printf("totalLeak         = %d\n", res.totalLeak)

	if res.panics > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）: 共享句柄池并发访问触发 panic，次数 = %d，首次 panic: %s\n", res.panics, res.firstPanic)
		t.Errorf("RED（红灯，缺陷未修复）: 并发崩溃 panics=%d first=%q", res.panics, res.firstPanic)
		return
	}
	if res.totalLeak > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）: 共享文件句柄泄漏，未关闭句柄总数 = %d\n", res.totalLeak)
		t.Errorf("RED（红灯，缺陷未修复）: 共享句柄泄漏 totalLeak=%d (stress=%d solo=%d batch=%d)",
			res.totalLeak, res.leakedAfterStress, res.leakedAfterSolo, res.leakedAfterBatch)
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）: 无 panic，所有共享文件句柄已正确关闭")
}
