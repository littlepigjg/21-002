package taskqueue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBugConcur002_WaitGroupDeadlock(t *testing.T) {
	q := NewQueue(10)
	var processed atomic.Int32
	handler := func(ctx context.Context, j Job) error {
		err := j.Run(ctx)
		if err == nil {
			processed.Add(1)
		}
		return err
	}
	mgr := NewManager(q, 2, handler)
	mgr.Start(context.Background())

	for i := 0; i < 3; i++ {
		job := Job{
			ID: fmt.Sprintf("wg-job-%d", i),
			Run: func(ctx context.Context) error {
				return nil
			},
		}
		if err := q.Enqueue(context.Background(), job); err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("enqueue failed: %v", err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	if processed.Load() != 3 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("expected 3 jobs processed, got %d", processed.Load())
	}

	done := make(chan bool, 1)
	go func() {
		mgr.Stop()
		done <- true
	}()

	select {
	case <-done:
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	case <-time.After(2 * time.Second):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("WaitGroup deadlock: Stop() blocked indefinitely")
	}
}

func TestBugConcur002_ConcurrentQueueStateRace(t *testing.T) {
	q := NewQueue(100)
	handler := func(ctx context.Context, j Job) error {
		return j.Run(ctx)
	}
	mgr := NewManager(q, 2, handler)
	mgr.Start(context.Background())

	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errCount atomic.Int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			job := Job{
				ID: fmt.Sprintf("race-job-%d", idx),
				Run: func(ctx context.Context) error {
					return nil
				},
			}
			err := q.Enqueue(context.Background(), job)
			if err == nil {
				successCount.Add(1)
			} else {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errCount.Load() > 0 && successCount.Load() == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("all enqueue attempts failed: %d errors", errCount.Load())
	}

	time.Sleep(300 * time.Millisecond)

	q.Close()

	time.Sleep(200 * time.Millisecond)

	var closeWg sync.WaitGroup
	for i := 0; i < 20; i++ {
		closeWg.Add(1)
		go func(idx int) {
			defer closeWg.Done()
			job := Job{
				ID: fmt.Sprintf("post-close-job-%d", idx),
				Run: func(ctx context.Context) error {
					return nil
				},
			}
			_ = q.Enqueue(context.Background(), job)
		}(i)
	}
	closeWg.Wait()

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
