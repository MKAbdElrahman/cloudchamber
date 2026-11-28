package runbus

import (
	"encoding/json"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
)

// Set of states the engine reports for an execution.
const (
	ExecutionOpen = iota
	ExecutionCompleted
	ExecutionFailed
	ExecutionTimedOut
	ExecutionCanceled
	ExecutionTerminated
)

// Execution is what the engine reports for a run's latest execution.
// Found is false when the engine has no execution for the run.
type Execution struct {
	Found           bool
	ID              string
	State           int
	Paused          bool
	CancelRequested bool
	Activities      []Activity
	Failure         string
	Result          json.RawMessage
	DateClosed      time.Time
}

// Activity is one activity the execution is waiting on.
// Running means an attempt is in flight on a worker. Paused means a pause was asked for.
type Activity struct {
	ID      string
	Running bool
	Paused  bool
}

// Set of actions the controller can take on the engine.
const (
	ActionNone = iota
	ActionStart
	ActionReset
	ActionPause
	ActionPauseActivities
	ActionUnpause
	ActionUnpauseActivities
	ActionCancel
	ActionTerminate
)

// Action is the one next step that moves the engine toward the spec.
// Activities lists the activity ids for the two activity actions.
type Action struct {
	Kind       int
	Activities []string
}

// Decision is the status to record and the action to take.
// The controller records the status, takes the action, and looks again.
type Decision struct {
	Status Status
	Action Action
}

// Set of reasons the controller records with a phase.
const (
	ReasonSuspended           = "Suspended"
	ReasonCanceling           = "Canceling"
	ReasonCanceledBeforeStart = "CanceledBeforeStart"
	ReasonExecutionLost       = "ExecutionLost"
	ReasonTimedOut            = "TimedOut"
)

// Decide compares what the user wants with what the engine reports. It is the
// whole control policy: it does no I/O and reads no clock but the one passed in.
// It returns one action at a time, so running it again after the action is safe.
// Steps pass through untouched. The workers own them.
func Decide(run Run, exec Execution, now time.Time, grace time.Duration) Decision {
	status := run.Status
	status.ObservedGeneration = run.Generation

	// An execution the status has not recorded is the attempt the controller
	// started just before it last stopped. It belongs to the attempt the spec asks for.
	if exec.Found && exec.ID != status.ExecutionID {
		status.ExecutionID = exec.ID
		status.Attempt = run.Spec.Attempt
		status.CancelRequestedAt = time.Time{}
	}

	switch {
	case !exec.Found:
		return decideMissing(run.Spec, status, now)
	case exec.State == ExecutionOpen:
		return decideOpen(run.Spec, status, exec, now, grace)
	default:
		return decideClosed(run.Spec, status, exec, now)
	}
}

// decideMissing handles a run the engine has no execution for.
func decideMissing(spec Spec, status Status, now time.Time) Decision {
	switch {
	case status.Attempt < spec.Attempt:
		return nextAttemptDecision(spec, status, Action{Kind: ActionStart}, now)
	case !status.Phase.Terminal():
		// The status says an attempt was open, and the engine no longer has it.
		status.Phase = phase.Failed
		status.Reason = ReasonExecutionLost
		status.Message = "the engine no longer has this execution"
		status.DateClosed = now
	}
	return Decision{Status: status}
}

// decideClosed handles an execution that has finished.
func decideClosed(spec Spec, status Status, exec Execution, now time.Time) Decision {
	status.Phase = phase.Failed
	status.Reason = ""
	status.Message = exec.Failure
	status.Result = nil
	switch exec.State {
	case ExecutionCompleted:
		status.Phase = phase.Succeeded
		status.Result = exec.Result
	case ExecutionTimedOut:
		status.Reason = ReasonTimedOut
	case ExecutionCanceled:
		status.Phase = phase.Canceled
	case ExecutionTerminated:
		status.Phase = phase.Terminated
	}
	status.DateClosed = exec.DateClosed
	if status.DateClosed.IsZero() {
		status.DateClosed = now
	}

	if status.Attempt >= spec.Attempt {
		return Decision{Status: status}
	}

	next := Action{Kind: ActionStart}
	if spec.ResumeFrom == resumefrom.Failure && exec.State != ExecutionCompleted {
		next = Action{Kind: ActionReset}
	}
	return nextAttemptDecision(spec, status, next, now)
}

// nextAttemptDecision handles an attempt the user asked for that has no execution yet.
// A cancel closes it before it starts. A suspend holds it. Otherwise it starts.
func nextAttemptDecision(spec Spec, status Status, start Action, now time.Time) Decision {
	switch {
	case spec.Cancel:
		status.Phase = phase.Canceled
		status.Reason = ReasonCanceledBeforeStart
		status.Message = ""
		status.Attempt = spec.Attempt
		status.DateClosed = now
		return Decision{Status: status}
	case spec.Suspend:
		status.Reason = ReasonSuspended
		return Decision{Status: status}
	default:
		return Decision{Status: status, Action: start}
	}
}

// decideOpen handles an execution that is still open.
func decideOpen(spec Spec, status Status, exec Execution, now time.Time, grace time.Duration) Decision {
	var running, paused, unpaused []string
	for _, act := range exec.Activities {
		if act.Running {
			running = append(running, act.ID)
		}
		if act.Paused {
			paused = append(paused, act.ID)
			continue
		}
		unpaused = append(unpaused, act.ID)
	}

	status.Phase = phase.Running
	if exec.Paused && len(running) == 0 {
		status.Phase = phase.Paused
	}
	status.Reason = ""
	status.Message = ""
	status.Result = nil
	status.DateClosed = time.Time{}

	switch {
	case spec.Cancel:
		status.Reason = ReasonCanceling
		if status.CancelRequestedAt.IsZero() {
			status.CancelRequestedAt = now
		}
		return Decision{Status: status, Action: cancelAction(status, exec, now, grace)}
	case spec.Suspend:
		return Decision{Status: status, Action: suspendAction(exec, unpaused)}
	default:
		return Decision{Status: status, Action: resumeAction(exec, paused)}
	}
}

// cancelAction asks the execution to stop, lets a paused one see the request,
// and terminates it when the grace period runs out.
func cancelAction(status Status, exec Execution, now time.Time, grace time.Duration) Action {
	switch {
	case !exec.CancelRequested:
		return Action{Kind: ActionCancel}
	case exec.Paused:
		// A paused execution records the cancel and acts on it only once unpaused.
		return Action{Kind: ActionUnpause}
	case now.Sub(status.CancelRequestedAt) > grace:
		return Action{Kind: ActionTerminate}
	default:
		return Action{}
	}
}

// suspendAction pauses the execution first, so no new step starts, and then
// the activities that are still going.
func suspendAction(exec Execution, unpaused []string) Action {
	switch {
	case !exec.Paused:
		return Action{Kind: ActionPause}
	case len(unpaused) > 0:
		return Action{Kind: ActionPauseActivities, Activities: unpaused}
	default:
		return Action{}
	}
}

// resumeAction undoes a suspend: the execution first, then its activities.
func resumeAction(exec Execution, paused []string) Action {
	switch {
	case exec.Paused:
		return Action{Kind: ActionUnpause}
	case len(paused) > 0:
		return Action{Kind: ActionUnpauseActivities, Activities: paused}
	default:
		return Action{}
	}
}
