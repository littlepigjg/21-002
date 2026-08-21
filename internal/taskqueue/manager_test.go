package taskqueue

import (
	"context"
	"testing"
	"time"
)

func TestBugSumm6_GoroutineLeakOnShutdown(t *testing.T) {
	handler := func(ctx context.Context, j Job) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	queue := NewQueue(64)
	mgr := NewManager(queue, 2, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	cancel()

	waitDone := make(chan struct{})
	go func() {
		mgr.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Logf("GREEN (defect fixed): workers exited after ctx cancel")
	case <-time.After(2 * time.Second):
		t.Logf("RED (defect not fixed): workers stuck after ctx cancel, Wait() did not return")
		t.Fail()
	}
}
