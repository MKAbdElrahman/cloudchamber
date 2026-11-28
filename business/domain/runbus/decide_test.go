package runbus

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
)

var (
	now   = time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	grace = time.Minute
)

// runAt builds a run whose first attempt is the given execution in the given phase.
func runAt(phs phase.Phase, edit func(*Run)) Run {
	run := Run{
		Generation: 3,
		Spec:       Spec{Attempt: 1, ResumeFrom: resumefrom.Start},
		Status:     Status{Phase: phs, Attempt: 1, ExecutionID: "exec-1", ObservedGeneration: 2},
	}
	if edit != nil {
		edit(&run)
	}
	return run
}

func open(edit func(*Execution)) Execution {
	exec := Execution{Found: true, ID: "exec-1", State: ExecutionOpen}
	if edit != nil {
		edit(&exec)
	}
	return exec
}

func closed(state int) Execution {
	return Execution{Found: true, ID: "exec-1", State: state, Failure: "boom", Result: json.RawMessage(`{"ok":true}`), DateClosed: now.Add(-time.Second)}
}

func TestDecide(t *testing.T) {
	fresh := Run{Generation: 1, Spec: Spec{Attempt: 1, ResumeFrom: resumefrom.Start}, Status: Status{Phase: phase.Pending}}

	tests := []struct {
		name       string
		run        Run
		exec       Execution
		action     int
		activities []string
		phase      phase.Phase
		reason     string
		attempt    int
	}{
		// A run that has never started.
		{name: "new run starts", run: fresh, action: ActionStart, phase: phase.Pending},
		{name: "new run that is suspended waits", run: with(fresh, func(r *Run) { r.Spec.Suspend = true }), action: ActionNone, phase: phase.Pending, reason: ReasonSuspended},
		{name: "new run that is canceled closes without starting", run: with(fresh, func(r *Run) { r.Spec.Cancel = true }), action: ActionNone, phase: phase.Canceled, reason: ReasonCanceledBeforeStart, attempt: 1},
		{name: "a start the status missed is adopted", run: fresh, exec: open(nil), action: ActionNone, phase: phase.Running, attempt: 1},

		// An open execution with nothing asked of it.
		{name: "running run is left alone", run: runAt(phase.Running, nil), exec: open(nil), action: ActionNone, phase: phase.Running, attempt: 1},

		// Suspend: the execution first, then the activities still going.
		{name: "suspend pauses the execution", run: runAt(phase.Running, suspended), exec: open(nil), action: ActionPause, phase: phase.Running, attempt: 1},
		{
			name: "suspend then pauses the running activity", run: runAt(phase.Running, suspended),
			exec:   open(func(e *Execution) { e.Paused = true; e.Activities = []Activity{{ID: "a", Running: true}} }),
			action: ActionPauseActivities, activities: []string{"a"}, phase: phase.Running, attempt: 1,
		},
		{
			name: "an activity that has not stopped yet keeps the run running", run: runAt(phase.Running, suspended),
			exec:   open(func(e *Execution) { e.Paused = true; e.Activities = []Activity{{ID: "a", Running: true, Paused: true}} }),
			action: ActionNone, phase: phase.Running, attempt: 1,
		},
		{
			name: "a stopped activity makes the run paused", run: runAt(phase.Running, suspended),
			exec:   open(func(e *Execution) { e.Paused = true; e.Activities = []Activity{{ID: "a", Paused: true}} }),
			action: ActionNone, phase: phase.Paused, attempt: 1,
		},
		{name: "paused between steps is paused", run: runAt(phase.Running, suspended), exec: open(func(e *Execution) { e.Paused = true }), action: ActionNone, phase: phase.Paused, attempt: 1},

		// Resume: the execution first, then its activities.
		{name: "resume unpauses the execution", run: runAt(phase.Paused, nil), exec: open(func(e *Execution) { e.Paused = true }), action: ActionUnpause, phase: phase.Paused, attempt: 1},
		{
			name: "resume then unpauses the activities", run: runAt(phase.Paused, nil),
			exec:   open(func(e *Execution) { e.Activities = []Activity{{ID: "a", Paused: true}, {ID: "b", Running: true}} }),
			action: ActionUnpauseActivities, activities: []string{"a"}, phase: phase.Running, attempt: 1,
		},

		// Cancel: ask, let a paused execution see it, then stop waiting.
		{name: "cancel asks the execution to stop", run: runAt(phase.Running, canceled), exec: open(nil), action: ActionCancel, phase: phase.Running, reason: ReasonCanceling, attempt: 1},
		{
			name: "cancel beats suspend", run: runAt(phase.Paused, func(r *Run) { r.Spec.Cancel = true; r.Spec.Suspend = true }),
			exec: open(func(e *Execution) { e.Paused = true }), action: ActionCancel, phase: phase.Paused, reason: ReasonCanceling, attempt: 1,
		},
		{
			name: "a paused execution is unpaused so it sees the cancel", run: runAt(phase.Paused, canceled),
			exec: open(func(e *Execution) { e.Paused = true; e.CancelRequested = true }), action: ActionUnpause, phase: phase.Paused, reason: ReasonCanceling, attempt: 1,
		},
		{
			name: "cancel waits within the grace period", run: runAt(phase.Running, func(r *Run) { r.Spec.Cancel = true; r.Status.CancelRequestedAt = now.Add(-grace / 2) }),
			exec: open(func(e *Execution) { e.CancelRequested = true }), action: ActionNone, phase: phase.Running, reason: ReasonCanceling, attempt: 1,
		},
		{
			name: "cancel terminates after the grace period", run: runAt(phase.Running, func(r *Run) { r.Spec.Cancel = true; r.Status.CancelRequestedAt = now.Add(-2 * grace) }),
			exec: open(func(e *Execution) { e.CancelRequested = true }), action: ActionTerminate, phase: phase.Running, reason: ReasonCanceling, attempt: 1,
		},

		// A closed execution is recorded as it ended.
		{name: "completed is succeeded", run: runAt(phase.Running, nil), exec: closed(ExecutionCompleted), action: ActionNone, phase: phase.Succeeded, attempt: 1},
		{name: "failed is failed", run: runAt(phase.Running, nil), exec: closed(ExecutionFailed), action: ActionNone, phase: phase.Failed, attempt: 1},
		{name: "timed out is failed with a reason", run: runAt(phase.Running, nil), exec: closed(ExecutionTimedOut), action: ActionNone, phase: phase.Failed, reason: ReasonTimedOut, attempt: 1},
		{name: "canceled is canceled", run: runAt(phase.Running, canceled), exec: closed(ExecutionCanceled), action: ActionNone, phase: phase.Canceled, attempt: 1},
		{name: "terminated is terminated", run: runAt(phase.Running, canceled), exec: closed(ExecutionTerminated), action: ActionNone, phase: phase.Terminated, attempt: 1},
		{name: "a closed run ignores a late suspend", run: runAt(phase.Failed, suspended), exec: closed(ExecutionFailed), action: ActionNone, phase: phase.Failed, attempt: 1},

		// A new attempt.
		{name: "retry resets a failed execution", run: runAt(phase.Failed, attempt(2, resumefrom.Failure)), exec: closed(ExecutionFailed), action: ActionReset, phase: phase.Failed, attempt: 1},
		{name: "retry resets a canceled execution", run: runAt(phase.Canceled, attempt(2, resumefrom.Failure)), exec: closed(ExecutionCanceled), action: ActionReset, phase: phase.Canceled, attempt: 1},
		{name: "restart starts again", run: runAt(phase.Failed, attempt(2, resumefrom.Start)), exec: closed(ExecutionFailed), action: ActionStart, phase: phase.Failed, attempt: 1},
		{name: "restart of a succeeded run starts again", run: runAt(phase.Succeeded, attempt(2, resumefrom.Start)), exec: closed(ExecutionCompleted), action: ActionStart, phase: phase.Succeeded, attempt: 1},
		{
			name: "the new execution is adopted as the new attempt", run: runAt(phase.Failed, attempt(2, resumefrom.Failure)),
			exec: Execution{Found: true, ID: "exec-2", State: ExecutionOpen}, action: ActionNone, phase: phase.Running, attempt: 2,
		},
		{
			name: "a new attempt canceled before it starts closes", run: runAt(phase.Failed, func(r *Run) { attempt(2, resumefrom.Start)(r); r.Spec.Cancel = true }),
			exec: closed(ExecutionFailed), action: ActionNone, phase: phase.Canceled, reason: ReasonCanceledBeforeStart, attempt: 2,
		},

		// The engine has forgotten the execution.
		{name: "a closed run whose history expired stays as it was", run: runAt(phase.Succeeded, nil), action: ActionNone, phase: phase.Succeeded, attempt: 1},
		{name: "retry without history starts again", run: runAt(phase.Failed, attempt(2, resumefrom.Failure)), action: ActionStart, phase: phase.Failed, attempt: 1},
		{name: "an open run the engine lost is failed", run: runAt(phase.Running, nil), action: ActionNone, phase: phase.Failed, reason: ReasonExecutionLost, attempt: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.run, tt.exec, now, grace)
			if got.Action.Kind != tt.action || !slices.Equal(got.Action.Activities, tt.activities) {
				t.Errorf("action = %+v, want kind %d activities %v", got.Action, tt.action, tt.activities)
			}
			if got.Status.Phase != tt.phase || got.Status.Reason != tt.reason || got.Status.Attempt != tt.attempt {
				t.Errorf("status = phase %s reason %q attempt %d, want %s %q %d",
					got.Status.Phase, got.Status.Reason, got.Status.Attempt, tt.phase, tt.reason, tt.attempt)
			}
			if got.Status.ObservedGeneration != tt.run.Generation {
				t.Errorf("observed generation = %d, want %d", got.Status.ObservedGeneration, tt.run.Generation)
			}
		})
	}
}

func TestDecideRecordsTheOutcome(t *testing.T) {
	done := Decide(runAt(phase.Running, nil), closed(ExecutionCompleted), now, grace).Status
	if string(done.Result) != `{"ok":true}` || !done.DateClosed.Equal(now.Add(-time.Second)) {
		t.Errorf("succeeded status = %+v", done)
	}

	failed := Decide(runAt(phase.Running, nil), closed(ExecutionFailed), now, grace).Status
	if failed.Message != "boom" || failed.Result != nil {
		t.Errorf("failed status = %+v", failed)
	}

	reopened := Decide(runAt(phase.Failed, func(r *Run) {
		attempt(2, resumefrom.Failure)(r)
		r.Status.Message = "boom"
		r.Status.DateClosed = now
		r.Status.CancelRequestedAt = now
	}), Execution{Found: true, ID: "exec-2", State: ExecutionOpen}, now, grace).Status
	if reopened.Message != "" || !reopened.DateClosed.IsZero() || !reopened.CancelRequestedAt.IsZero() || reopened.ExecutionID != "exec-2" {
		t.Errorf("reopened status = %+v", reopened)
	}
}

func TestDecideStampsTheCancelOnce(t *testing.T) {
	first := Decide(runAt(phase.Running, canceled), open(nil), now, grace)
	if !first.Status.CancelRequestedAt.Equal(now) {
		t.Fatalf("cancel time = %s", first.Status.CancelRequestedAt)
	}
	run := runAt(phase.Running, canceled)
	run.Status = first.Status
	later := Decide(run, open(func(e *Execution) { e.CancelRequested = true }), now.Add(grace/2), grace)
	if !later.Status.CancelRequestedAt.Equal(now) || later.Action.Kind != ActionNone {
		t.Fatalf("later = %+v", later)
	}
}

// Decide is safe to run again: with the same observation it asks for the same thing.
func TestDecideIsStable(t *testing.T) {
	run := runAt(phase.Running, suspended)
	exec := open(func(e *Execution) { e.Paused = true })
	first := Decide(run, exec, now, grace)
	run.Status = first.Status
	second := Decide(run, exec, now.Add(time.Hour), grace)
	if second.Action.Kind != ActionNone || second.Status.Phase != phase.Paused {
		t.Fatalf("second = %+v", second)
	}
}

func TestCommands(t *testing.T) {
	open := runAt(phase.Running, nil)
	failed := runAt(phase.Failed, nil)
	succeeded := runAt(phase.Succeeded, nil)
	canceling := runAt(phase.Running, canceled)
	restarting := runAt(phase.Failed, attempt(2, resumefrom.Start))

	tests := []struct {
		name    string
		command Command
		run     Run
		allowed bool
		check   func(Spec) bool
	}{
		{name: "suspend an open run", command: Suspend, run: open, allowed: true, check: func(s Spec) bool { return s.Suspend }},
		{name: "suspend a closed run", command: Suspend, run: failed},
		{name: "suspend a canceling run", command: Suspend, run: canceling},
		{name: "resume an open run", command: Resume, run: runAt(phase.Paused, suspended), allowed: true, check: func(s Spec) bool { return !s.Suspend }},
		{name: "resume a closed run", command: Resume, run: succeeded},
		{name: "cancel an open run", command: Cancel, run: open, allowed: true, check: func(s Spec) bool { return s.Cancel }},
		{name: "cancel a closed run", command: Cancel, run: failed},
		{name: "cancel a run that is restarting", command: Cancel, run: restarting, allowed: true, check: func(s Spec) bool { return s.Cancel && s.Attempt == 2 }},
		{name: "retry a failed run", command: Retry, run: failed, allowed: true, check: func(s Spec) bool { return s.Attempt == 2 && s.ResumeFrom == resumefrom.Failure }},
		{name: "retry a succeeded run", command: Retry, run: succeeded},
		{name: "retry an open run", command: Retry, run: open},
		{name: "retry twice", command: Retry, run: restarting},
		{name: "restart a succeeded run", command: Restart, run: succeeded, allowed: true, check: func(s Spec) bool { return s.Attempt == 2 && s.ResumeFrom == resumefrom.Start }},
		{name: "restart an open run", command: Restart, run: open},
		{
			name: "a new attempt drops the old cancel", command: Restart, run: runAt(phase.Canceled, canceled), allowed: true,
			check: func(s Spec) bool { return !s.Cancel && !s.Suspend && s.Attempt == 2 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := tt.command.apply(tt.run)
			switch {
			case !tt.allowed:
				if !errors.Is(err, ErrNotAllowed) {
					t.Fatalf("err = %v, want not allowed", err)
				}
			case err != nil:
				t.Fatalf("err = %v", err)
			case !tt.check(spec):
				t.Fatalf("spec = %+v", spec)
			}
		})
	}
}

func with(run Run, edit func(*Run)) Run {
	edit(&run)
	return run
}

func suspended(r *Run) { r.Spec.Suspend = true }

func canceled(r *Run) { r.Spec.Cancel = true }

func attempt(n int, from resumefrom.ResumeFrom) func(*Run) {
	return func(r *Run) {
		r.Spec.Attempt = n
		r.Spec.ResumeFrom = from
	}
}

func TestConverged(t *testing.T) {
	tests := []struct {
		name string
		run  Run
		want bool
	}{
		{name: "running as asked", run: runAt(phase.Running, nil), want: true},
		{name: "pausing", run: runAt(phase.Running, suspended), want: false},
		{name: "paused as asked", run: runAt(phase.Paused, suspended), want: true},
		{name: "resuming", run: runAt(phase.Paused, nil), want: false},
		{name: "canceling", run: runAt(phase.Running, canceled), want: false},
		{name: "closed", run: runAt(phase.Failed, nil), want: true},
		{name: "restarting", run: runAt(phase.Failed, attempt(2, resumefrom.Start)), want: false},
		{name: "not started yet", run: Run{Spec: Spec{Attempt: 1}, Status: Status{Phase: phase.Pending}}, want: false},
		{name: "held before it starts", run: Run{Spec: Spec{Attempt: 1, Suspend: true}, Status: Status{Phase: phase.Pending}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.run.Converged(); got != tt.want {
				t.Errorf("converged = %v, want %v", got, tt.want)
			}
		})
	}
}
