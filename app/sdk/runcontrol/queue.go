package runcontrol

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// queue is the controller's work queue. It follows the contract of the
// Kubernetes workqueue:
//
//   - A run added many times before it is taken is taken once.
//   - A run is never held by two workers at once. A run added while a worker
//     holds it is handed out again when that worker is done.
//   - A run can be added after a delay, and a run whose reconcile failed comes
//     back after a delay that doubles with each failure in a row.
type queue struct {
	mu   sync.Mutex
	cond *sync.Cond

	order      []uuid.UUID
	dirty      map[uuid.UUID]struct{}
	processing map[uuid.UUID]struct{}
	// due is when a delayed add for a run fires. A later request for the same
	// run is dropped: the earlier look covers it.
	due      map[uuid.UUID]time.Time
	failures map[uuid.UUID]int
}

const (
	// retryBase and retryMax bound the wait before a failed run is tried again.
	retryBase = 250 * time.Millisecond
	retryMax  = time.Minute
)

func newQueue() *queue {
	q := &queue{
		dirty:      make(map[uuid.UUID]struct{}),
		processing: make(map[uuid.UUID]struct{}),
		due:        make(map[uuid.UUID]time.Time),
		failures:   make(map[uuid.UUID]int),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Add marks a run as needing a look.
func (q *queue) Add(id uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.dirty[id]; ok {
		return
	}
	q.dirty[id] = struct{}{}
	if _, ok := q.processing[id]; ok {
		return
	}
	q.order = append(q.order, id)
	q.cond.Signal()
}

// AddAfter marks a run as needing a look once the delay has passed.
func (q *queue) AddAfter(id uuid.UUID, delay time.Duration) {
	if delay <= 0 {
		q.Add(id)
		return
	}

	at := time.Now().Add(delay)

	q.mu.Lock()
	if existing, ok := q.due[id]; ok && !existing.After(at) {
		q.mu.Unlock()
		return
	}
	q.due[id] = at
	q.mu.Unlock()

	time.AfterFunc(delay, func() {
		q.mu.Lock()
		fire := q.due[id].Equal(at)
		if fire {
			delete(q.due, id)
		}
		q.mu.Unlock()
		if fire {
			q.Add(id)
		}
	})
}

// Get blocks until a run needs a look and hands it to the caller, who must
// call Done. It reports false once ctx is done. The queue itself outlives ctx:
// runs added while no worker is waiting are kept for the next one.
func (q *queue) Get(ctx context.Context) (uuid.UUID, bool) {
	// Wake this waiter when ctx ends, so it can see that and return.
	stop := context.AfterFunc(ctx, func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.cond.Broadcast()
	})
	defer stop()

	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.order) == 0 && ctx.Err() == nil {
		q.cond.Wait()
	}
	if ctx.Err() != nil {
		return uuid.Nil, false
	}

	id := q.order[0]
	q.order = q.order[1:]
	q.processing[id] = struct{}{}
	delete(q.dirty, id)
	return id, true
}

// Done says a worker has finished with a run. A run that was added again
// meanwhile goes back on the queue.
func (q *queue) Done(id uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.processing, id)
	if _, ok := q.dirty[id]; ok {
		q.order = append(q.order, id)
		q.cond.Signal()
	}
}

// Retry adds a run back after a failure, waiting longer each time it fails in a row.
func (q *queue) Retry(id uuid.UUID) {
	q.mu.Lock()
	n := q.failures[id]
	q.failures[id] = n + 1
	q.mu.Unlock()

	delay := retryMax
	if n < 16 {
		delay = min(retryBase<<n, retryMax)
	}
	q.AddAfter(id, delay)
}

// Forget clears a run's failure count after a reconcile that worked.
func (q *queue) Forget(id uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.failures, id)
}
