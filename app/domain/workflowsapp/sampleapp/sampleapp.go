// Package sampleapp is a small workflow with no product meaning. It
// exists to exercise the run platform end to end: three steps, one of which
// takes time, reports its place, and can be stopped and continued.
//
// It is also the shape a real workflow takes: the workflow function, its
// activities, a definition for the platform, and a Register function for the worker.
// Nothing in it knows how runs are stored, paused, or retried.
package sampleapp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runcontrol"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	// Name is the workflow a caller asks for.
	Name = "sample"
	// TaskQueue is the queue the workflow's worker polls.
	TaskQueue = "cloudchamber-sample"

	workflowType    = "sample.Run"
	activityPrepare = "sample.Prepare"
	activityWork    = "sample.Work"
	activityFinish  = "sample.Finish"
)

// Input is what a caller sends as the run's input.
type Input struct {
	// Units is how many ticks the work step takes. Zero means ten.
	Units int `json:"units"`
	// TickMillis is the length of one tick. Zero means 500.
	TickMillis int `json:"tickMillis"`
	// FailForSeconds makes the finish step refuse until that long after the
	// run first started, so a retry after the wait succeeds.
	FailForSeconds int `json:"failForSeconds"`
}

// Result is what a finished run returns.
type Result struct {
	Units int `json:"units"`
}

// maxUnits bounds how long one run of the workflow can be asked to take.
const maxUnits = 10_000

// Definition describes the workflow to the platform.
func Definition() workflows.Workflow {
	return workflows.Workflow{
		Name:         Name,
		Title:        "Sample",
		WorkflowType: workflowType,
		TaskQueue:    TaskQueue,
		Steps: []workflows.Step{
			{Key: activityPrepare, Label: "Prepare"},
			{Key: activityWork, Label: "Work"},
			{Key: activityFinish, Label: "Finish"},
		},
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"units": {"type": "integer", "minimum": 0, "maximum": 10000, "title": "Ticks of work", "default": 10},
				"tickMillis": {"type": "integer", "minimum": 0, "maximum": 60000, "title": "Length of one tick, in milliseconds", "default": 500},
				"failForSeconds": {"type": "integer", "minimum": 0, "maximum": 3600, "title": "Seconds the finish step refuses for", "default": 0}
			}
		}`),
		Validate: validate,
	}
}

// validate refuses an input the workflow could not run, before the run is stored.
func validate(_ context.Context, req workflows.Request) error {
	var input Input
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return &workflows.InputError{Field: "input", Message: "input must be an object with numeric fields"}
	}
	switch {
	case input.Units < 0 || input.Units > maxUnits:
		return &workflows.InputError{Field: "units", Message: "units must be between 0 and 10000"}
	case input.TickMillis < 0 || input.TickMillis > 60_000:
		return &workflows.InputError{Field: "tickMillis", Message: "tickMillis must be between 0 and 60000"}
	case input.FailForSeconds < 0 || input.FailForSeconds > 3600:
		return &workflows.InputError{Field: "failForSeconds", Message: "failForSeconds must be between 0 and 3600"}
	}
	return nil
}

// Register adds the workflow and its activities to a worker on TaskQueue.
func Register(w worker.Worker) {
	w.RegisterWorkflowWithOptions(Run, workflow.RegisterOptions{Name: workflowType})
	w.RegisterActivityWithOptions(prepare, activity.RegisterOptions{Name: activityPrepare})
	w.RegisterActivityWithOptions(work, activity.RegisterOptions{Name: activityWork})
	w.RegisterActivityWithOptions(finish, activity.RegisterOptions{Name: activityFinish})
}

// Run is the workflow. It is plain Temporal code.
func Run(ctx workflow.Context, in runcontrol.Input) (Result, error) {
	var input Input
	if err := json.Unmarshal(in.Input, &input); err != nil {
		return Result{}, temporal.NewNonRetryableApplicationError("the input is not valid", "BadInput", err)
	}
	if input.Units <= 0 {
		input.Units = 10
	}
	if input.TickMillis <= 0 {
		input.TickMillis = 500
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		// A suspend or a cancel reaches a running step on its next heartbeat, and the
		// SDK sends heartbeats at most every 0.8 of this timeout. A short timeout
		// is what makes "stop now" quick.
		HeartbeatTimeout:    3 * time.Second,
		WaitForCancellation: true,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	})

	// The first start time survives a retry from the failed step, because that keeps the earlier history.
	refuseUntil := workflow.Now(ctx).Add(time.Duration(input.FailForSeconds) * time.Second)

	if err := workflow.ExecuteActivity(ctx, activityPrepare).Get(ctx, nil); err != nil {
		return Result{}, err
	}
	if err := workflow.ExecuteActivity(ctx, activityWork, input).Get(ctx, nil); err != nil {
		return Result{}, err
	}
	if err := workflow.ExecuteActivity(ctx, activityFinish, refuseUntil).Get(ctx, nil); err != nil {
		return Result{}, err
	}

	return Result{Units: input.Units}, nil
}

func prepare(context.Context) error {
	return nil
}

// work counts ticks. Each tick is recorded as a heartbeat, so an attempt that
// was stopped continues from the tick it reached.
func work(ctx context.Context, input Input) error {
	var done int
	if activity.HasHeartbeatDetails(ctx) {
		if err := activity.GetHeartbeatDetails(ctx, &done); err != nil {
			return err
		}
	}

	tick := time.NewTicker(time.Duration(input.TickMillis) * time.Millisecond)
	defer tick.Stop()

	for done < input.Units {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
		done++
		activity.RecordHeartbeat(ctx, done)
	}

	return nil
}

func finish(_ context.Context, refuseUntil time.Time) error {
	if time.Now().Before(refuseUntil) {
		return temporal.NewNonRetryableApplicationError("the finish step is set to refuse for now", "Refused", errors.New("refused"))
	}
	return nil
}
