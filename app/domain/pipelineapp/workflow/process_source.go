package workflow

import (
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
)

// ProcessSourceWorkflow orchestrates the CloudChamber happy path. It is
// deterministic: no I/O, no wall clock/randomness outside Temporal APIs, and
// only file ids flow through workflow history.
func ProcessSourceWorkflow(ctx workflow.Context, in ProcessSourceInput) (*ProcessSourceResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("process source workflow started", "jobID", in.JobID, "fileID", in.FileID, "ownerID", in.OwnerID)

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
	}
	asrOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueASR,
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
	}
	renderOpts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueRender,
		StartToCloseTimeout: 5 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy:         retry,
	}

	var resolved ResolveSourceOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, mediaOpts),
		ActivityResolveSource,
		ResolveSourceInput(in),
	).Get(ctx, &resolved); err != nil {
		return nil, err
	}

	var audio ExtractAudioOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, mediaOpts),
		ActivityExtractAudio,
		ExtractAudioInput{JobID: in.JobID, OwnerID: resolved.OwnerID, ProjectID: resolved.ProjectID, Name: resolved.Name},
	).Get(ctx, &audio); err != nil {
		return nil, err
	}

	var transcript TranscribeOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, asrOpts),
		ActivityTranscribe,
		TranscribeInput{JobID: in.JobID, OwnerID: audio.OwnerID, ProjectID: resolved.ProjectID, FileID: audio.FileID, Name: audio.Name},
	).Get(ctx, &transcript); err != nil {
		return nil, err
	}

	framesOpts := mediaOpts
	if workflow.GetVersion(ctx, "extract-key-frames-queue", workflow.DefaultVersion, 1) == 1 {
		framesOpts.TaskQueue = TaskQueueFrames
	}

	var frames ExtractKeyFramesOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, framesOpts),
		ActivityExtractKeyFrames,
		ExtractKeyFramesInput{
			JobID:       in.JobID,
			OwnerID:     resolved.OwnerID,
			ProjectID:   resolved.ProjectID,
			Name:        resolved.Name,
			ContentType: resolved.ContentType,
		},
	).Get(ctx, &frames); err != nil {
		return nil, err
	}

	var doc ComposeExportOutput
	if err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, renderOpts),
		ActivityComposeExport,
		ComposeExportInput{
			JobID:            in.JobID,
			OwnerID:          resolved.OwnerID,
			ProjectID:        resolved.ProjectID,
			FileID:           in.FileID,
			Name:             resolved.Name,
			TranscriptFileID: transcript.FileID,
			KeyFramesFileID:  frames.FileID,
		},
	).Get(ctx, &doc); err != nil {
		return nil, err
	}

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
