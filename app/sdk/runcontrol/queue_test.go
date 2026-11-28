package runcontrol

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
)

func TestQueueHandsARunOutOnce(t *testing.T) {
	q := newQueue()
	id := uuid.New()

	q.Add(id)
	q.Add(id)
	q.Add(id)

	got, ok := q.Get(t.Context())
	if !ok || got != id {
		t.Fatalf("get = %s %v", got, ok)
	}
	if queued(q) != 0 {
		t.Fatalf("the run is queued %d more times", queued(q))
	}
}

func TestQueueRequeuesARunAddedWhileItIsHeld(t *testing.T) {
	q := newQueue()
	id := uuid.New()

	q.Add(id)
	held, _ := q.Get(t.Context())

	q.Add(id)
	if queued(q) != 0 {
		t.Fatal("a held run was handed out again")
	}

	q.Done(held)
	again, ok := q.Get(t.Context())
	if !ok || again != id {
		t.Fatalf("get after done = %s %v", again, ok)
	}
	q.Done(again)
	if queued(q) != 0 {
		t.Fatal("the run is queued with nothing new to do")
	}
}

func TestQueueNeverGivesOneRunToTwoWorkers(t *testing.T) {
	q := newQueue()
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	var mu sync.Mutex
	held := make(map[uuid.UUID]bool)
	var clash bool
	var handled int

	ctx, stop := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for {
				id, ok := q.Get(ctx)
				if !ok {
					return
				}
				mu.Lock()
				if held[id] {
					clash = true
				}
				held[id] = true
				handled++
				mu.Unlock()

				time.Sleep(time.Millisecond)

				mu.Lock()
				held[id] = false
				mu.Unlock()
				q.Done(id)
			}
		})
	}

	for range 200 {
		for _, id := range ids {
			q.Add(id)
		}
		time.Sleep(100 * time.Microsecond)
	}
	time.Sleep(20 * time.Millisecond)
	stop()
	wg.Wait()

	if clash {
		t.Fatal("two workers held the same run")
	}
	// How many of the 600 adds were merged depends on timing, so it is not
	// checked here. TestQueueHandsARunOutOnce checks merging on its own.
	if handled == 0 {
		t.Fatal("no run was handled")
	}
}

func TestQueueDelaysAndBacksOff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		q := newQueue()
		id := uuid.New()

		// The earlier of two delayed adds wins, and the run is handed out once.
		q.AddAfter(id, 10*time.Second)
		q.AddAfter(id, 2*time.Second)
		q.AddAfter(id, 30*time.Second)

		time.Sleep(2*time.Second - time.Millisecond)
		synctest.Wait()
		if queued(q) != 0 {
			t.Fatal("the run was added early")
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if queued(q) != 1 {
			t.Fatalf("queued %d after the shorter delay", queued(q))
		}
		got, _ := q.Get(t.Context())
		q.Done(got)

		time.Sleep(time.Minute)
		synctest.Wait()
		if queued(q) != 0 {
			t.Fatal("a later delayed add fired after the earlier one")
		}

		// Each failure in a row doubles the wait, up to the limit.
		waits := []time.Duration{retryBase, 2 * retryBase, 4 * retryBase}
		for _, want := range waits {
			q.Retry(id)
			time.Sleep(want - time.Millisecond)
			synctest.Wait()
			if queued(q) != 0 {
				t.Fatalf("retry came back before %s", want)
			}
			time.Sleep(time.Millisecond)
			synctest.Wait()
			if queued(q) != 1 {
				t.Fatalf("retry did not come back after %s", want)
			}
			got, _ := q.Get(t.Context())
			q.Done(got)
		}

		q.Forget(id)
		q.Retry(id)
		time.Sleep(retryBase)
		synctest.Wait()
		if queued(q) != 1 {
			t.Fatal("forget did not reset the wait")
		}
	})
}

func TestQueueReleasesAWaitingWorkerAndKeepsLaterRuns(t *testing.T) {
	q := newQueue()
	ctx, stop := context.WithCancel(t.Context())

	done := make(chan bool)
	go func() {
		_, ok := q.Get(ctx)
		done <- ok
	}()

	stop()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("get reported a run after its context ended")
		}
	case <-time.After(time.Second):
		t.Fatal("get did not return after its context ended")
	}

	// A run added while nobody is working is there for the next worker.
	id := uuid.New()
	q.Add(id)
	got, ok := q.Get(t.Context())
	if !ok || got != id {
		t.Fatalf("get = %s %v", got, ok)
	}
}

// queued is how many runs are waiting to be handed out.
func queued(q *queue) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.order)
}
