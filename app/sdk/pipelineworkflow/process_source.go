package workflow

import (
	"errors"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// Activity names registered by capability workers.
	ActivityResolveSource    = "pipeline.ResolveSource"
	ActivityExtractAudio     = "pipeline.ExtractAudio"
	ActivityTranscribe       = "pipeline.Transcribe"
	ActivityExtractKeyFrames = "pipeline.ExtractKeyFrames"
	ActivityComposeExport    = "pipeline.ComposeExport"
	ActivityMarkCanceled     = "pipeline.MarkCanceled"
)

// ProcessSourceWorkflow orchestrates the CloudChamber happy path. It is
// deterministic: no I/O, no wall clock or randomness outside Temporal APIs.
// Each activity writes the job outcome for the work it did.
func ProcessSourceWorkflow(ctx workflow.Context, in ProcessSourceInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("process source workflow started", "jobID", in.JobID, "fileID", in.FileID, "ownerID", in.OwnerID)

	progress := ProcessSourceProgress{}
	if err := workflow.SetQueryHandler(ctx, QueryProgress, func() (ProcessSourceProgress, error) {
		return progress, nil
	}); err != nil {
		return err
	}

	retry := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    5,
	}

	mediaOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueMedia,
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
		WaitForCancellation: true,
	}
	asrOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueASR,
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
		WaitForCancellation: true,
	}
	renderOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueRender,
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
		WaitForCancellation: true,
	}
	framesOpts := mediaOpts
	framesOpts.TaskQueue = TaskQueueFrames

	var resolved ResolveSourceOutput
	if err := runStep(ctx, in.JobID, &progress, mediaOpts, ActivityResolveSource, ResolveSourceInput(in), &resolved); err != nil {
		return err
	}

	var audio ExtractAudioOutput
	if err := runStep(ctx, in.JobID, &progress, mediaOpts, ActivityExtractAudio, ExtractAudioInput{
		JobID: in.JobID, OwnerID: resolved.OwnerID, ProjectID: resolved.ProjectID, Name: resolved.Name,
	}, &audio); err != nil {
		return err
	}

	var transcript TranscribeOutput
	if err := runStep(ctx, in.JobID, &progress, asrOpts, ActivityTranscribe, TranscribeInput{
		JobID: in.JobID, OwnerID: audio.OwnerID, ProjectID: resolved.ProjectID, FileID: audio.FileID, Name: audio.Name,
	}, &transcript); err != nil {
		return err
	}

	var frames ExtractKeyFramesOutput
	if err := runStep(ctx, in.JobID, &progress, framesOpts, ActivityExtractKeyFrames, ExtractKeyFramesInput{
		JobID:       in.JobID,
		OwnerID:     resolved.OwnerID,
		ProjectID:   resolved.ProjectID,
		Name:        resolved.Name,
		ContentType: resolved.ContentType,
	}, &frames); err != nil {
		return err
	}

	if err := runStep(ctx, in.JobID, &progress, renderOpts, ActivityComposeExport, ComposeExportInput{
		JobID:            in.JobID,
		OwnerID:          resolved.OwnerID,
		ProjectID:        resolved.ProjectID,
		FileID:           in.FileID,
		Name:             resolved.Name,
		TranscriptFileID: transcript.FileID,
		KeyFramesFileID:  frames.FileID,
	}, &ComposeExportOutput{}); err != nil {
		return err
	}

	logger.Info("process source workflow completed", "jobID", in.JobID)
	return nil
}

// runStep schedules one activity. A cancel that arrives before that schedule
// waits until MarkCanceled has stored the job. The activity that was already
// running stores its own outcome before it returns.
func runStep(ctx workflow.Context, jobID string, progress *ProcessSourceProgress, opts workflow.ActivityOptions, name string, input any, output any) error {
	step := progress.Activity
	err := callActivity(ctx, opts, progress, name, input, output)
	if err == nil || !isCanceled(err) || progress.Activity != step {
		return err
	}

	disconnected, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	if markErr := markCanceled(disconnected, jobID, step); markErr != nil {
		workflow.GetLogger(ctx).Info("mark canceled failed", "jobID", jobID, "err", markErr)
	}
	return err
}

func callActivity(ctx workflow.Context, opts workflow.ActivityOptions, progress *ProcessSourceProgress, name string, input any, output any) error {
	if errors.Is(ctx.Err(), workflow.ErrCanceled) {
		return ctx.Err()
	}
	progress.Activity = name
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, opts), name, input).Get(ctx, output)
}

func isCanceled(err error) bool {
	return temporal.IsCanceledError(err) || errors.Is(err, workflow.ErrCanceled)
}

func markCanceled(ctx workflow.Context, jobID, step string) error {
	opts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueMedia,
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
		},
	}
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, opts), ActivityMarkCanceled, MarkCanceledInput{
		JobID: jobID,
		Step:  step,
	}).Get(ctx, nil)
}
