package runapp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

// Spec is what the caller asked for.
type Spec struct {
	Input      json.RawMessage `json:"input"`
	Suspend    bool            `json:"suspend"`
	Cancel     bool            `json:"cancel"`
	Attempt    int             `json:"attempt"`
	ResumeFrom string          `json:"resumeFrom"`
}

// Step is the last reported state of one step.
type Step struct {
	Key     string `json:"key"`
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
	// Fraction is how far a working step has got, from 0 to 1. It is only meaningful when Measured is true.
	Fraction    float64 `json:"fraction"`
	Measured    bool    `json:"measured"`
	DateStarted string  `json:"dateStarted"`
	DateUpdated string  `json:"dateUpdated"`
}

// Status is what the platform last observed.
type Status struct {
	Phase              string          `json:"phase"`
	ObservedGeneration int64           `json:"observedGeneration"`
	Attempt            int             `json:"attempt"`
	Reason             string          `json:"reason,omitempty"`
	Message            string          `json:"message,omitempty"`
	Steps              []Step          `json:"steps"`
	Result             json.RawMessage `json:"result,omitempty"`
	DateClosed         string          `json:"dateClosed,omitempty"`
}

// Run is one run returned to the client. Settled is true when the platform has
// nothing left to do for it: a run that is not settled is still starting,
// running, pausing, or stopping.
type Run struct {
	ID          string `json:"id"`
	Workflow    string `json:"workflow"`
	OwnerID     string `json:"ownerId"`
	ProjectID   string `json:"projectId"`
	Generation  int64  `json:"generation"`
	Settled     bool   `json:"settled"`
	Spec        Spec   `json:"spec"`
	Status      Status `json:"status"`
	DateCreated string `json:"dateCreated"`
	DateUpdated string `json:"dateUpdated"`
}

// Encode implements the encoder interface.
func (app Run) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}

func fromBusRunResponse(run runbus.Run) Run {
	steps := make([]Step, len(run.Status.Steps))
	for i, step := range run.Status.Steps {
		steps[i] = Step{
			Key:         step.Key,
			State:       step.State.String(),
			Message:     step.Message,
			Fraction:    step.Fraction,
			Measured:    step.Measured,
			DateStarted: step.DateStarted.Format(time.RFC3339),
			DateUpdated: step.DateUpdated.Format(time.RFC3339),
		}
	}

	var closed string
	if !run.Status.DateClosed.IsZero() {
		closed = run.Status.DateClosed.Format(time.RFC3339)
	}

	return Run{
		ID:         run.ID.String(),
		Workflow:   run.Workflow.String(),
		OwnerID:    run.OwnerID.String(),
		ProjectID:  run.ProjectID.String(),
		Generation: run.Generation,
		Settled:    run.Settled(),
		Spec: Spec{
			Input:      run.Spec.Input,
			Suspend:    run.Spec.Suspend,
			Cancel:     run.Spec.Cancel,
			Attempt:    run.Spec.Attempt,
			ResumeFrom: run.Spec.ResumeFrom.String(),
		},
		Status: Status{
			Phase:              run.Status.Phase.String(),
			ObservedGeneration: run.Status.ObservedGeneration,
			Attempt:            run.Status.Attempt,
			Reason:             run.Status.Reason,
			Message:            run.Status.Message,
			Steps:              steps,
			Result:             run.Status.Result,
			DateClosed:         closed,
		},
		DateCreated: run.DateCreated.Format(time.RFC3339),
		DateUpdated: run.DateUpdated.Format(time.RFC3339),
	}
}

func fromBusRunsResponse(runs []runbus.Run) []Run {
	app := make([]Run, len(runs))
	for i, run := range runs {
		app[i] = fromBusRunResponse(run)
	}
	return app
}

// =============================================================================

// NewRun is the data needed to start a run. The owner is the caller.
// Workflow names what to run. Input is that workflow's own input and is passed to it as sent.
type NewRun struct {
	Workflow       string          `json:"workflow"`
	ProjectID      string          `json:"projectId"`
	Input          json.RawMessage `json:"input"`
	IdempotencyKey string          `json:"idempotencyKey"`
}

// Decode implements the decoder interface.
func (app *NewRun) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

// toBusNewRun parses and validates the request. The handler sets the
// idempotency key hash and asks the workflow to check the input.
func toBusNewRun(ctx context.Context, app NewRun) (runbus.NewRun, error) {
	var fieldErrs errs.FieldErrors

	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		fieldErrs.Add("ownerId", err)
	}

	name, err := workflowname.Parse(strings.TrimSpace(app.Workflow))
	if err != nil {
		fieldErrs.Add("workflow", err)
	}

	projectID, err := uuid.Parse(strings.TrimSpace(app.ProjectID))
	if err != nil {
		fieldErrs.Add("projectId", err)
	}

	input := app.Input
	switch {
	case len(input) == 0:
		input = json.RawMessage("{}")
	case !json.Valid(input):
		fieldErrs.Add("input", fmt.Errorf("input must be JSON"))
	}

	if len(fieldErrs) > 0 {
		return runbus.NewRun{}, fmt.Errorf("validate: %w", fieldErrs.ToError())
	}

	return runbus.NewRun{
		ID:        uuid.New(),
		Workflow:  name,
		OwnerID:   ownerID,
		ProjectID: projectID,
		Input:     input,
	}, nil
}

// =============================================================================

// Event is one recorded command or observed change.
type Event struct {
	ID          int64  `json:"id"`
	ActorID     string `json:"actorId,omitempty"`
	Type        string `json:"type"`
	Message     string `json:"message,omitempty"`
	DateCreated string `json:"dateCreated"`
}

// Events is the history of one run, oldest first.
type Events struct {
	Items []Event `json:"items"`
}

// Encode implements the encoder interface.
func (app Events) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}

func fromBusEventsResponse(events []runbus.Event) Events {
	items := make([]Event, len(events))
	for i, event := range events {
		var actor string
		if event.ActorID != uuid.Nil {
			actor = event.ActorID.String()
		}
		items[i] = Event{
			ID:          event.ID,
			ActorID:     actor,
			Type:        event.Type,
			Message:     event.Message,
			DateCreated: event.DateCreated.Format(time.RFC3339),
		}
	}
	return Events{Items: items}
}

// =============================================================================

// WorkflowStep names one step of a workflow.
type WorkflowStep struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Workflow is one workflow a caller can start a run of.
type Workflow struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Steps       []WorkflowStep  `json:"steps"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Workflows is every workflow a caller can start a run of.
type Workflows struct {
	Items []Workflow `json:"items"`
}

// Encode implements the encoder interface.
func (app Workflows) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}

func fromWorkflowsResponse(list []workflows.Workflow) Workflows {
	items := make([]Workflow, len(list))
	for i, wf := range list {
		steps := make([]WorkflowStep, len(wf.Steps))
		for j, step := range wf.Steps {
			steps[j] = WorkflowStep{Key: step.Key, Label: step.Label}
		}
		items[i] = Workflow{Name: wf.Name, Title: wf.Title, Steps: steps, InputSchema: wf.InputSchema}
	}
	return Workflows{Items: items}
}
