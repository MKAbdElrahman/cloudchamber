// Package sim provides simulated CloudChamber pipeline activities for development.
// Each step writes a real object and registers it as a file the owner can open.
package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/workflow"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
)

const (
	simStepDelay = 700 * time.Millisecond
	simSteps     = 3
)

// Activities implements simulated pipeline activities.
type Activities struct {
	artifacts artifactbus.ExtBusiness
	objects   *objectstore.Client
}

// New constructs simulated activities that store objects and register files.
func New(artifacts artifactbus.ExtBusiness, objects *objectstore.Client) *Activities {
	return &Activities{artifacts: artifacts, objects: objects}
}

func (a *Activities) ResolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	if err := simulate(ctx, "resolve-source", in.JobID); err != nil {
		return workflow.ResolveSourceOutput{}, err
	}

	fileID, err := uuid.Parse(in.FileID)
	if err != nil {
		return workflow.ResolveSourceOutput{}, fmt.Errorf("file id: %w", err)
	}
	art, err := a.artifacts.QueryByFileID(ctx, fileID)
	if err != nil {
		return workflow.ResolveSourceOutput{}, fmt.Errorf("load source: %w", err)
	}
	if _, err := a.objects.HeadObject(ctx, art.Bucket, art.ObjectKey); err != nil {
		return workflow.ResolveSourceOutput{}, fmt.Errorf("head source: %w", err)
	}

	name := art.Name
	if name == "" {
		name = "recording"
	}

	return workflow.ResolveSourceOutput{
		JobID:       in.JobID,
		OwnerID:     art.OwnerID.String(),
		FileID:      art.FileID.String(),
		Name:        name,
		ContentType: art.ContentType,
	}, nil
}

func (a *Activities) ExtractAudio(ctx context.Context, in workflow.ExtractAudioInput) (workflow.ExtractAudioOutput, error) {
	if err := simulate(ctx, "extract-audio", in.JobID); err != nil {
		return workflow.ExtractAudioOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, fmt.Errorf("owner id: %w", err)
	}
	name := stem(in.Name) + ".wav"
	fileID, err := a.publish(ctx, in.JobID, ownerID, artifactbus.KindAudio, name, "audio/wav", toneWAV())
	if err != nil {
		return workflow.ExtractAudioOutput{}, err
	}

	return workflow.ExtractAudioOutput{
		JobID:   in.JobID,
		OwnerID: in.OwnerID,
		FileID:  fileID.String(),
		Name:    in.Name,
	}, nil
}

func (a *Activities) Transcribe(ctx context.Context, in workflow.TranscribeInput) (workflow.TranscribeOutput, error) {
	if err := simulate(ctx, "transcribe", in.JobID); err != nil {
		return workflow.TranscribeOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.TranscribeOutput{}, fmt.Errorf("owner id: %w", err)
	}
	body, err := json.Marshal(transcript(in.Name))
	if err != nil {
		return workflow.TranscribeOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, artifactbus.KindTranscript, stem(in.Name)+".transcript.json", "application/json", body)
	if err != nil {
		return workflow.TranscribeOutput{}, err
	}

	return workflow.TranscribeOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func (a *Activities) ExtractKeyFrames(ctx context.Context, in workflow.ExtractKeyFramesInput) (workflow.ExtractKeyFramesOutput, error) {
	if err := simulate(ctx, "extract-keyframes", in.JobID); err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}
	if strings.HasPrefix(in.ContentType, "audio/") {
		return workflow.ExtractKeyFramesOutput{JobID: in.JobID}, nil
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, fmt.Errorf("owner id: %w", err)
	}
	body, err := framePNG()
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, artifactbus.KindFrames, stem(in.Name)+"-frame.png", "image/png", body)
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}

	return workflow.ExtractKeyFramesOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func (a *Activities) ComposeExport(ctx context.Context, in workflow.ComposeExportInput) (workflow.ComposeExportOutput, error) {
	if err := simulate(ctx, "compose-export", in.JobID); err != nil {
		return workflow.ComposeExportOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ComposeExportOutput{}, fmt.Errorf("owner id: %w", err)
	}
	body := []byte(documentMarkdown(in.Name, in.KeyFramesFileID != ""))
	fileID, err := a.publish(ctx, in.JobID, ownerID, artifactbus.KindDocument, stem(in.Name)+".md", "text/markdown", body)
	if err != nil {
		return workflow.ComposeExportOutput{}, err
	}

	return workflow.ComposeExportOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func (a *Activities) publish(ctx context.Context, jobID string, ownerID uuid.UUID, kind, name, contentType string, body []byte) (uuid.UUID, error) {
	fileID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(jobID+"/"+kind))
	key := fmt.Sprintf("outputs/%s/%s", ownerID, fileID)
	if err := a.objects.PutObject(ctx, "", key, contentType, body); err != nil {
		return uuid.Nil, err
	}
	if _, err := a.artifacts.Register(ctx, artifactbus.NewArtifact{
		FileID:      fileID,
		OwnerID:     ownerID,
		Bucket:      a.objects.Bucket(),
		ObjectKey:   key,
		Kind:        artifactbus.MustParseKind(kind),
		ContentType: contentType,
		Name:        name,
		Status:      artifactbus.MustParseStatus(artifactbus.StatusReady),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("register %s: %w", kind, err)
	}
	return fileID, nil
}

func stem(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "recording"
	}
	if dot := strings.LastIndex(trimmed, "."); dot > 0 {
		return trimmed[:dot]
	}
	return trimmed
}

func transcript(name string) map[string]any {
	return map[string]any{
		"simulated": true,
		"source":    name,
		"segments": []map[string]string{
			{"start": "0:00", "text": "The recording starts with the topic and what these notes will cover."},
			{"start": "4:12", "text": "The first claim, in the speaker's words, followed by the example that supports it."},
			{"start": "11:05", "text": "A second point. The term is defined before it is used again."},
			{"start": "18:40", "text": "The recording ends on what to review before the next session."},
		},
	}
}

func documentMarkdown(name string, withFrame bool) string {
	frame := ""
	if withFrame {
		frame = "\n\n> Frame at 4:18. A slide with three boxes: the recording, the transcript, and this document.\n"
	}
	title := stem(name)
	return fmt.Sprintf(`Simulated composition of %s. The bytes in this file are what the pipeline stored.

# Opening

**0:00** — The recording starts with the topic and what these notes will cover.

# Main points

**4:12** — The first claim, in the speaker's words, followed by the example that supports it.
%s
**11:05** — A second point. The term is defined before it is used again.

# Close

**18:40** — The recording ends on what to review before the next session.

# %s
`, name, frame, title)
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
