// Package workflows describes the workflows a run can be started from.
//
// A workflow is declared once, in its own package, and read from here by
// everything that needs it: the controller to start it, the API to check a
// request before it stores a run, and the client to know what it can start and
// how to label its steps. There is one set of endpoints for every workflow.
// What differs between workflows is data in this description.
//
// A workflow is the definition. A run is one execution of it.
package workflows

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runcontrol"
	"github.com/google/uuid"
)

// Step names one step of a workflow. Key is the activity type name the
// workers report, and Label is what a person reads.
type Step struct {
	Key   string
	Label string
}

// Request is what a caller asks for when it starts a run of a workflow.
type Request struct {
	OwnerID   uuid.UUID
	ProjectID uuid.UUID
	Input     json.RawMessage
}

// InputError says a request's input is not acceptable. Field names the part
// of the input at fault, as a path such as "fileId".
type InputError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *InputError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Workflow is one workflow a run can be started from.
type Workflow struct {
	// Name is what a caller asks for.
	Name string
	// Title is the workflow's name for a person.
	Title string
	// WorkflowType is the workflow type name registered with the engine, and
	// TaskQueue is the queue the workflow's worker polls.
	WorkflowType string
	TaskQueue    string
	// Steps are the workflow's steps in the order they usually run. A step a
	// run does not need may be missing from that run.
	Steps []Step
	// InputSchema is a JSON Schema for the input. A client can build a form from it.
	InputSchema json.RawMessage
	// Validate checks a request before the run is stored. It returns an
	// *InputError for a request the caller should fix, and any other error for a
	// check that could not be made. A nil Validate accepts every request.
	Validate func(ctx context.Context, req Request) error
}

// Registry holds the workflows this service knows.
type Registry struct {
	workflows map[string]Workflow
}

// NewRegistry constructs a registry. It panics on a workflow with no name, no
// workflow type or no task queue, and on two workflows with one name: these are
// mistakes in the program, found at start.
func NewRegistry(workflows ...Workflow) *Registry {
	r := &Registry{workflows: make(map[string]Workflow, len(workflows))}
	for _, wf := range workflows {
		switch {
		case wf.Name == "" || wf.WorkflowType == "" || wf.TaskQueue == "":
			panic(fmt.Sprintf("workflows: workflow %q needs a name, a workflow type and a task queue", wf.Name))
		case r.Has(wf.Name):
			panic(fmt.Sprintf("workflows: workflow %q is registered twice", wf.Name))
		}
		r.workflows[wf.Name] = wf
	}
	return r
}

// Has reports whether a workflow is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.workflows[name]
	return ok
}

// Get returns a workflow by name.
func (r *Registry) Get(name string) (Workflow, bool) {
	wf, ok := r.workflows[name]
	return wf, ok
}

// List returns every workflow, ordered by name.
func (r *Registry) List() []Workflow {
	list := make([]Workflow, 0, len(r.workflows))
	for _, wf := range r.workflows {
		list = append(list, wf)
	}
	slices.SortFunc(list, func(a, b Workflow) int { return cmp.Compare(a.Name, b.Name) })
	return list
}

// Definitions returns what the controller needs to start each workflow.
func (r *Registry) Definitions() map[string]runcontrol.Definition {
	defs := make(map[string]runcontrol.Definition, len(r.workflows))
	for name, wf := range r.workflows {
		defs[name] = runcontrol.Definition{WorkflowType: wf.WorkflowType, TaskQueue: wf.TaskQueue}
	}
	return defs
}

// Validate checks a request against its workflow. A workflow that is not registered is an input error.
func (r *Registry) Validate(ctx context.Context, name string, req Request) error {
	wf, ok := r.workflows[name]
	if !ok {
		return &InputError{Field: "workflow", Message: fmt.Sprintf("workflow %q is not registered", name)}
	}
	if wf.Validate == nil {
		return nil
	}
	return wf.Validate(ctx, req)
}
