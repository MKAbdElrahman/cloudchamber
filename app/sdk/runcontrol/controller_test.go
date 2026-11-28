package runcontrol_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runcontrol"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runworker"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus/stores/rundb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/dbtest"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// hostPortEnv names a Temporal frontend to test against, such as the KIND
// cluster's localhost:7233. Unset, the tests start the SDK's dev server.
const hostPortEnv = "CLOUDCHAMBER_TEST_TEMPORAL_HOSTPORT"

const (
	namespace  = "default"
	threeSteps = "three-steps"
	quietSteps = "quiet"
	tick       = 100 * time.Millisecond
)

// stepsInput is the input of the three-steps workflow.
type stepsInput struct {
	Units int `json:"units"`
}

// seen is what the activities of one run observed.
type seen struct {
	a, b, c     atomic.Int64
	progress    atomic.Int64
	resumedFrom atomic.Int64
	sawPause    atomic.Bool
	failC       atomic.Bool
}

var observations sync.Map

func seenBy(runID uuid.UUID) *seen {
	v, _ := observations.LoadOrStore(runbus.WorkflowID(runID), &seen{})
	return v.(*seen)
}

func seenIn(ctx context.Context) *seen {
	v, _ := observations.LoadOrStore(activity.GetInfo(ctx).WorkflowExecution.ID, &seen{})
	return v.(*seen)
}

// threeStepsWorkflow runs a, then b, then c. Step b works in ticks and heartbeats its place.
func threeStepsWorkflow(ctx workflow.Context, in runcontrol.Input) (map[string]string, error) {
	var input stepsInput
	if err := json.Unmarshal(in.Input, &input); err != nil {
		return nil, temporal.NewNonRetryableApplicationError("input", "BadInput", err)
	}

	opts := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		HeartbeatTimeout:    2 * time.Second,
		WaitForCancellation: true,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: tick, BackoffCoefficient: 1},
	}
	step := func(id string, fn any, args ...any) error {
		o := opts
		o.ActivityID = id
		return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, o), fn, args...).Get(ctx, nil)
	}

	if err := step("a", stepA); err != nil {
		return nil, err
	}
	if err := step("b", stepB, input.Units); err != nil {
		return nil, err
	}
	if err := step("c", stepC); err != nil {
		return nil, err
	}
	return map[string]string{"run": in.RunID}, nil
}

func stepA(ctx context.Context) error {
	seenIn(ctx).a.Add(1)
	return nil
}

func stepB(ctx context.Context, units int) error {
	st := seenIn(ctx)
	st.b.Add(1)

	var start int
	if activity.HasHeartbeatDetails(ctx) {
		if err := activity.GetHeartbeatDetails(ctx, &start); err != nil {
			return err
		}
		st.resumedFrom.Store(int64(start))
	}

	for i := start; i < units; i++ {
		select {
		case <-ctx.Done():
			if errors.Is(context.Cause(ctx), activity.ErrActivityPaused) {
				st.sawPause.Store(true)
			}
			return ctx.Err()
		case <-time.After(tick):
		}
		activity.RecordHeartbeat(ctx, i+1)
		st.progress.Store(int64(i + 1))
	}
	return nil
}

func stepC(ctx context.Context) error {
	st := seenIn(ctx)
	st.c.Add(1)
	if st.failC.Load() {
		return temporal.NewNonRetryableApplicationError("step c refused", "Refused", errors.New("the underlying cause"))
	}
	return nil
}

// quiet runs one activity that works for a while and never heartbeats or
// reports anything. It stops only when its context is canceled.
func quiet(ctx workflow.Context, _ runcontrol.Input) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		HeartbeatTimeout:    2 * time.Second,
		WaitForCancellation: true,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: tick, BackoffCoefficient: 1},
	})
	return workflow.ExecuteActivity(ctx, stepQuiet).Get(ctx, nil)
}

func stepQuiet(ctx context.Context) error {
	st := seenIn(ctx)
	st.b.Add(1)
	select {
	case <-ctx.Done():
		if errors.Is(context.Cause(ctx), activity.ErrActivityPaused) {
			st.sawPause.Store(true)
		}
		return ctx.Err()
	case <-time.After(4 * time.Second):
		return nil
	}
}

var temporalClient client.Client

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	switch hostPort := os.Getenv(hostPortEnv); hostPort {
	case "":
		srv, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
			ClientOptions: &client.Options{Namespace: namespace},
			LogLevel:      "error",
			ExtraArgs:     []string{"--dynamic-config-value", "frontend.WorkflowPauseEnabled=true"},
		})
		if err != nil {
			fmt.Println("dev server:", err)
			return 1
		}
		defer func() { _ = srv.Stop() }()
		temporalClient = srv.Client()

	default:
		c, err := client.Dial(client.Options{HostPort: hostPort, Namespace: namespace})
		if err != nil {
			fmt.Println("dial:", err)
			return 1
		}
		defer c.Close()
		temporalClient = c
	}

	return m.Run()
}

// harness is one database, one task queue, one worker and one controller.
type harness struct {
	t     *testing.T
	db    *dbtest.Database
	eng   runcontrol.Engine
	bus   runbus.ExtBusiness
	ctrl  *runcontrol.Controller
	owner uuid.UUID
	proj  uuid.UUID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := t.Context()

	db := dbtest.New(t, t.Name())
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

	bus := db.BusDomain.Run

	queue := "runcontrol-test-" + uuid.NewString()
	w := worker.New(temporalClient, queue, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{runworker.NewStepInterceptor(db.Log, bus)},
	})
	w.RegisterWorkflowWithOptions(threeStepsWorkflow, workflow.RegisterOptions{Name: "ThreeSteps"})
	w.RegisterActivity(stepA)
	w.RegisterActivity(stepB)
	w.RegisterActivity(stepC)
	w.RegisterWorkflowWithOptions(quiet, workflow.RegisterOptions{Name: "Quiet"})
	w.RegisterActivity(stepQuiet)
	if err := w.Start(); err != nil {
		t.Fatalf("worker: %s", err)
	}
	t.Cleanup(w.Stop)

	engine := runcontrol.NewTemporal(temporalClient, namespace, map[string]runcontrol.Definition{
		threeSteps: {WorkflowType: "ThreeSteps", TaskQueue: queue},
		quietSteps: {WorkflowType: "Quiet", TaskQueue: queue},
	})
	ctrl := runcontrol.New(runcontrol.Config{Log: db.Log, Runs: bus, Engine: engine, Grace: 5 * time.Second})

	return &harness{t: t, db: db, eng: engine, bus: bus, ctrl: ctrl, owner: user.ID, proj: projectID}
}

func (h *harness) create(units int) runbus.Run {
	h.t.Helper()
	created, err := h.bus.Create(h.t.Context(), runbus.NewRun{
		ID:        uuid.New(),
		Workflow:  workflowname.MustParse(threeSteps),
		OwnerID:   h.owner,
		ProjectID: h.proj,
		Input:     json.RawMessage(fmt.Sprintf(`{"units": %d}`, units)),
	})
	if err != nil {
		h.t.Fatalf("create: %s", err)
	}
	// A run the engine still holds would outlive the test. Stop it on the way out.
	h.t.Cleanup(func() {
		_ = temporalClient.TerminateWorkflow(context.Background(), runbus.WorkflowID(created.ID), "", "test cleanup")
	})
	return created
}

func (h *harness) command(run runbus.Run, command runbus.Command) runbus.Run {
	h.t.Helper()
	current, err := h.bus.QueryByID(h.t.Context(), run.ID)
	if err != nil {
		h.t.Fatalf("query: %s", err)
	}
	changed, err := h.bus.Command(h.t.Context(), current, command, h.owner)
	if err != nil {
		h.t.Fatalf("command: %s", err)
	}
	return changed
}

// until reconciles the run, as the controller's resync would, until the condition holds.
func (h *harness) until(run runbus.Run, what string, cond func(runbus.Run) bool) runbus.Run {
	h.t.Helper()
	ctx := h.t.Context()
	deadline := time.Now().Add(45 * time.Second)
	var last runbus.Run
	for time.Now().Before(deadline) {
		if err := h.ctrl.Reconcile(ctx, run.ID); err != nil {
			h.t.Fatalf("reconcile: %s", err)
		}
		got, err := h.bus.QueryByID(ctx, run.ID)
		if err != nil {
			h.t.Fatalf("query: %s", err)
		}
		if cond(got) {
			return got
		}
		last = got
		time.Sleep(tick)
	}
	h.t.Fatalf("timed out waiting for %s: phase=%s reason=%q message=%q attempt=%d", what, last.Status.Phase, last.Status.Reason, last.Status.Message, last.Status.Attempt)
	return runbus.Run{}
}

// steps renders the run's steps as "key=state" in the order they started.
func steps(run runbus.Run) string {
	var parts []string
	for _, step := range run.Status.Steps {
		parts = append(parts, step.Key+"="+step.State.String())
	}
	return strings.Join(parts, " ")
}

func inPhase(want phase.Phase) func(runbus.Run) bool {
	return func(run runbus.Run) bool { return run.Status.Phase == want }
}

func TestRunSucceeds(t *testing.T) {
	h := newHarness(t)
	run := h.create(3)

	done := h.until(run, "succeeded", inPhase(phase.Succeeded))

	var result map[string]string
	if err := json.Unmarshal(done.Status.Result, &result); err != nil || result["run"] != run.ID.String() {
		t.Errorf("result = %s err=%v", done.Status.Result, err)
	}
	if done.Status.Attempt != 1 || done.Status.ExecutionID == "" || done.Status.DateClosed.IsZero() || !done.Settled() {
		t.Errorf("status = %+v", done.Status)
	}
	st := seenBy(run.ID)
	if st.a.Load() != 1 || st.b.Load() != 1 || st.c.Load() != 1 {
		t.Errorf("steps ran a=%d b=%d c=%d", st.a.Load(), st.b.Load(), st.c.Load())
	}
	if got := steps(done); got != "stepA=done stepB=done stepC=done" {
		t.Errorf("steps = %q", got)
	}
}

// Suspend stops the step that is running, and resume continues it from where it stopped.
func TestSuspendStopsNowAndResumeContinues(t *testing.T) {
	h := newHarness(t)
	run := h.create(40)
	st := seenBy(run.ID)

	h.until(run, "step b under way", func(runbus.Run) bool { return st.progress.Load() >= 5 })

	h.command(run, runbus.Suspend)
	paused := h.until(run, "paused", inPhase(phase.Paused))

	if got := steps(paused); got != "stepA=done stepB=paused" {
		t.Errorf("steps while paused = %q", got)
	}
	if !st.sawPause.Load() {
		t.Error("the running step was not told to pause")
	}
	stoppedAt := st.progress.Load()
	time.Sleep(10 * tick)
	if now := st.progress.Load(); now != stoppedAt {
		t.Errorf("the step kept working while paused: %d then %d", stoppedAt, now)
	}
	if stoppedAt >= 40 {
		t.Fatalf("the step finished before the pause landed")
	}

	h.command(run, runbus.Resume)
	done := h.until(run, "succeeded", inPhase(phase.Succeeded))
	if got := steps(done); got != "stepA=done stepB=done stepC=done" {
		t.Errorf("steps = %q", got)
	}

	if from := st.resumedFrom.Load(); from == 0 || from > stoppedAt {
		t.Errorf("resumed from %d, stopped at %d", from, stoppedAt)
	}
	if st.a.Load() != 1 || st.c.Load() != 1 {
		t.Errorf("steps ran a=%d c=%d", st.a.Load(), st.c.Load())
	}
}

func TestCancelAPausedRun(t *testing.T) {
	h := newHarness(t)
	run := h.create(200)
	st := seenBy(run.ID)

	h.until(run, "step b under way", func(runbus.Run) bool { return st.progress.Load() >= 3 })
	h.command(run, runbus.Suspend)
	h.until(run, "paused", inPhase(phase.Paused))

	h.command(run, runbus.Cancel)
	closed := h.until(run, "closed", func(r runbus.Run) bool { return r.Settled() })

	if closed.Status.Phase != phase.Canceled && closed.Status.Phase != phase.Terminated {
		t.Errorf("phase = %s", closed.Status.Phase)
	}
	if st.c.Load() != 0 {
		t.Error("step c ran after the cancel")
	}
}

func TestCancelBeforeStart(t *testing.T) {
	h := newHarness(t)
	run := h.create(3)

	h.command(run, runbus.Cancel)
	closed := h.until(run, "canceled", inPhase(phase.Canceled))

	if closed.Status.Reason != runbus.ReasonCanceledBeforeStart || closed.Status.ExecutionID != "" || !closed.Settled() {
		t.Errorf("status = %+v", closed.Status)
	}
	if seenBy(run.ID).a.Load() != 0 {
		t.Error("a step ran")
	}
}

// Retry continues from the step that failed. The steps before it do not run again.
func TestRetryContinuesFromTheFailedStep(t *testing.T) {
	h := newHarness(t)
	run := h.create(3)
	st := seenBy(run.ID)
	st.failC.Store(true)

	failed := h.until(run, "failed", inPhase(phase.Failed))
	if failed.Status.Message != "step c refused" {
		t.Errorf("message = %q", failed.Status.Message)
	}
	if got := steps(failed); got != "stepA=done stepB=done stepC=failed" || failed.Status.Steps[2].Message != "step c refused" {
		t.Errorf("steps = %q message %q", got, failed.Status.Steps[2].Message)
	}
	first := failed.Status.ExecutionID

	st.failC.Store(false)
	h.command(run, runbus.Retry)
	done := h.until(run, "succeeded", inPhase(phase.Succeeded))

	if done.Status.Attempt != 2 || done.Status.ExecutionID == first || done.Status.Message != "" {
		t.Errorf("status = %+v", done.Status)
	}
	if st.a.Load() != 1 || st.b.Load() != 1 || st.c.Load() != 2 {
		t.Errorf("steps ran a=%d b=%d c=%d, want 1 1 2", st.a.Load(), st.b.Load(), st.c.Load())
	}
	if got := steps(done); got != "stepA=done stepB=done stepC=done" {
		t.Errorf("steps = %q", got)
	}
}

// Restart runs every step again.
func TestRestartRunsEverythingAgain(t *testing.T) {
	h := newHarness(t)
	run := h.create(2)
	st := seenBy(run.ID)

	first := h.until(run, "succeeded", inPhase(phase.Succeeded))

	h.command(run, runbus.Restart)
	second := h.until(run, "second attempt succeeded", func(r runbus.Run) bool {
		return r.Status.Phase == phase.Succeeded && r.Status.Attempt == 2
	})

	if second.Status.ExecutionID == first.Status.ExecutionID {
		t.Error("the restart reused the first execution")
	}
	if st.a.Load() != 2 || st.b.Load() != 2 || st.c.Load() != 2 {
		t.Errorf("steps ran a=%d b=%d c=%d, want 2 2 2", st.a.Load(), st.b.Load(), st.c.Load())
	}
	if got := steps(second); got != "stepA=done stepB=done stepC=done" {
		t.Errorf("steps = %q", got)
	}
	if !second.Status.Steps[0].DateStarted.After(first.Status.Steps[0].DateStarted) {
		t.Error("the restart kept the first attempt's steps")
	}

	events, err := h.bus.QueryEvents(t.Context(), run.ID)
	if err != nil {
		t.Fatalf("events: %s", err)
	}
	var types []string
	for _, event := range events {
		types = append(types, event.Type+":"+event.Message)
	}
	t.Logf("events: %v", types)
}

// A step that never heartbeats is still stopped by a suspend, because the
// worker's interceptor heartbeats for it.
func TestSuspendStopsAStepThatNeverHeartbeats(t *testing.T) {
	h := newHarness(t)
	created, err := h.bus.Create(t.Context(), runbus.NewRun{
		ID:        uuid.New(),
		Workflow:  workflowname.MustParse(quietSteps),
		OwnerID:   h.owner,
		ProjectID: h.proj,
	})
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	t.Cleanup(func() {
		_ = temporalClient.TerminateWorkflow(context.Background(), runbus.WorkflowID(created.ID), "", "test cleanup")
	})
	st := seenBy(created.ID)

	h.until(created, "the step under way", func(r runbus.Run) bool { return steps(r) == "stepQuiet=working" })

	h.command(created, runbus.Suspend)
	paused := h.until(created, "paused", inPhase(phase.Paused))
	if !st.sawPause.Load() || steps(paused) != "stepQuiet=paused" {
		t.Errorf("saw pause = %v, steps = %q", st.sawPause.Load(), steps(paused))
	}

	h.command(created, runbus.Resume)
	done := h.until(created, "succeeded", inPhase(phase.Succeeded))
	if steps(done) != "stepQuiet=done" || st.b.Load() != 2 {
		t.Errorf("steps = %q, attempts = %d", steps(done), st.b.Load())
	}
}

// The controller as it runs in the API: it leads, hears changes from the
// database, and is never told to reconcile by hand. The resync is an hour away,
// so everything it does here it does because a change was announced.
func TestControllerFollowsAnnouncedChanges(t *testing.T) {
	h := newHarness(t)
	ctx := t.Context()

	ctrl := runcontrol.New(runcontrol.Config{Log: h.db.Log, Runs: h.bus, Engine: h.eng, Grace: 5 * time.Second, Resync: time.Hour})
	go rundb.Lead(ctx, h.db.Log, h.db.DB, ctrl.Run)
	go rundb.Watch(ctx, h.db.Log, h.db.DB,
		func(change rundb.Change) {
			if change.Spec {
				ctrl.Nudge(change.RunID)
				return
			}
			ctrl.NudgeBatched(change.RunID)
		},
		func() { ctrl.Resync(ctx) },
	)
	// Let the watch start listening before the run is created.
	time.Sleep(500 * time.Millisecond)

	wait := func(run runbus.Run, what string, limit time.Duration, cond func(runbus.Run) bool) (runbus.Run, time.Duration) {
		t.Helper()
		start := time.Now()
		var last runbus.Run
		for time.Since(start) < limit {
			got, err := h.bus.QueryByID(ctx, run.ID)
			if err != nil {
				t.Fatalf("query: %s", err)
			}
			if cond(got) {
				return got, time.Since(start)
			}
			last = got
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for %s: phase=%s steps=%q", what, last.Status.Phase, steps(last))
		return runbus.Run{}, 0
	}

	run := h.create(30)
	st := seenBy(run.ID)

	_, took := wait(run, "running", 5*time.Second, inPhase(phase.Running))
	t.Logf("created to running: %s", took.Round(10*time.Millisecond))

	wait(run, "step b under way", 10*time.Second, func(runbus.Run) bool { return st.progress.Load() >= 5 })
	h.command(run, runbus.Suspend)
	_, took = wait(run, "paused", 10*time.Second, inPhase(phase.Paused))
	t.Logf("suspend to paused: %s", took.Round(10*time.Millisecond))

	h.command(run, runbus.Resume)
	wait(run, "running again", 5*time.Second, inPhase(phase.Running))

	// The run finishes on its own. Nobody asks for anything, so only the
	// worker's last step report tells the controller to look.
	wait(run, "step c done", 30*time.Second, func(r runbus.Run) bool { return steps(r) == "stepA=done stepB=done stepC=done" })
	done, took := wait(run, "succeeded", 5*time.Second, inPhase(phase.Succeeded))
	t.Logf("last step done to succeeded: %s", took.Round(10*time.Millisecond))
	if !done.Settled() {
		t.Errorf("status = %+v", done.Status)
	}
}
