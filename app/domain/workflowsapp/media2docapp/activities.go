package media2docapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runworker"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/asrbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/framesbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/mediabus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/renderbus"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

// The activities are the workers' handlers. Each one does what an HTTP handler
// does: it converts the request, calls the worker's business package, and
// converts the answer. What is particular to an activity is that it reports
// progress and says which failures the engine must not retry.

// simulated is how long a stand-in step works, so a person can follow it and stop it.
const simulated = 20 * time.Second

// tick is how often a stand-in step records its place.
const tick = 500 * time.Millisecond

// MediaActivities are the media worker's steps.
type MediaActivities struct {
	media *mediabus.Business
}

// NewMediaActivities constructs the media worker's activities.
func NewMediaActivities(media *mediabus.Business) *MediaActivities {
	return &MediaActivities{media: media}
}

// ResolveSource checks that the recording is one the run may convert and that
// its bytes are in storage.
func (a *MediaActivities) ResolveSource(ctx context.Context, in ResolveInput) (Source, error) {
	runworker.Progress(ctx, -1, "Loading the recording.")

	ownerID, projectID, fileID, err := toBusResolve(in)
	if err != nil {
		return Source{}, invalidInput(ctx, err)
	}

	src, err := a.media.ResolveSource(ctx, ownerID, projectID, fileID)
	if err != nil {
		return Source{}, sourceFailure(ctx, err)
	}

	return fromBusSource(in.RunID, src), nil
}

// ExtractAudio downloads the recording and writes its audio.
// An attempt that is stopped starts the download again when it continues.
func (a *MediaActivities) ExtractAudio(ctx context.Context, in Source) (File, error) {
	runID, src, err := toBusSource(in)
	if err != nil {
		return File{}, invalidInput(ctx, err)
	}

	fileID, err := a.media.ExtractAudio(ctx, runID, src, func(fraction float64, message string) {
		runworker.Progress(ctx, fraction, message)
	})
	if err != nil {
		switch {
		case errors.Is(err, mediabus.ErrFFmpegNotFound):
			return File{}, permanent(ctx, "FFmpegNotFound", "ffmpeg was not found on the worker", err)
		case errors.Is(err, mediabus.ErrNoAudio):
			return File{}, permanent(ctx, errTypeInvalidInput, "the recording has no audio", err)
		default:
			return File{}, sourceFailure(ctx, err)
		}
	}

	return File{FileID: fileID.String()}, nil
}

// ASRActivities are the ASR worker's steps.
type ASRActivities struct {
	asr *asrbus.Business
}

// NewASRActivities constructs the ASR worker's activities.
func NewASRActivities(asr *asrbus.Business) *ASRActivities {
	return &ASRActivities{asr: asr}
}

// Transcribe writes the transcript.
func (a *ASRActivities) Transcribe(ctx context.Context, in TranscribeInput) (File, error) {
	runID, src, err := toBusSource(in.Source)
	if err != nil {
		return File{}, invalidInput(ctx, err)
	}
	if err := simulate(ctx, "Writing the transcript."); err != nil {
		return File{}, err
	}

	fileID, err := a.asr.Transcribe(ctx, runID, src)
	if err != nil {
		return File{}, err
	}

	return File{FileID: fileID.String()}, nil
}

// FramesActivities are the frames worker's steps.
type FramesActivities struct {
	frames *framesbus.Business
}

// NewFramesActivities constructs the frames worker's activities.
func NewFramesActivities(frames *framesbus.Business) *FramesActivities {
	return &FramesActivities{frames: frames}
}

// ExtractKeyFrames writes the key frames.
func (a *FramesActivities) ExtractKeyFrames(ctx context.Context, in Source) (File, error) {
	runID, src, err := toBusSource(in)
	if err != nil {
		return File{}, invalidInput(ctx, err)
	}
	if err := simulate(ctx, "Choosing the key frames."); err != nil {
		return File{}, err
	}

	fileID, err := a.frames.ExtractKeyFrames(ctx, runID, src)
	if err != nil {
		return File{}, err
	}

	return File{FileID: fileID.String()}, nil
}

// RenderActivities are the render worker's steps.
type RenderActivities struct {
	render *renderbus.Business
}

// NewRenderActivities constructs the render worker's activities.
func NewRenderActivities(render *renderbus.Business) *RenderActivities {
	return &RenderActivities{render: render}
}

// ComposeDocument writes the document.
func (a *RenderActivities) ComposeDocument(ctx context.Context, in ComposeInput) (File, error) {
	runID, src, err := toBusSource(in.Source)
	if err != nil {
		return File{}, invalidInput(ctx, err)
	}
	if err := simulate(ctx, "Writing the document."); err != nil {
		return File{}, err
	}

	fileID, err := a.render.Compose(ctx, runID, src, in.KeyFramesFileID != "")
	if err != nil {
		return File{}, err
	}

	return File{FileID: fileID.String()}, nil
}

// =============================================================================

// toBusResolve parses the ids a run names its recording with.
func toBusResolve(in ResolveInput) (ownerID, projectID, fileID uuid.UUID, err error) {
	if ownerID, err = uuid.Parse(in.OwnerID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("ownerId is not an id: %w", err)
	}
	if projectID, err = uuid.Parse(in.ProjectID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("projectId is not an id: %w", err)
	}
	if fileID, err = uuid.Parse(in.FileID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, fmt.Errorf("fileId is not a file id: %w", err)
	}
	return ownerID, projectID, fileID, nil
}

// toBusSource parses the recording a workflow passes between its steps.
// A recording that reached a later step was ready when the run started.
func toBusSource(in Source) (uuid.UUID, media2docbus.Source, error) {
	runID, err := uuid.Parse(in.RunID)
	if err != nil {
		return uuid.Nil, media2docbus.Source{}, fmt.Errorf("runId is not an id: %w", err)
	}
	ownerID, projectID, fileID, err := toBusResolve(ResolveInput{OwnerID: in.OwnerID, ProjectID: in.ProjectID, FileID: in.FileID})
	if err != nil {
		return uuid.Nil, media2docbus.Source{}, err
	}
	return runID, media2docbus.Source{
		FileID:      fileID,
		OwnerID:     ownerID,
		ProjectID:   projectID,
		Name:        in.Name,
		ContentType: in.ContentType,
		Ready:       true,
	}, nil
}

// fromBusSource converts the recording into what the workflow passes between its steps.
func fromBusSource(runID string, src media2docbus.Source) Source {
	return Source{
		RunID:       runID,
		OwnerID:     src.OwnerID.String(),
		ProjectID:   src.ProjectID.String(),
		FileID:      src.FileID.String(),
		Name:        src.Name,
		ContentType: src.ContentType,
	}
}

// simulate works for the simulated time in ticks. Each tick is recorded as a
// heartbeat, so an attempt that was stopped continues from the tick it reached.
func simulate(ctx context.Context, message string) error {
	ticks := int(simulated / tick)

	var done int
	if activity.HasHeartbeatDetails(ctx) {
		if err := activity.GetHeartbeatDetails(ctx, &done); err != nil {
			return err
		}
	}
	runworker.Progress(ctx, float64(done)/float64(ticks), message)

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for done < ticks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		done++
		activity.RecordHeartbeat(ctx, done)
		runworker.Progress(ctx, float64(done)/float64(ticks), message)
	}

	return nil
}
