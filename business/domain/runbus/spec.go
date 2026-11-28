package runbus

import (
	"errors"

	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
)

// ErrNotAllowed is returned when a command does not apply to the run's current state.
var ErrNotAllowed = errors.New("command is not allowed in the current state")

// Command is one user request. It edits the spec and does nothing else.
type Command struct {
	event string
	apply func(Run) (Spec, error)
}

// The commands a user can issue.
var (
	// Suspend asks an open run to stop now and hold its place.
	Suspend = Command{event: EventSuspendRequested, apply: suspend}
	// Resume asks a suspended run to continue.
	Resume = Command{event: EventResumeRequested, apply: resume}
	// Cancel asks an open run to stop for good. It cannot be withdrawn.
	Cancel = Command{event: EventCancelRequested, apply: cancel}
	// Retry asks a run that did not succeed to continue from where it failed.
	Retry = Command{event: EventRetryRequested, apply: retry}
	// Restart asks a closed run to run again from the beginning.
	Restart = Command{event: EventRestartRequested, apply: restart}
)

func suspend(run Run) (Spec, error) {
	if run.Settled() || run.Spec.Cancel {
		return Spec{}, ErrNotAllowed
	}
	spec := run.Spec
	spec.Suspend = true
	return spec, nil
}

func resume(run Run) (Spec, error) {
	if run.Settled() || run.Spec.Cancel {
		return Spec{}, ErrNotAllowed
	}
	spec := run.Spec
	spec.Suspend = false
	return spec, nil
}

func cancel(run Run) (Spec, error) {
	if run.Settled() {
		return Spec{}, ErrNotAllowed
	}
	spec := run.Spec
	spec.Cancel = true
	return spec, nil
}

func retry(run Run) (Spec, error) {
	if !run.Settled() || run.Status.Phase == phase.Succeeded {
		return Spec{}, ErrNotAllowed
	}
	return nextAttempt(run.Spec, resumefrom.Failure), nil
}

func restart(run Run) (Spec, error) {
	if !run.Settled() {
		return Spec{}, ErrNotAllowed
	}
	return nextAttempt(run.Spec, resumefrom.Start), nil
}

// nextAttempt asks for one more attempt. A cancel or a suspend belonged to the
// attempt that closed, so the new attempt starts without them.
func nextAttempt(spec Spec, from resumefrom.ResumeFrom) Spec {
	spec.Attempt++
	spec.ResumeFrom = from
	spec.Suspend = false
	spec.Cancel = false
	return spec
}

// sameIntent reports whether two specs ask for the same thing. Input never changes after create.
func sameIntent(a, b Spec) bool {
	return a.Suspend == b.Suspend && a.Cancel == b.Cancel && a.Attempt == b.Attempt && a.ResumeFrom == b.ResumeFrom
}
