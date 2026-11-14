package workflow

import (
	"errors"
	"strings"
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
	ActivityRecordRun        = "pipeline.RecordRun"
)

// ProcessSourceWorkflow orchestrates the CloudChamber happy path. It is
// deterministic: no I/O, no wall clock or randomness outside Temporal APIs.
// Each activity writes its step. The workflow appends the ending fact on the way out.
func ProcessSourceWorkflow(ctx workflow.Context, in ProcessSourceInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("process source workflow started", "jobID", in.JobID, "fileID", in.FileID, "ownerID", in.OwnerID)

	err := processSource(ctx, in)
	disconnected, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	if err == nil {
		if recErr := recordRunFact(disconnected, in.JobID, "runCompleted", ""); recErr != nil {
			logger.Info("record run failed", "jobID", in.JobID, "err", recErr)
		}
		logger.Info("process source workflow completed", "jobID", in.JobID)
		return nil
	}

	if recErr := recordRun(disconnected, in.JobID, err); recErr != nil {
		logger.Info("record run failed", "jobID", in.JobID, "err", recErr)
	}
	return err
}

func processSource(ctx workflow.Context, in ProcessSourceInput) error {
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
	if err := runStep(ctx, mediaOpts, ActivityResolveSource, ResolveSourceInput(in), &resolved); err != nil {
		return err
	}

	var audio ExtractAudioOutput
	if err := runStep(ctx, mediaOpts, ActivityExtractAudio, ExtractAudioInput{
		JobID: in.JobID, OwnerID: resolved.OwnerID, ProjectID: resolved.ProjectID, Name: resolved.Name,
	}, &audio); err != nil {
		return err
	}

	var transcript TranscribeOutput
	if err := runStep(ctx, asrOpts, ActivityTranscribe, TranscribeInput{
		JobID: in.JobID, OwnerID: audio.OwnerID, ProjectID: resolved.ProjectID, FileID: audio.FileID, Name: audio.Name,
	}, &transcript); err != nil {
		return err
	}

	var frames ExtractKeyFramesOutput
	if err := runStep(ctx, framesOpts, ActivityExtractKeyFrames, ExtractKeyFramesInput{
		JobID:       in.JobID,
		OwnerID:     resolved.OwnerID,
		ProjectID:   resolved.ProjectID,
		Name:        resolved.Name,
		ContentType: resolved.ContentType,
	}, &frames); err != nil {
		return err
	}

	if err := runStep(ctx, renderOpts, ActivityComposeExport, ComposeExportInput{
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

	return nil
}

func recordRun(ctx workflow.Context, jobID string, err error) error {
	fact := "runFailed"
	message := clip(err.Error(), 500)
	if isCanceled(err) {
		fact = "runCanceled"
		message = ""
	}
	return recordRunFact(ctx, jobID, fact, message)
}

func recordRunFact(ctx workflow.Context, jobID, fact, message string) error {
	opts := workflow.ActivityOptions{
		TaskQueue:           TaskQueueWorkflow,
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, opts), ActivityRecordRun, RecordRunInput{
		JobID:   jobID,
		Fact:    fact,
		Message: message,
	}).Get(ctx, nil)
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// runStep schedules one activity. A cancel that is already on the context is not scheduled.
// The workflow records the run once, on the way out.
func runStep(ctx workflow.Context, opts workflow.ActivityOptions, name string, input any, output any) error {
	if errors.Is(ctx.Err(), workflow.ErrCanceled) {
		return ctx.Err()
	}
	return workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, opts), name, input).Get(ctx, output)
}

func isCanceled(err error) bool {
	return temporal.IsCanceledError(err) || errors.Is(err, workflow.ErrCanceled)
}
