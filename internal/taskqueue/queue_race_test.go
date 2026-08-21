package taskqueue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBugConcur003_ClosedChannelPanic(t *testing.T) {
	panicCount := 0
	var mu sync.Mutex

	for round := 0; round < 20; round++ {
		q := NewQueue(1024)
		roundPanicked := runRaceTest(q)
		if roundPanicked {
			mu.Lock()
			panicCount++
			mu.Unlock()
		}
	}

	if panicCount > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）：%d/20 轮次出现 send on closed channel panic\n", panicCount)
		t.Fatalf("RED: %d rounds panicked", panicCount)
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）：20 轮次均无 panic")
}

func runRaceTest(q *Queue) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()

	var wg sync.WaitGroup
	ctx := context.Background()
	var enqueued int64

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				job := Job{
					ID:  fmt.Sprintf("job-%d-%d", id, j),
					Run: func(ctx context.Context) error { return nil },
				}
				err := q.Enqueue(ctx, job)
				if err == nil {
					atomic.AddInt64(&enqueued, 1)
				}
			}
		}(i)
	}

	time.Sleep(1 * time.Microsecond)

	q.Close()

	wg.Wait()
	return false
}
