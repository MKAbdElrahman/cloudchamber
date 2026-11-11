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

	// cancelWaitsChangeID versions the cancel path. DefaultVersion resolves the
	// activity future when the request arrives. Version 1 waits for the activity
	// to close. Version 2 also returns before scheduling the next activity when
	// the workflow context is already canceled.
	cancelWaitsChangeID = "cancel-waits"
)

// ProcessSourceWorkflow orchestrates the CloudChamber happy path. It is
// deterministic: no I/O, no wall clock/randomness outside Temporal APIs, and
// only file ids flow through workflow history.
func ProcessSourceWorkflow(ctx workflow.Context, in ProcessSourceInput) (*ProcessSourceResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("process source workflow started", "jobID", in.JobID, "fileID", in.FileID, "ownerID", in.OwnerID)

	progress := ProcessSourceProgress{}
	if err := workflow.SetQueryHandler(ctx, QueryProgress, func() (ProcessSourceProgress, error) {
		return progress, nil
	}); err != nil {
		return nil, err
	}

	cancelWaits := workflow.GetVersion(ctx, cancelWaitsChangeID, workflow.DefaultVersion, 2)
	waitForCancellation := false
	stopWhenCanceled := false
	if cancelWaits >= 1 {
		waitForCancellation = true
	}
	if cancelWaits >= 2 {
		stopWhenCanceled = true
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
		WaitForCancellation: waitForCancellation,
	}
	asrOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueASR,
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
		WaitForCancellation: waitForCancellation,
	}
	renderOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueRender,
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
		WaitForCancellation: waitForCancellation,
	}

	var resolved ResolveSourceOutput
	if err := callActivity(ctx, mediaOpts, &progress, stopWhenCanceled, ActivityResolveSource, ResolveSourceInput(in), &resolved); err != nil {
		return nil, err
	}

	var audio ExtractAudioOutput
	if err := callActivity(ctx, mediaOpts, &progress, stopWhenCanceled, ActivityExtractAudio, ExtractAudioInput{
		JobID: in.JobID, OwnerID: resolved.OwnerID, ProjectID: resolved.ProjectID, Name: resolved.Name,
	}, &audio); err != nil {
		return nil, err
	}
	progress.AudioFileID = audio.FileID

	var transcript TranscribeOutput
	if err := callActivity(ctx, asrOpts, &progress, stopWhenCanceled, ActivityTranscribe, TranscribeInput{
		JobID: in.JobID, OwnerID: audio.OwnerID, ProjectID: resolved.ProjectID, FileID: audio.FileID, Name: audio.Name,
	}, &transcript); err != nil {
		return nil, err
	}
	progress.TranscriptFileID = transcript.FileID

	framesOpts := mediaOpts
	if workflow.GetVersion(ctx, "extract-key-frames-queue", workflow.DefaultVersion, 1) == 1 {
		framesOpts.TaskQueue = TaskQueueFrames
	}

	var frames ExtractKeyFramesOutput
	if err := callActivity(ctx, framesOpts, &progress, stopWhenCanceled, ActivityExtractKeyFrames, ExtractKeyFramesInput{
		JobID:       in.JobID,
		OwnerID:     resolved.OwnerID,
		ProjectID:   resolved.ProjectID,
		Name:        resolved.Name,
		ContentType: resolved.ContentType,
	}, &frames); err != nil {
		return nil, err
	}
	progress.KeyFramesFileID = frames.FileID

	var doc ComposeExportOutput
	if err := callActivity(ctx, renderOpts, &progress, stopWhenCanceled, ActivityComposeExport, ComposeExportInput{
		JobID:            in.JobID,
		OwnerID:          resolved.OwnerID,
		ProjectID:        resolved.ProjectID,
		FileID:           in.FileID,
		Name:             resolved.Name,
		TranscriptFileID: transcript.FileID,
		KeyFramesFileID:  frames.FileID,
	}, &doc); err != nil {
		return nil, err
	}
	progress.DocumentFileID = doc.FileID

	result := &ProcessSourceResult{
		JobID:            in.JobID,
		FileID:           in.FileID,
		AudioFileID:      audio.FileID,
		TranscriptFileID: transcript.FileID,
		KeyFramesFileID:  frames.FileID,
		DocumentFileID:   doc.FileID,
	}

	logger.Info("process source workflow completed", "jobID", in.JobID, "documentFileID", result.DocumentFileID)
	return result, nil
}

func callActivity(ctx workflow.Context, opts workflow.ActivityOptions, progress *ProcessSourceProgress, stopWhenCanceled bool, name string, input any, output any) error {
	if stopWhenCanceled && errors.Is(ctx.Err(), workflow.ErrCanceled) {
		return ctx.Err()
	}
	progress.Activity = name
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, opts), name, input).Get(ctx, output)
}
