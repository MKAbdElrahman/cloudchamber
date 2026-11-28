package runbus

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
	"github.com/MKAbdElrahman/cloudchamber/business/types/stepstate"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
)

// Run is one long-running workflow a user asked for.
// Spec is what the user wants. Status is what the controller last observed.
type Run struct {
	ID                 uuid.UUID
	Workflow           workflowname.WorkflowName
	OwnerID            uuid.UUID
	ProjectID          uuid.UUID
	IdempotencyKeyHash string
	Generation         int64
	Spec               Spec
	Status             Status
	DateCreated        time.Time
	DateUpdated        time.Time
}

// Spec is the desired state. Only a user command changes it, and each change
// raises the run's generation by one.
type Spec struct {
	Input      json.RawMessage
	Suspend    bool
	Cancel     bool
	Attempt    int
	ResumeFrom resumefrom.ResumeFrom
}

// Status is the observed state. Only the controller writes it, except Steps,
// which the workers report.
type Status struct {
	Phase              phase.Phase
	ObservedGeneration int64
	Attempt            int
	ExecutionID        string
	Reason             string
	Message            string
	Steps              []Step
	Result             json.RawMessage
	CancelRequestedAt  time.Time
	DateClosed         time.Time
}

// Step is the last reported state of one step of a run.
// Fraction is how far a working step has got, from 0 to 1. It means nothing
// unless Measured is true: a step that cannot tell how far it is reports only a message.
type Step struct {
	Key         string
	State       stepstate.StepState
	Message     string
	Fraction    float64
	Measured    bool
	DateStarted time.Time
	DateUpdated time.Time
}

// StepReport is what a worker says about the step it is running.
type StepReport struct {
	Key      string
	State    stepstate.StepState
	Message  string
	Fraction float64
	Measured bool
}

// NewRun is what we require to create a run.
type NewRun struct {
	ID                 uuid.UUID
	Workflow           workflowname.WorkflowName
	OwnerID            uuid.UUID
	ProjectID          uuid.UUID
	IdempotencyKeyHash string
	Input              json.RawMessage
}

// Event records one command or one observed change. Nothing reads events to
// compute state. A nil ActorID is the platform.
type Event struct {
	ID          int64
	RunID       uuid.UUID
	ActorID     uuid.UUID
	Type        string
	Message     string
	DateCreated time.Time
}

// Set of event types.
const (
	EventCreated          = "Created"
	EventSuspendRequested = "SuspendRequested"
	EventResumeRequested  = "ResumeRequested"
	EventCancelRequested  = "CancelRequested"
	EventRetryRequested   = "RetryRequested"
	EventRestartRequested = "RestartRequested"
	EventPhaseChanged     = "PhaseChanged"
)

// Settled reports whether the controller has nothing left to do for this run:
// the attempt the user asked for has closed.
func (r Run) Settled() bool {
	return r.Status.Phase.Terminal() && r.Status.Attempt >= r.Spec.Attempt
}

// Converged reports whether what was observed matches what the spec asks for.
// A run that has not converged is in transition: starting, pausing, resuming or stopping.
func (r Run) Converged() bool {
	switch {
	case r.Settled():
		return true
	case r.Spec.Cancel:
		return false
	case r.Status.Attempt < r.Spec.Attempt:
		// An attempt that is suspended before it starts is held until it is resumed.
		return r.Spec.Suspend
	case r.Spec.Suspend:
		return r.Status.Phase == phase.Paused
	default:
		return r.Status.Phase == phase.Running
	}
}

// WorkflowID is the engine's id for a run. Every attempt of the run uses it.
func WorkflowID(runID uuid.UUID) string {
	return "run-" + runID.String()
}

// Same reports whether two statuses record the same observation.
// Steps are not compared. The workers own them.
func (s Status) Same(other Status) bool {
	return s.Phase == other.Phase &&
		s.ObservedGeneration == other.ObservedGeneration &&
		s.Attempt == other.Attempt &&
		s.ExecutionID == other.ExecutionID &&
		s.Reason == other.Reason &&
		s.Message == other.Message &&
		bytes.Equal(s.Result, other.Result) &&
		s.CancelRequestedAt.Equal(other.CancelRequestedAt) &&
		s.DateClosed.Equal(other.DateClosed)
}

// RunIDFromWorkflow reads the run id from an engine workflow id.
// It reports false for a workflow that is not a run.
func RunIDFromWorkflow(workflowID string) (uuid.UUID, bool) {
	raw, ok := strings.CutPrefix(workflowID, "run-")
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
