package store

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"summarizer/internal/model"
)

func TestBugConcur005_ResultMetaRace(t *testing.T) {
	if os.Getenv("BUGCONCUR005_WORKER") == "1" {
		runRaceWorker(t)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestBugConcur005_ResultMetaRace$", "-test.v", "-test.timeout=30s")
	cmd.Env = append(os.Environ(), "BUGCONCUR005_WORKER=1")
	out, err := cmd.CombinedOutput()

	output := string(out)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("defect detected (exit code != 0):\n%s", output)
	}

	if !containsGreen(output) {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("defect detected, GREEN not found in output:\n%s", output)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

func runRaceWorker(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()

	for i := 0; i < 50; i++ {
		err := st.SaveResult(ctx, &model.AnalysisResult{
			ArticleID:     "art-" + strconv.Itoa(i),
			Summary:       "summary-" + strconv.Itoa(i),
			SentenceCount: 1,
			CreatedAt:     time.Now(),
		})
		if err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	var panicCount atomic.Int32

	var wg sync.WaitGroup
	done := make(chan struct{})

	for g := 0; g < 40; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			i := 0
			for {
				select {
				case <-done:
					return
				default:
				}
				r := &model.AnalysisResult{
					ArticleID:     "art-" + strconv.Itoa(id*100000+i),
					Summary:       "concurrent-write-" + strconv.Itoa(i),
					SentenceCount: 1,
					CreatedAt:     time.Now(),
				}
				_ = st.SaveResult(ctx, r)
				i++
			}
		}(g)
	}

	for g := 0; g < 40; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			for {
				select {
				case <-done:
					return
				default:
				}
				_ = st.Snapshot(ctx)
				_, _, _ = st.ListResults(ctx, 0, 50)
			}
		}()
	}

	time.Sleep(4 * time.Second)
	close(done)
	wg.Wait()

	if panicCount.Load() > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("concurrent map panic detected, panic count: %d", panicCount.Load())
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

func containsGreen(s string) bool {
	for i := 0; i+len("GREEN") <= len(s); i++ {
		if s[i:i+len("GREEN")] == "GREEN" {
			return true
		}
	}
	return false
}
