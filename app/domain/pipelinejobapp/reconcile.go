package pipelinejobapp

import (
	"context"

	workflow "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

// activityInFlight reads the activity the open run is in. The name is run state
// and is not stored. A terminal job already holds its outcome from the worker.
func (a *app) activityInFlight(ctx context.Context, job pipelinejobbus.Job) (string, error) {
	if a.temporal == nil || job.WorkflowID == "" || job.Status.IsTerminal() {
		return "", nil
	}

	progress, err := a.queryProgress(ctx, job)
	if err != nil {
		return "", err
	}
	return progress.Activity, nil
}

func (a *app) queryProgress(ctx context.Context, job pipelinejobbus.Job) (workflow.ProcessSourceProgress, error) {
	var progress workflow.ProcessSourceProgress
	encoded, err := a.temporal.QueryWorkflow(ctx, job.WorkflowID, job.RunID, workflow.QueryProgress)
	if err != nil {
		return progress, err
	}
	if err := encoded.Get(&progress); err != nil {
		return workflow.ProcessSourceProgress{}, err
	}
	return progress, nil
}
