// Package sim provides simulated CloudChamber pipeline activities for development.
package sim

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/workflow"
)

const (
	artifactScheme = "s3://cloudchamber-artifacts/jobs"
	simStepDelay   = 700 * time.Millisecond
	simSteps       = 3
)

// Activities implements simulated pipeline activities.
type Activities struct{}

// New constructs simulated pipeline activities.
func New() *Activities {
	return &Activities{}
}

func (a *Activities) ResolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	if err := simulate(ctx, "resolve-source", in.JobID); err != nil {
		return workflow.ResolveSourceOutput{}, err
	}

	return workflow.ResolveSourceOutput{
		JobID:         in.JobID,
		ResolvedRef:   fmt.Sprintf("%s/%s/source/resolved", artifactScheme, in.JobID),
		SourceType:    "video",
		DurationHints: "unknown",
	}, nil
}

func (a *Activities) ExtractAudio(ctx context.Context, in workflow.ExtractAudioInput) (workflow.ExtractAudioOutput, error) {
	if err := simulate(ctx, "extract-audio", in.JobID); err != nil {
		return workflow.ExtractAudioOutput{}, err
	}

	return workflow.ExtractAudioOutput{
		JobID:    in.JobID,
		AudioRef: fmt.Sprintf("%s/%s/media/audio.wav", artifactScheme, in.JobID),
	}, nil
}

func (a *Activities) Transcribe(ctx context.Context, in workflow.TranscribeInput) (workflow.TranscribeOutput, error) {
	if err := simulate(ctx, "transcribe", in.JobID); err != nil {
		return workflow.TranscribeOutput{}, err
	}

	return workflow.TranscribeOutput{
		JobID:         in.JobID,
		TranscriptRef: fmt.Sprintf("%s/%s/asr/transcript.json", artifactScheme, in.JobID),
	}, nil
}

func (a *Activities) ExtractKeyFrames(ctx context.Context, in workflow.ExtractKeyFramesInput) (workflow.ExtractKeyFramesOutput, error) {
	if err := simulate(ctx, "extract-keyframes", in.JobID); err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}

	return workflow.ExtractKeyFramesOutput{
		JobID:        in.JobID,
		KeyFramesRef: fmt.Sprintf("%s/%s/media/keyframes.json", artifactScheme, in.JobID),
	}, nil
}

func (a *Activities) ComposeExport(ctx context.Context, in workflow.ComposeExportInput) (workflow.ComposeExportOutput, error) {
	if err := simulate(ctx, "compose-export", in.JobID); err != nil {
		return workflow.ComposeExportOutput{}, err
	}

	return workflow.ComposeExportOutput{
		JobID:       in.JobID,
		DocumentRef: fmt.Sprintf("%s/%s/exports/document.md", artifactScheme, in.JobID),
	}, nil
}

func simulate(ctx context.Context, stage, jobID string) error {
	logger := activity.GetLogger(ctx)
	for i := 1; i <= simSteps; i++ {
		select {
		case <-ctx.Done():
			logger.Info("activity canceled", "stage", stage, "jobID", jobID, "step", i)
			return ctx.Err()
		case <-time.After(simStepDelay):
			activity.RecordHeartbeat(ctx, fmt.Sprintf("%s step %d/%d", stage, i, simSteps))
			logger.Info("activity heartbeat", "stage", stage, "jobID", jobID, "step", i, "steps", simSteps)
		}
	}
	return nil
}
