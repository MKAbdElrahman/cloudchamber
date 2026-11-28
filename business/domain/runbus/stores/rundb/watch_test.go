package rundb_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus/stores/rundb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/dbtest"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/business/types/stepstate"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

// heard collects the changes a watch reported.
type heard struct {
	mu      sync.Mutex
	changes []rundb.Change
}

func (h *heard) add(change rundb.Change) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.changes = append(h.changes, change)
}

// take waits for n changes, returns them, and forgets them.
func (h *heard) take(t *testing.T, n int) []rundb.Change {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		if len(h.changes) >= n {
			got := h.changes
			h.changes = nil
			h.mu.Unlock()
			return got
		}
		h.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("heard %d changes, want %d", len(h.changes), n)
	return nil
}

// quiet fails when a change arrives that should not have been announced.
func (h *heard) quiet(t *testing.T, what string) {
	t.Helper()
	time.Sleep(300 * time.Millisecond)
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.changes) != 0 {
		t.Fatalf("%s was announced: %+v", what, h.changes)
	}
}

func Test_WatchHearsSpecAndStepChanges(t *testing.T) {
	db := dbtest.New(t, "Test_Watch")
	ctx := t.Context()

	user, err := db.BusDomain.User.Create(ctx, uuid.Nil, userbus.TestNewUsers(1, role.User)[0])
	if err != nil {
		t.Fatalf("seed user: %s", err)
	}
	projectID := uuid.New()
	if _, err := db.DB.ExecContext(ctx,
		`INSERT INTO projects (project_id, owner_id, name, date_created, date_updated) VALUES ($1, $2, 'Lectures', now(), now())`,
		projectID, user.ID); err != nil {
		t.Fatalf("seed project: %s", err)
	}

	var h heard
	ready := make(chan struct{}, 1)
	go rundb.Watch(ctx, db.Log, db.DB, h.add, func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not start")
	}

	bus := db.BusDomain.Run
	run, err := bus.Create(ctx, runbus.NewRun{ID: uuid.New(), Workflow: workflowname.MustParse("sample"), OwnerID: user.ID, ProjectID: projectID})
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	if got := h.take(t, 1); got[0] != (rundb.Change{RunID: run.ID, Spec: true}) {
		t.Fatalf("create announced %+v", got)
	}

	suspended, err := bus.Command(ctx, run, runbus.Suspend, user.ID)
	if err != nil {
		t.Fatalf("suspend: %s", err)
	}
	if got := h.take(t, 1); got[0] != (rundb.Change{RunID: run.ID, Spec: true}) {
		t.Fatalf("suspend announced %+v", got)
	}

	report := func(state stepstate.StepState, fraction float64) {
		t.Helper()
		if err := bus.ReportStep(ctx, run.ID, runbus.StepReport{Key: "work", State: state, Fraction: fraction, Measured: true}); err != nil {
			t.Fatalf("report: %s", err)
		}
	}

	report(stepstate.Working, 0.1)
	if got := h.take(t, 1); got[0] != (rundb.Change{RunID: run.ID, Spec: false}) {
		t.Fatalf("a step starting announced %+v", got)
	}

	report(stepstate.Working, 0.5)
	report(stepstate.Working, 0.9)
	h.quiet(t, "progress within a step")

	report(stepstate.Done, 1)
	if got := h.take(t, 1); got[0] != (rundb.Change{RunID: run.ID, Spec: false}) {
		t.Fatalf("a step finishing announced %+v", got)
	}

	status := suspended.Status
	status.Phase = phase.Running
	status.ObservedGeneration = suspended.Generation
	if _, err := bus.RecordStatus(ctx, suspended, status); err != nil {
		t.Fatalf("record status: %s", err)
	}
	h.quiet(t, "the controller's own status write")
}

func Test_OnlyOneControllerLeads(t *testing.T) {
	db := dbtest.New(t, "Test_Lead")

	var leading atomic.Int32
	var most atomic.Int32
	candidate := func(ctx context.Context, started chan<- int, id int) {
		rundb.Lead(ctx, db.Log, db.DB, func(ctx context.Context) {
			n := leading.Add(1)
			for {
				seen := most.Load()
				if n <= seen || most.CompareAndSwap(seen, n) {
					break
				}
			}
			started <- id
			<-ctx.Done()
			leading.Add(-1)
		})
	}

	started := make(chan int, 4)
	ctxs := make([]context.CancelFunc, 2)
	for id := range 2 {
		ctx, cancel := context.WithCancel(t.Context())
		ctxs[id] = cancel
		go candidate(ctx, started, id)
	}

	var first int
	select {
	case first = <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("no controller took the lead")
	}

	select {
	case second := <-started:
		t.Fatalf("controllers %d and %d both lead", first, second)
	case <-time.After(5 * time.Second):
	}

	// The leader stops. The other one takes over.
	ctxs[first]()
	select {
	case next := <-started:
		if next == first {
			t.Fatalf("controller %d led again after it stopped", first)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("no controller took over")
	}

	if most.Load() != 1 {
		t.Fatalf("%d controllers led at once", most.Load())
	}
	ctxs[1-first]()
}
