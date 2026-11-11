// Package sim provides simulated CloudChamber pipeline activities for development.
// Each step writes a real object and registers it as a file the owner can open.
package sim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	workflow "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	simStepDelay = 700 * time.Millisecond
	simSteps     = 3
)

// Activities implements simulated pipeline activities.
type Activities struct {
	artifacts artifactbus.ExtBusiness
	jobs      pipelinejobbus.ExtBusiness
	objects   *objectstore.Client
	tracer    trace.Tracer
}

// New constructs simulated activities that store objects, register files, and record those files on the job.
func New(artifacts artifactbus.ExtBusiness, jobs pipelinejobbus.ExtBusiness, objects *objectstore.Client, tracer trace.Tracer) *Activities {
	return &Activities{artifacts: artifacts, jobs: jobs, objects: objects, tracer: tracer}
}

// traced stores the worker tracer the way the HTTP middleware does, so
// business spans started with otel.AddSpan join the activity trace.
func (a *Activities) traced(ctx context.Context) context.Context {
	return otel.InjectTracing(ctx, a.tracer)
}

func (a *Activities) ResolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	out, err := a.resolveSource(ctx, in)
	return out, a.finish(ctx, in.JobID, workflow.ActivityResolveSource, err)
}

func (a *Activities) resolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	if err := simulate(ctx, "resolve-source", in.JobID); err != nil {
		return workflow.ResolveSourceOutput{}, err
	}

	fileID, err := uuid.Parse(in.FileID)
	if err != nil {
		return workflow.ResolveSourceOutput{}, invalidInput(ctx, "file id", err)
	}
	art, err := a.artifacts.QueryByFileID(a.traced(ctx), fileID)
	if err != nil {
		return workflow.ResolveSourceOutput{}, loadSourceFailure(ctx, err)
	}
	if _, err := a.objects.HeadObject(ctx, art.Bucket, art.ObjectKey); err != nil {
		return workflow.ResolveSourceOutput{}, headSourceFailure(ctx, err)
	}

	name := art.Name
	if name == "" {
		name = "recording"
	}

	return workflow.ResolveSourceOutput{
		JobID:       in.JobID,
		OwnerID:     art.OwnerID.String(),
		ProjectID:   art.ProjectID.String(),
		FileID:      art.FileID.String(),
		Name:        name,
		ContentType: art.ContentType,
	}, nil
}

func (a *Activities) ExtractAudio(ctx context.Context, in workflow.ExtractAudioInput) (workflow.ExtractAudioOutput, error) {
	out, err := a.extractAudio(ctx, in)
	return out, a.finish(ctx, in.JobID, workflow.ActivityExtractAudio, err)
}

func (a *Activities) extractAudio(ctx context.Context, in workflow.ExtractAudioInput) (workflow.ExtractAudioOutput, error) {
	if err := simulate(ctx, "extract-audio", in.JobID); err != nil {
		return workflow.ExtractAudioOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, invalidInput(ctx, "owner id", err)
	}
	name := stem(in.Name) + ".wav"
	projectID, err := parseProjectID(ctx, in.ProjectID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, projectID, artifactbus.KindAudio, name, "audio/wav", toneWAV())
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
	out, err := a.transcribe(ctx, in)
	return out, a.finish(ctx, in.JobID, workflow.ActivityTranscribe, err)
}

func (a *Activities) transcribe(ctx context.Context, in workflow.TranscribeInput) (workflow.TranscribeOutput, error) {
	if err := simulate(ctx, "transcribe", in.JobID); err != nil {
		return workflow.TranscribeOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.TranscribeOutput{}, invalidInput(ctx, "owner id", err)
	}
	body, err := json.Marshal(transcript(in.Name))
	if err != nil {
		return workflow.TranscribeOutput{}, err
	}
	projectID, err := parseProjectID(ctx, in.ProjectID)
	if err != nil {
		return workflow.TranscribeOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, projectID, artifactbus.KindTranscript, stem(in.Name)+".transcript.json", "application/json", body)
	if err != nil {
		return workflow.TranscribeOutput{}, err
	}

	return workflow.TranscribeOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func (a *Activities) ExtractKeyFrames(ctx context.Context, in workflow.ExtractKeyFramesInput) (workflow.ExtractKeyFramesOutput, error) {
	out, err := a.extractKeyFrames(ctx, in)
	return out, a.finish(ctx, in.JobID, workflow.ActivityExtractKeyFrames, err)
}

func (a *Activities) extractKeyFrames(ctx context.Context, in workflow.ExtractKeyFramesInput) (workflow.ExtractKeyFramesOutput, error) {
	if err := simulate(ctx, "extract-keyframes", in.JobID); err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}
	if strings.HasPrefix(in.ContentType, "audio/") {
		return workflow.ExtractKeyFramesOutput{JobID: in.JobID}, nil
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, invalidInput(ctx, "owner id", err)
	}
	body, err := framePNG()
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}
	projectID, err := parseProjectID(ctx, in.ProjectID)
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, projectID, artifactbus.KindFrames, stem(in.Name)+"-frame.png", "image/png", body)
	if err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
	}

	return workflow.ExtractKeyFramesOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func (a *Activities) ComposeExport(ctx context.Context, in workflow.ComposeExportInput) (workflow.ComposeExportOutput, error) {
	out, err := a.composeExport(ctx, in)
	if err != nil {
		return out, a.finish(ctx, in.JobID, workflow.ActivityComposeExport, err)
	}
	if err := a.persistOutcome(ctx, in.JobID, pipelinejobbus.StatusCompleted, "", ""); err != nil {
		return out, err
	}
	return out, nil
}

func (a *Activities) composeExport(ctx context.Context, in workflow.ComposeExportInput) (workflow.ComposeExportOutput, error) {
	if err := simulate(ctx, "compose-export", in.JobID); err != nil {
		return workflow.ComposeExportOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ComposeExportOutput{}, invalidInput(ctx, "owner id", err)
	}
	body := []byte(documentMarkdown(in.Name, in.KeyFramesFileID != ""))
	projectID, err := parseProjectID(ctx, in.ProjectID)
	if err != nil {
		return workflow.ComposeExportOutput{}, err
	}
	fileID, err := a.publish(ctx, in.JobID, ownerID, projectID, artifactbus.KindDocument, stem(in.Name)+".md", "text/markdown", body)
	if err != nil {
		return workflow.ComposeExportOutput{}, err
	}

	return workflow.ComposeExportOutput{JobID: in.JobID, FileID: fileID.String()}, nil
}

func parseProjectID(ctx context.Context, raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, invalidInput(ctx, "project id", err)
	}
	return id, nil
}

func (a *Activities) publish(ctx context.Context, jobID string, ownerID, projectID uuid.UUID, kind, name, contentType string, body []byte) (uuid.UUID, error) {
	if err := contextDone(ctx, "before-publish"); err != nil {
		return uuid.Nil, err
	}

	ctx = a.traced(ctx)
	fileID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(jobID+"/"+kind))
	key := fmt.Sprintf("outputs/%s/%s", ownerID, fileID)
	if err := a.objects.PutObject(ctx, "", key, contentType, body); err != nil {
		return uuid.Nil, err
	}
	if err := contextDone(ctx, "before-register"); err != nil {
		return uuid.Nil, err
	}
	if _, err := a.artifacts.Register(ctx, artifactbus.NewArtifact{
		FileID:      fileID,
		OwnerID:     ownerID,
		ProjectID:   projectID,
		Bucket:      a.objects.Bucket(),
		ObjectKey:   key,
		Kind:        artifactbus.MustParseKind(kind),
		ContentType: contentType,
		Name:        name,
		Status:      artifactbus.MustParseStatus(artifactbus.StatusReady),
	}); err != nil {
		return uuid.Nil, fmt.Errorf("register %s: %w", kind, err)
	}
	// The object is already stored. Record it on the job even if this activity is canceling,
	// then surface the cancellation so the workflow does not schedule the next step.
	if err := a.recordFile(context.WithoutCancel(ctx), jobID, kind, fileID); err != nil {
		return uuid.Nil, err
	}
	if err := ctx.Err(); err != nil {
		return uuid.Nil, err
	}
	return fileID, nil
}

// recordFile stores the file a step registered on the pipeline job.
func (a *Activities) recordFile(ctx context.Context, jobID, kind string, fileID uuid.UUID) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return invalidInput(ctx, "job id", err)
	}
	if a.jobs == nil {
		return fmt.Errorf("record %s: pipeline jobs not configured", kind)
	}

	job, err := a.jobs.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pipelinejobbus.ErrNotFound) {
			return invalidInput(ctx, "job id", err)
		}
		return fmt.Errorf("record %s: %w", kind, err)
	}

	uj, changed, err := fileUpdate(job, kind, fileID)
	if err != nil {
		return invalidInput(ctx, "kind", err)
	}
	if !changed {
		return nil
	}
	if _, err := a.jobs.Update(ctx, job, uj); err != nil {
		return fmt.Errorf("record %s: %w", kind, err)
	}
	return nil
}

// fileUpdate sets the job column for one registered file. An id already stored is left as it is.
func fileUpdate(job pipelinejobbus.Job, kind string, fileID uuid.UUID) (pipelinejobbus.UpdateJob, bool, error) {
	switch kind {
	case artifactbus.KindAudio:
		if job.AudioFileID == fileID {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{AudioFileID: new(fileID)}, true, nil
	case artifactbus.KindTranscript:
		if job.TranscriptFileID == fileID {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{TranscriptFileID: new(fileID)}, true, nil
	case artifactbus.KindFrames:
		if job.KeyFramesFileID == fileID {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{KeyFramesFileID: new(fileID)}, true, nil
	case artifactbus.KindDocument:
		if job.DocumentFileID == fileID {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{DocumentFileID: new(fileID)}, true, nil
	default:
		return pipelinejobbus.UpdateJob{}, false, fmt.Errorf("unknown kind %q", kind)
	}
}

// MarkCanceled stores a cancel that landed after one step returned and before the next was scheduled.
func (a *Activities) MarkCanceled(ctx context.Context, in workflow.MarkCanceledInput) error {
	return a.persistOutcome(ctx, in.JobID, pipelinejobbus.StatusCanceled, in.Step, "")
}

// finish stores a terminal outcome for the activity that just ran.
// The run-ending error is returned only after that write lands. An earlier retryable attempt leaves the job running.
func (a *Activities) finish(ctx context.Context, jobID, step string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case canceled(ctx, err):
		if recErr := a.persistOutcome(ctx, jobID, pipelinejobbus.StatusCanceled, step, ""); recErr != nil {
			return recErr
		}
		return err
	case recordFailure(ctx, err):
		if recErr := a.persistOutcome(ctx, jobID, pipelinejobbus.StatusFailed, step, failureMessage(err)); recErr != nil {
			return recErr
		}
		return err
	default:
		return err
	}
}

// persistOutcome writes the closed job. A retryable failure is tried again until the activity deadline.
// The caller returns the run-ending error only after this returns nil.
func (a *Activities) persistOutcome(ctx context.Context, jobID, status, step, message string) error {
	wait := time.Second
	var err error
	for {
		err = a.recordOutcome(ctx, jobID, status, step, message)
		if err == nil || !retryableOutcome(err) || !activity.IsActivity(ctx) {
			return err
		}
		deadline := activity.GetInfo(ctx).Deadline
		if deadline.IsZero() || time.Until(deadline) < wait+time.Second {
			return err
		}
		activity.RecordHeartbeat(ctx, "record outcome")
		time.Sleep(wait)
		if wait < 10*time.Second {
			wait *= 2
		}
	}
}

func retryableOutcome(err error) bool {
	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	return !ok || !appErr.NonRetryable()
}

func canceled(ctx context.Context, err error) bool {
	return errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) || temporal.IsCanceledError(err)
}

func recordFailure(ctx context.Context, err error) bool {
	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok && appErr.NonRetryable() {
		return true
	}
	if !activity.IsActivity(ctx) {
		return false
	}
	info := activity.GetInfo(ctx)
	if info.RetryPolicy == nil || info.RetryPolicy.MaximumAttempts <= 0 {
		return false
	}
	return info.Attempt >= info.RetryPolicy.MaximumAttempts
}

func failureMessage(err error) string {
	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok {
		if msg := strings.TrimSpace(appErr.Message()); msg != "" {
			return msg
		}
	}
	return strings.TrimSpace(err.Error())
}

func (a *Activities) recordOutcome(ctx context.Context, jobID, status, step, message string) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return invalidInput(ctx, "job id", err)
	}
	st, err := pipelinejobbus.ParseStatus(status)
	if err != nil {
		return invalidInput(ctx, "status", err)
	}
	if a.jobs == nil {
		return fmt.Errorf("record outcome: pipeline jobs not configured")
	}

	ctx = context.WithoutCancel(a.traced(ctx))
	job, err := a.jobs.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pipelinejobbus.ErrNotFound) {
			return invalidInput(ctx, "job id", err)
		}
		return fmt.Errorf("record outcome: %w", err)
	}

	uj, changed, err := outcomeUpdate(job, st, step, message)
	if err != nil {
		return invalidInput(ctx, "status", err)
	}
	if !changed {
		return nil
	}
	if _, err := a.jobs.Update(ctx, job, uj); err != nil {
		return fmt.Errorf("record outcome: %w", err)
	}
	return nil
}

// outcomeUpdate sets the closed status and the step that stopped. A job that already stores that outcome is left as it is.
func outcomeUpdate(job pipelinejobbus.Job, status pipelinejobbus.Status, step, message string) (pipelinejobbus.UpdateJob, bool, error) {
	step = strings.TrimSpace(step)
	message = strings.TrimSpace(message)
	switch status.String() {
	case pipelinejobbus.StatusCompleted:
		if job.Status.Equal(status) && job.ErrorMessage == nil && job.StoppedStep == "" {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{Status: new(status), ClearError: true, StoppedStep: new("")}, true, nil
	case pipelinejobbus.StatusCanceled:
		if job.Status.Equal(status) && job.ErrorMessage == nil && job.StoppedStep == step {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{Status: new(status), ClearError: true, StoppedStep: new(step)}, true, nil
	case pipelinejobbus.StatusFailed:
		if message == "" {
			message = status.String()
		}
		if job.Status.Equal(status) && job.ErrorMessage != nil && *job.ErrorMessage == message && job.StoppedStep == step {
			return pipelinejobbus.UpdateJob{}, false, nil
		}
		return pipelinejobbus.UpdateJob{Status: new(status), ErrorMessage: new(message), StoppedStep: new(step)}, true, nil
	default:
		return pipelinejobbus.UpdateJob{}, false, fmt.Errorf("outcome %s", status)
	}
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
