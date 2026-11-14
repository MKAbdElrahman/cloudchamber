// Package sim provides simulated CloudChamber pipeline activities for development.
// Each step writes a real object and registers it as a file the owner can open.
package sim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/activities/audioextract"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	workflow "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"go.opentelemetry.io/otel/trace"
)

// Activities implements simulated pipeline activities.
type Activities struct {
	artifacts artifactbus.ExtBusiness
	jobs      pipelinejobbus.ExtBusiness
	objects   *objectstore.Client
	tracer    trace.Tracer
	ffmpeg    string
}

// New constructs simulated activities that store objects, register files, and record those files on the job.
func New(artifacts artifactbus.ExtBusiness, jobs pipelinejobbus.ExtBusiness, objects *objectstore.Client, tracer trace.Tracer) *Activities {
	return &Activities{artifacts: artifacts, jobs: jobs, objects: objects, tracer: tracer}
}

// WithFFmpeg sets the ffmpeg binary ExtractAudio uses. An empty path searches PATH.
func (a *Activities) WithFFmpeg(path string) *Activities {
	a.ffmpeg = path
	return a
}

// traced stores the worker tracer the way the HTTP middleware does, so
// business spans started with otel.AddSpan join the activity trace.
func (a *Activities) traced(ctx context.Context) context.Context {
	return otel.InjectTracing(ctx, a.tracer)
}

func (a *Activities) ResolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	out, err := a.resolveSource(ctx, in)
	return out, a.complete(ctx, in.JobID, "", err, sourceDone, sourceFailed, sourceCanceled)
}

func (a *Activities) resolveSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
	var out workflow.ResolveSourceOutput
	err := a.emit(ctx, in.JobID, "resolve source", "Loading the recording.", pipelinejobbus.SourceStage, func(float64) factSample {
		return factSample{Message: "Loading the recording."}
	}, func(ctx context.Context) error {
		resolved, err := a.lookupSource(ctx, in)
		if err != nil {
			return err
		}
		out = resolved
		return nil
	})
	if err != nil {
		return workflow.ResolveSourceOutput{}, err
	}
	return out, nil
}

func (a *Activities) lookupSource(ctx context.Context, in workflow.ResolveSourceInput) (workflow.ResolveSourceOutput, error) {
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
	return out, a.complete(ctx, in.JobID, out.FileID, err, audioDone, audioFailed, audioCanceled)
}

func (a *Activities) extractAudio(ctx context.Context, in workflow.ExtractAudioInput) (workflow.ExtractAudioOutput, error) {
	if err := contextDone(ctx, "extract-audio"); err != nil {
		return workflow.ExtractAudioOutput{}, err
	}

	ownerID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, invalidInput(ctx, "owner id", err)
	}
	jobID, err := uuid.Parse(in.JobID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, invalidInput(ctx, "job id", err)
	}
	if a.jobs == nil {
		return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: pipeline jobs not configured")
	}
	if a.objects == nil {
		return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: object store not configured")
	}

	job, err := a.jobs.QueryByID(a.traced(ctx), jobID)
	if err != nil {
		if errors.Is(err, pipelinejobbus.ErrNotFound) {
			return workflow.ExtractAudioOutput{}, invalidInput(ctx, "job id", err)
		}
		return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: %w", err)
	}
	if job.InputFileID == uuid.Nil {
		return workflow.ExtractAudioOutput{}, invalidInput(ctx, "input file id", errors.New("missing source file"))
	}

	art, err := a.artifacts.QueryByFileID(a.traced(ctx), job.InputFileID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, loadSourceFailure(ctx, err)
	}

	dir, err := os.MkdirTemp("", "cloudchamber-audio-")
	if err != nil {
		return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: %w", err)
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "source"+sourceExt(art.Name, art.ContentType))
	dst := filepath.Join(dir, "audio.wav")
	if err := a.pullAndExtract(ctx, in.JobID, art.Bucket, art.ObjectKey, src, dst); err != nil {
		switch {
		case errors.Is(err, audioextract.ErrNotFound):
			return workflow.ExtractAudioOutput{}, permanent(ctx, "FFmpegNotFound", "ffmpeg not found", err)
		case objectstore.IsMissing(err):
			return workflow.ExtractAudioOutput{}, readSourceFailure(ctx, err)
		default:
			return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: %w", err)
		}
	}

	body, err := os.ReadFile(dst)
	if err != nil {
		return workflow.ExtractAudioOutput{}, fmt.Errorf("extract audio: %w", err)
	}
	if len(body) == 0 {
		return workflow.ExtractAudioOutput{}, invalidInput(ctx, "audio", errors.New("extracted audio is empty"))
	}

	projectID, err := parseProjectID(ctx, in.ProjectID)
	if err != nil {
		return workflow.ExtractAudioOutput{}, err
	}
	name := stem(in.Name) + ".wav"
	fileID, err := a.publish(ctx, in.JobID, ownerID, projectID, artifactbus.KindAudio, name, "audio/wav", body)
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

// pullAndExtract downloads the source and writes the WAV.
// Progress facts go through emit, the same path the other steps use.
func (a *Activities) pullAndExtract(ctx context.Context, jobID, bucket, key, src, dst string) error {
	var total int64
	if info, err := a.objects.HeadObject(ctx, bucket, key); err == nil {
		total = info.Size
	}

	var downloaded atomic.Int64
	var live audioLive
	return a.emit(ctx, jobID, "extract audio", "Downloading the recording.", pipelinejobbus.AudioStage, func(shown float64) factSample {
		live.noteDownload(total, downloaded.Load())
		_, _, message, ok := live.snapshot()
		if !ok {
			return factSample{}
		}
		measured, has := live.measured(shown)
		return factSample{Fraction: measured, Has: has, Message: message}
	}, func(ctx context.Context) error {
		return a.writeExtractedWAV(ctx, bucket, key, src, dst, &downloaded, &live)
	})
}

func (a *Activities) writeExtractedWAV(ctx context.Context, bucket, key, src, dst string, downloaded *atomic.Int64, live *audioLive) error {
	f, err := os.Create(src)
	if err != nil {
		return err
	}
	readErr := a.objects.GetObject(ctx, bucket, key, countingWriter{w: f, n: downloaded})
	closeErr := f.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	live.startEncode()
	return audioextract.ExtractProgress(ctx, a.ffmpeg, src, dst, false, func(sample audioextract.Progress) {
		live.setEncode(sample)
	})
}

// audioProgressStep is the most the shown percent may move in one second.
const audioProgressStep = 0.05

// audioLive is the newest sentence and fraction for the audio step.
// Download updates stop once ffmpeg has started. The encode fraction fills
// whatever part of the bar the download has not already used.
type audioLive struct {
	mu          sync.Mutex
	phase       string
	fraction    float64
	hasFraction bool
	message     string
	floor       float64
	latched     bool
}

func (p *audioLive) noteDownload(total, n int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.phase == "encode" {
		return
	}
	p.phase = "download"
	p.message = "Downloading the recording."
	if total <= 0 || n <= 0 {
		return
	}
	p.fraction = float64(n) / float64(total)
	if p.fraction > 1 {
		p.fraction = 1
	}
	p.hasFraction = true
}

func (p *audioLive) startEncode() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase = "encode"
	p.hasFraction = false
	p.fraction = 0
	p.message = "Writing the audio."
}

func (p *audioLive) setEncode(sample audioextract.Progress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase = "encode"
	p.fraction = sample.Fraction
	if p.fraction > 1 {
		p.fraction = 1
	}
	p.hasFraction = true
	p.message = "Writing the audio."
	if extra := remainingText(sample.Remaining); extra != "" {
		p.message += " " + extra
	}
}

func (p *audioLive) snapshot() (fraction float64, hasFraction bool, message string, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.message == "" {
		return 0, false, "", false
	}
	return p.fraction, p.hasFraction, p.message, true
}

// measured is the fraction the bar is heading toward. The first encode sample
// continues from the percent already shown.
func (p *audioLive) measured(shown float64) (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.phase {
	case "download":
		if !p.hasFraction {
			return 0, false
		}
		return p.fraction, true
	case "encode":
		if !p.latched {
			p.floor = shown
			if p.floor < 0 {
				p.floor = 0
			}
			p.latched = true
		}
		if !p.hasFraction {
			return p.floor, true
		}
		return p.floor + p.fraction*(1-p.floor), true
	default:
		return 0, false
	}
}

// advanceProgress moves the shown fraction toward measured, and never backward.
func advanceProgress(shown float64, haveShown bool, measured float64) float64 {
	if measured < 0 {
		measured = 0
	}
	if measured > 1 {
		measured = 1
	}
	if !haveShown {
		if measured > audioProgressStep {
			return audioProgressStep
		}
		return measured
	}
	if measured < shown {
		return shown
	}
	next := shown + audioProgressStep
	if measured < next {
		return measured
	}
	return next
}

func remainingText(left time.Duration) string {
	if left <= 0 {
		return ""
	}
	switch {
	case left < 45*time.Second:
		return "Less than a minute left."
	case left < 90*time.Second:
		return "About a minute left."
	default:
		minutes := int((left + 30*time.Second) / time.Minute)
		return fmt.Sprintf("About %d min left.", minutes)
	}
}

type countingWriter struct {
	w io.Writer
	n *atomic.Int64
}

func (c countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		c.n.Add(int64(n))
	}
	return n, err
}

func sourceExt(name, contentType string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(name)))
	if ext != "" && ext != "." {
		return ext
	}
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "webm"):
		return ".webm"
	case strings.Contains(ct, "mp4"), strings.Contains(ct, "m4a"):
		return ".mp4"
	case strings.Contains(ct, "mpeg"), strings.Contains(ct, "mp3"):
		return ".mp3"
	case strings.Contains(ct, "wav"):
		return ".wav"
	case strings.Contains(ct, "ogg"):
		return ".ogg"
	case strings.Contains(ct, "flac"):
		return ".flac"
	default:
		return ".bin"
	}
}

func (a *Activities) Transcribe(ctx context.Context, in workflow.TranscribeInput) (workflow.TranscribeOutput, error) {
	out, err := a.transcribe(ctx, in)
	return out, a.complete(ctx, in.JobID, out.FileID, err, transcriptDone, transcriptFailed, transcriptCanceled)
}

func (a *Activities) transcribe(ctx context.Context, in workflow.TranscribeInput) (workflow.TranscribeOutput, error) {
	if err := a.stub(ctx, in.JobID, "transcribe", "Writing the transcript.", pipelinejobbus.TranscriptStage); err != nil {
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
	if err != nil {
		return out, a.finish(ctx, in.JobID, err, keyFramesFailed, keyFramesCanceled)
	}
	if strings.HasPrefix(in.ContentType, "audio/") {
		return out, a.record(context.WithoutCancel(a.traced(ctx)), in.JobID, pipelinejobbus.KeyFramesSkipped{})
	}
	return out, a.complete(ctx, in.JobID, out.FileID, nil, keyFramesDone, keyFramesFailed, keyFramesCanceled)
}

func (a *Activities) extractKeyFrames(ctx context.Context, in workflow.ExtractKeyFramesInput) (workflow.ExtractKeyFramesOutput, error) {
	if strings.HasPrefix(in.ContentType, "audio/") {
		return workflow.ExtractKeyFramesOutput{JobID: in.JobID}, nil
	}
	if err := a.stub(ctx, in.JobID, "extract key frames", "Writing a key frame.", pipelinejobbus.KeyFramesStage); err != nil {
		return workflow.ExtractKeyFramesOutput{}, err
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
		return out, a.finish(ctx, in.JobID, err, documentFailed, documentCanceled)
	}
	return out, a.complete(ctx, in.JobID, out.FileID, nil, documentDone, documentFailed, documentCanceled)
}

func (a *Activities) composeExport(ctx context.Context, in workflow.ComposeExportInput) (workflow.ComposeExportOutput, error) {
	if err := a.stub(ctx, in.JobID, "compose export", "Writing the document.", pipelinejobbus.DocumentStage); err != nil {
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
	if err := ctx.Err(); err != nil {
		return uuid.Nil, err
	}
	return fileID, nil
}

// loadJob parses the id, requires the jobs client, and loads the row.
// what is the prefix of the error, such as "record step".
func (a *Activities) loadJob(ctx context.Context, jobID, what string) (pipelinejobbus.Job, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return pipelinejobbus.Job{}, invalidInput(ctx, "job id", err)
	}
	if a.jobs == nil {
		return pipelinejobbus.Job{}, fmt.Errorf("%s: pipeline jobs not configured", what)
	}
	job, err := a.jobs.QueryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pipelinejobbus.ErrNotFound) {
			return pipelinejobbus.Job{}, invalidInput(ctx, "job id", err)
		}
		return pipelinejobbus.Job{}, fmt.Errorf("%s: %w", what, err)
	}
	return job, nil
}

// RecordRun asks the business to append the ending fact.
func (a *Activities) RecordRun(ctx context.Context, in workflow.RecordRunInput) error {
	return a.recordRun(ctx, in.JobID, in.Fact, in.Message)
}

func (a *Activities) recordRun(ctx context.Context, jobID, factName, message string) error {
	var note *string
	if message != "" {
		note = &message
	}
	cause, err := pipelinejobbus.Ending(factName, note)
	if err != nil {
		return invalidInput(ctx, "fact", err)
	}
	storeCtx := context.WithoutCancel(a.traced(ctx))
	job, err := a.loadJob(storeCtx, jobID, "record run")
	if err != nil {
		return err
	}
	if _, err := a.jobs.RecordRun(storeCtx, job, cause); err != nil {
		return fmt.Errorf("record run: %w", err)
	}
	return nil
}

// complete records the stage's finished sentence. A failure is stored by finish.
// fileID is the file this stage produced, or empty when the sentence has none.
func (a *Activities) complete(ctx context.Context, jobID, fileID string, err error, done func(string) (pipelinejobbus.Cause, error), failed func(*string) pipelinejobbus.Cause, stop func() pipelinejobbus.Cause) error {
	if err != nil {
		return a.finish(ctx, jobID, err, failed, stop)
	}
	cause, err := done(fileID)
	if err != nil {
		return invalidInput(ctx, "file id", err)
	}
	return a.record(context.WithoutCancel(a.traced(ctx)), jobID, cause)
}

// recordLaterProgress stores a later progress update. A failed write does not fail the activity.
func (a *Activities) recordLaterProgress(ctx context.Context, job pipelinejobbus.Job, stage pipelinejobbus.Stage, fraction *float64, message string) {
	_ = a.jobs.RecordProgress(a.traced(ctx), job, stage, fraction, headline(message))
}

// recordProgressJob stores the first progress update. A failed write fails the activity.
func (a *Activities) recordProgressJob(ctx context.Context, jobID string, stage pipelinejobbus.Stage, fraction *float64, headline *string) (pipelinejobbus.Job, error) {
	job, err := a.loadJob(ctx, jobID, "progress")
	if err != nil {
		return pipelinejobbus.Job{}, err
	}
	if err := a.jobs.RecordProgress(ctx, job, stage, fraction, headline); err != nil {
		return pipelinejobbus.Job{}, fmt.Errorf("progress: %w", err)
	}
	return job, nil
}

func (a *Activities) record(ctx context.Context, jobID string, cause pipelinejobbus.Cause) error {
	_, err := a.recordJob(ctx, jobID, cause)
	return err
}

func (a *Activities) recordJob(ctx context.Context, jobID string, cause pipelinejobbus.Cause) (pipelinejobbus.Job, error) {
	job, err := a.loadJob(ctx, jobID, "record")
	if err != nil {
		return pipelinejobbus.Job{}, err
	}
	stored, err := a.jobs.Record(ctx, job, cause)
	if err != nil {
		return pipelinejobbus.Job{}, fmt.Errorf("record: %w", err)
	}
	return stored, nil
}

// finish stores the stage's cancel or failure sentence.
// The run itself is recorded by the workflow after this activity returns.
func (a *Activities) finish(ctx context.Context, jobID string, err error, failed func(*string) pipelinejobbus.Cause, stop func() pipelinejobbus.Cause) error {
	if err == nil {
		return nil
	}
	switch {
	case canceled(ctx, err):
		if recErr := a.record(context.WithoutCancel(a.traced(ctx)), jobID, stop()); recErr != nil {
			return recErr
		}
		return err
	case recordFailure(ctx, err):
		if recErr := a.record(context.WithoutCancel(a.traced(ctx)), jobID, failed(headline(failureMessage(err)))); recErr != nil {
			return recErr
		}
		return err
	default:
		return err
	}
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
