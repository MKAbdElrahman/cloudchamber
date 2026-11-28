// Package media2docapp is the workflow that turns one recording into one document.
//
// It is a workflow like any other: the workflow function, its activities, a
// definition for the platform, and Register functions for the workers. It contains no
// bookkeeping. The run platform records the steps, and starts, pauses, resumes,
// cancels and retries the run.
package media2docapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runcontrol"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/framesbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/mediabus"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	// Name is the workflow a caller asks for.
	Name = "media2doc"

	// QueueWorkflow is the queue the workflow worker polls.
	QueueWorkflow = "cloudchamber-media2doc"
	// QueueMedia is the queue for reading the source and extracting audio.
	QueueMedia = "cloudchamber-media"
	// QueueASR is the queue for transcription.
	QueueASR = "cloudchamber-asr"
	// QueueFrames is the queue for key-frame extraction.
	QueueFrames = "cloudchamber-frames"
	// QueueRender is the queue for composing the document.
	QueueRender = "cloudchamber-render"

	workflowType = "media2doc.Convert"

	// The activity type names are the step keys a client sees.
	ActivityResolveSource    = "media2doc.ResolveSource"
	ActivityExtractAudio     = "media2doc.ExtractAudio"
	ActivityTranscribe       = "media2doc.Transcribe"
	ActivityExtractKeyFrames = "media2doc.ExtractKeyFrames"
	ActivityComposeDocument  = "media2doc.ComposeDocument"
)

// Input is what a caller sends as the run's input.
type Input struct {
	// FileID is the recording to convert. It must be a ready file in the run's project.
	FileID string `json:"fileId"`
}

// Result is what a finished run returns: the file each step produced.
// KeyFramesFileID is empty for a recording with no picture.
type Result struct {
	SourceFileID     string `json:"sourceFileId"`
	AudioFileID      string `json:"audioFileId"`
	TranscriptFileID string `json:"transcriptFileId"`
	KeyFramesFileID  string `json:"keyFramesFileId,omitempty"`
	DocumentFileID   string `json:"documentFileId"`
}

// ResolveInput names the recording a run was asked to convert.
type ResolveInput struct {
	RunID     string `json:"runId"`
	OwnerID   string `json:"ownerId"`
	ProjectID string `json:"projectId"`
	FileID    string `json:"fileId"`
}

// Source is the recording, once it is known to exist and to belong to the run's project.
// It carries ids and names, never bytes or storage locations.
type Source struct {
	RunID       string `json:"runId"`
	OwnerID     string `json:"ownerId"`
	ProjectID   string `json:"projectId"`
	FileID      string `json:"fileId"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
}

// File is one file a step produced.
type File struct {
	FileID string `json:"fileId"`
}

// TranscribeInput is the input of the transcription step.
type TranscribeInput struct {
	Source      Source `json:"source"`
	AudioFileID string `json:"audioFileId"`
}

// ComposeInput is the input of the composition step.
type ComposeInput struct {
	Source           Source `json:"source"`
	TranscriptFileID string `json:"transcriptFileId"`
	KeyFramesFileID  string `json:"keyFramesFileId"`
}

// Definition describes the workflow to the platform. media is used to check,
// before a run is stored, that the recording is one the caller may convert.
func Definition(media *mediabus.Business) workflows.Workflow {
	return workflows.Workflow{
		Name:         Name,
		Title:        "Recording to document",
		WorkflowType: workflowType,
		TaskQueue:    QueueWorkflow,
		Steps: []workflows.Step{
			{Key: ActivityResolveSource, Label: "Resolve source"},
			{Key: ActivityExtractAudio, Label: "Extract audio"},
			{Key: ActivityTranscribe, Label: "Transcribe"},
			{Key: ActivityExtractKeyFrames, Label: "Extract key frames"},
			{Key: ActivityComposeDocument, Label: "Compose export"},
		},
		InputSchema: json.RawMessage(`{
			"type": "object",
			"required": ["fileId"],
			"properties": {
				"fileId": {"type": "string", "format": "uuid", "title": "Recording"}
			}
		}`),
		Validate: func(ctx context.Context, req workflows.Request) error {
			return validate(ctx, media, req)
		},
	}
}

// validate refuses a recording the caller may not convert, before the run is
// stored. It asks the media worker's business package, which ResolveSource asks
// again when the run starts, because the file can change in between.
func validate(ctx context.Context, media *mediabus.Business, req workflows.Request) error {
	var input Input
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return &workflows.InputError{Field: "input", Message: "input must be an object with a fileId"}
	}
	fileID, err := uuid.Parse(strings.TrimSpace(input.FileID))
	if err != nil {
		return &workflows.InputError{Field: "fileId", Message: "fileId is required"}
	}

	_, err = media.CheckSource(ctx, req.OwnerID, req.ProjectID, fileID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, media2docbus.ErrSourceNotFound),
		errors.Is(err, media2docbus.ErrSourceNotInProject),
		errors.Is(err, media2docbus.ErrSourceNotReady):
		return &workflows.InputError{Field: "fileId", Message: "fileId must be a ready file in this project"}
	default:
		return fmt.Errorf("check file: %w", err)
	}
}

// RegisterWorkflow adds the workflow to a worker on QueueWorkflow.
func RegisterWorkflow(w worker.Worker) {
	w.RegisterWorkflowWithOptions(Convert, workflow.RegisterOptions{Name: workflowType})
}

// RegisterMedia adds the source and audio activities to a worker on QueueMedia.
func RegisterMedia(w worker.Worker, a *MediaActivities) {
	w.RegisterActivityWithOptions(a.ResolveSource, activity.RegisterOptions{Name: ActivityResolveSource})
	w.RegisterActivityWithOptions(a.ExtractAudio, activity.RegisterOptions{Name: ActivityExtractAudio})
}

// RegisterASR adds the transcription activity to a worker on QueueASR.
func RegisterASR(w worker.Worker, a *ASRActivities) {
	w.RegisterActivityWithOptions(a.Transcribe, activity.RegisterOptions{Name: ActivityTranscribe})
}

// RegisterFrames adds the key-frame activity to a worker on QueueFrames.
func RegisterFrames(w worker.Worker, a *FramesActivities) {
	w.RegisterActivityWithOptions(a.ExtractKeyFrames, activity.RegisterOptions{Name: ActivityExtractKeyFrames})
}

// RegisterRender adds the composition activity to a worker on QueueRender.
func RegisterRender(w worker.Worker, a *RenderActivities) {
	w.RegisterActivityWithOptions(a.ComposeDocument, activity.RegisterOptions{Name: ActivityComposeDocument})
}

// Convert is the workflow. It is deterministic: no I/O, and no clock or
// randomness outside the Temporal APIs.
func Convert(ctx workflow.Context, in runcontrol.Input) (Result, error) {
	var input Input
	if err := json.Unmarshal(in.Input, &input); err != nil || strings.TrimSpace(input.FileID) == "" {
		return Result{}, temporal.NewNonRetryableApplicationError("the input must name a recording as fileId", errTypeInvalidInput, err)
	}

	media := step(ctx, QueueMedia, 15*time.Minute)
	asr := step(ctx, QueueASR, 30*time.Minute)
	frames := step(ctx, QueueFrames, 15*time.Minute)
	render := step(ctx, QueueRender, 10*time.Minute)

	var source Source
	err := workflow.ExecuteActivity(media, ActivityResolveSource, ResolveInput{
		RunID:     in.RunID,
		OwnerID:   in.OwnerID,
		ProjectID: in.ProjectID,
		FileID:    strings.TrimSpace(input.FileID),
	}).Get(ctx, &source)
	if err != nil {
		return Result{}, err
	}

	var audio File
	if err := workflow.ExecuteActivity(media, ActivityExtractAudio, source).Get(ctx, &audio); err != nil {
		return Result{}, err
	}

	var transcript File
	err = workflow.ExecuteActivity(asr, ActivityTranscribe, TranscribeInput{Source: source, AudioFileID: audio.FileID}).Get(ctx, &transcript)
	if err != nil {
		return Result{}, err
	}

	// A recording with no picture has no frames. The step is not scheduled, so
	// the run's history shows that it was skipped on purpose.
	var keyFrames File
	if framesbus.HasPicture(source.ContentType) {
		if err := workflow.ExecuteActivity(frames, ActivityExtractKeyFrames, source).Get(ctx, &keyFrames); err != nil {
			return Result{}, err
		}
	}

	var document File
	err = workflow.ExecuteActivity(render, ActivityComposeDocument, ComposeInput{
		Source:           source,
		TranscriptFileID: transcript.FileID,
		KeyFramesFileID:  keyFrames.FileID,
	}).Get(ctx, &document)
	if err != nil {
		return Result{}, err
	}

	return Result{
		SourceFileID:     source.FileID,
		AudioFileID:      audio.FileID,
		TranscriptFileID: transcript.FileID,
		KeyFramesFileID:  keyFrames.FileID,
		DocumentFileID:   document.FileID,
	}, nil
}

// step returns the context for the activities of one queue.
func step(ctx workflow.Context, queue string, limit time.Duration) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           queue,
		StartToCloseTimeout: limit,
		// A suspend or a cancel reaches a running step on its next heartbeat, and the
		// SDK sends heartbeats at most every 0.8 of this timeout.
		HeartbeatTimeout:    5 * time.Second,
		WaitForCancellation: true,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	})
}
