package runbus_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/dbtest"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/business/types/stepstate"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

func Test_Run(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_Run")
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

	bus := db.BusDomain.Run
	newRun := func(key string) runbus.NewRun {
		return runbus.NewRun{
			ID:                 uuid.New(),
			Workflow:           workflowname.MustParse("video-to-document"),
			OwnerID:            user.ID,
			ProjectID:          projectID,
			IdempotencyKeyHash: key,
			Input:              json.RawMessage(`{"fileId": "f-1"}`),
		}
	}

	t.Run("create stores a pending run that needs the controller", func(t *testing.T) {
		created, err := bus.Create(ctx, newRun(""))
		if err != nil {
			t.Fatalf("create: %s", err)
		}
		got, err := bus.QueryByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("query: %s", err)
		}
		if got.Status.Phase != phase.Pending || got.Generation != 1 || got.Spec.Attempt != 1 || got.Spec.ResumeFrom != resumefrom.Start {
			t.Fatalf("run = %+v", got)
		}
		var input map[string]string
		if err := json.Unmarshal(got.Spec.Input, &input); err != nil || input["fileId"] != "f-1" {
			t.Fatalf("input = %s err=%v", got.Spec.Input, err)
		}
		if got.Status.Result != nil || len(got.Status.Steps) != 0 || !got.Status.DateClosed.IsZero() {
			t.Fatalf("status = %+v", got.Status)
		}
		if !contains(t, ctx, bus, created.ID) {
			t.Fatal("a pending run is not unsettled")
		}
	})

	t.Run("an idempotency key is used once per owner", func(t *testing.T) {
		if _, err := bus.Create(ctx, newRun("key-1")); err != nil {
			t.Fatalf("create: %s", err)
		}
		if _, err := bus.Create(ctx, newRun("key-1")); !errors.Is(err, runbus.ErrIdempotencyKey) {
			t.Fatalf("err = %v, want idempotency key", err)
		}
		if _, err := bus.Create(ctx, newRun("")); err != nil {
			t.Fatalf("a run without a key: %s", err)
		}
	})

	t.Run("a run cannot be created in a project the owner does not have", func(t *testing.T) {
		nr := newRun("")
		nr.ProjectID = uuid.New()
		if _, err := bus.Create(ctx, nr); !errors.Is(err, runbus.ErrProject) {
			t.Fatalf("err = %v, want project", err)
		}
	})

	t.Run("a command raises the generation and a stale one conflicts", func(t *testing.T) {
		run, err := bus.Create(ctx, newRun(""))
		if err != nil {
			t.Fatalf("create: %s", err)
		}
		stale := run

		suspended, err := bus.Command(ctx, run, runbus.Suspend, user.ID)
		if err != nil {
			t.Fatalf("suspend: %s", err)
		}
		if suspended.Generation != 2 || !suspended.Spec.Suspend {
			t.Fatalf("suspended = %+v", suspended)
		}

		again, err := bus.Command(ctx, suspended, runbus.Suspend, user.ID)
		if err != nil || again.Generation != 2 {
			t.Fatalf("repeated suspend = generation %d err=%v", again.Generation, err)
		}

		if _, err := bus.Command(ctx, stale, runbus.Cancel, user.ID); !errors.Is(err, runbus.ErrConflict) {
			t.Fatalf("stale cancel err = %v, want conflict", err)
		}
		stored, err := bus.QueryByID(ctx, run.ID)
		if err != nil || stored.Spec.Cancel || !stored.Spec.Suspend || stored.Generation != 2 {
			t.Fatalf("stored = %+v err=%v", stored.Spec, err)
		}

		if _, err := bus.Command(ctx, stored, runbus.Retry, user.ID); !errors.Is(err, runbus.ErrNotAllowed) {
			t.Fatalf("retry of an open run err = %v, want not allowed", err)
		}
	})

	t.Run("the controller records status without touching spec or steps", func(t *testing.T) {
		run, err := bus.Create(ctx, newRun(""))
		if err != nil {
			t.Fatalf("create: %s", err)
		}
		if err := bus.ReportStep(ctx, run.ID, runbus.StepReport{Key: "extract-audio", State: stepstate.Working, Message: "Reading the recording."}); err != nil {
			t.Fatalf("report step: %s", err)
		}

		decision := runbus.Decide(run, runbus.Execution{Found: true, ID: "exec-1"}, time.Now().UTC(), time.Minute)
		running, err := bus.RecordStatus(ctx, run, decision.Status)
		if err != nil {
			t.Fatalf("record status: %s", err)
		}
		if running.Status.Phase != phase.Running {
			t.Fatalf("phase = %s", running.Status.Phase)
		}

		closedAt := time.Now().UTC().Truncate(time.Microsecond)
		done := runbus.Decide(running, runbus.Execution{
			Found: true, ID: "exec-1", State: runbus.ExecutionCompleted,
			Result: json.RawMessage(`{"documentId": "d-1"}`), DateClosed: closedAt,
		}, time.Now().UTC(), time.Minute)
		if _, err := bus.RecordStatus(ctx, running, done.Status); err != nil {
			t.Fatalf("record status: %s", err)
		}

		got, err := bus.QueryByID(ctx, run.ID)
		if err != nil {
			t.Fatalf("query: %s", err)
		}
		if got.Status.Phase != phase.Succeeded || got.Status.Attempt != 1 || got.Status.ExecutionID != "exec-1" || got.Status.ObservedGeneration != 1 {
			t.Fatalf("status = %+v", got.Status)
		}
		if !got.Status.DateClosed.Equal(closedAt) || string(got.Status.Result) == "" {
			t.Fatalf("closed = %s result = %s", got.Status.DateClosed, got.Status.Result)
		}
		if len(got.Status.Steps) != 1 || got.Status.Steps[0].Key != "extract-audio" {
			t.Fatalf("steps = %+v", got.Status.Steps)
		}
		if !got.Settled() || contains(t, ctx, bus, run.ID) {
			t.Fatal("a succeeded run is still unsettled")
		}

		restarted, err := bus.Command(ctx, got, runbus.Restart, user.ID)
		if err != nil {
			t.Fatalf("restart: %s", err)
		}
		if restarted.Spec.Attempt != 2 || !contains(t, ctx, bus, run.ID) {
			t.Fatalf("restarted = %+v", restarted.Spec)
		}

		events, err := bus.QueryEvents(ctx, run.ID)
		if err != nil {
			t.Fatalf("events: %s", err)
		}
		var types []string
		for _, event := range events {
			types = append(types, event.Type+":"+event.Message)
		}
		want := []string{"Created:", "PhaseChanged:running", "PhaseChanged:succeeded", "RestartRequested:"}
		if len(types) != len(want) {
			t.Fatalf("events = %v, want %v", types, want)
		}
		for i := range want {
			if types[i] != want[i] {
				t.Fatalf("events = %v, want %v", types, want)
			}
		}
		if events[0].ActorID != user.ID || events[1].ActorID != uuid.Nil {
			t.Fatalf("actors = %s, %s", events[0].ActorID, events[1].ActorID)
		}
	})

	t.Run("a step keeps its start time and its place", func(t *testing.T) {
		run, err := bus.Create(ctx, newRun(""))
		if err != nil {
			t.Fatalf("create: %s", err)
		}
		report := func(key string, state stepstate.StepState, message string) {
			t.Helper()
			if err := bus.ReportStep(ctx, run.ID, runbus.StepReport{Key: key, State: state, Message: message, Fraction: 0.4, Measured: state == stepstate.Working}); err != nil {
				t.Fatalf("report %s: %s", key, err)
			}
			time.Sleep(2 * time.Millisecond)
		}
		report("first", stepstate.Working, "one")
		report("second", stepstate.Working, "two")
		report("first", stepstate.Done, "")

		got, err := bus.QueryByID(ctx, run.ID)
		if err != nil {
			t.Fatalf("query: %s", err)
		}
		steps := got.Status.Steps
		if len(steps) != 2 || steps[0].Key != "first" || steps[1].Key != "second" {
			t.Fatalf("steps = %+v", steps)
		}
		if steps[0].State != stepstate.Done || steps[0].Message != "" || !steps[0].DateUpdated.After(steps[0].DateStarted) {
			t.Fatalf("first = %+v", steps[0])
		}
		if steps[0].Measured || !steps[1].Measured || steps[1].Fraction != 0.4 {
			t.Fatalf("fractions = %+v", steps)
		}

		if err := bus.ClearSteps(ctx, got); err != nil {
			t.Fatalf("clear: %s", err)
		}
		cleared, err := bus.QueryByID(ctx, run.ID)
		if err != nil || len(cleared.Status.Steps) != 0 {
			t.Fatalf("cleared = %+v err=%v", cleared.Status.Steps, err)
		}

		if err := bus.ReportStep(ctx, uuid.New(), runbus.StepReport{Key: "first", State: stepstate.Done}); !errors.Is(err, runbus.ErrNotFound) {
			t.Fatalf("report to a missing run err = %v", err)
		}
	})

	t.Run("the database refuses a phase it does not know", func(t *testing.T) {
		run, err := bus.Create(ctx, newRun(""))
		if err != nil {
			t.Fatalf("create: %s", err)
		}
		if _, err := db.DB.ExecContext(ctx, `UPDATE runs SET phase = 'done' WHERE run_id = $1`, run.ID); err == nil {
			t.Fatal("an unknown phase was stored")
		}
		if _, err := db.DB.ExecContext(ctx, `UPDATE runs SET observed_attempt = 5 WHERE run_id = $1`, run.ID); err == nil {
			t.Fatal("an observed attempt ahead of the spec was stored")
		}
	})
}

func contains(t *testing.T, ctx context.Context, bus runbus.Observer, runID uuid.UUID) bool {
	t.Helper()
	runs, err := bus.Unsettled(ctx, 1000)
	if err != nil {
		t.Fatalf("unsettled: %s", err)
	}
	for _, run := range runs {
		if run.ID == runID {
			return true
		}
	}
	return false
}
