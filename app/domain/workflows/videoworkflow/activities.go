package videoworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/workflows/videoworkflow/audioextract"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runworker"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
)

// simulated is how long a stand-in step works, so a person can follow it and stop it.
const simulated = 20 * time.Second

// tick is how often a working step heartbeats and reports how far it is.
const tick = 500 * time.Millisecond

// Activities are the steps of the workflow. ExtractAudio is real. Transcribe,
// ExtractKeyFrames and ComposeDocument are stand-ins that write a small file of
// the right type. Every step registers what it wrote as a file in the run's project.
type Activities struct {
	artifacts artifactbus.ExtBusiness
	objects   *objectstore.Client
	tracer    trace.Tracer
	ffmpeg    string
}

// NewActivities constructs the activities. ffmpeg is the binary ExtractAudio
// uses, and an empty path searches PATH.
func NewActivities(artifacts artifactbus.ExtBusiness, objects *objectstore.Client, tracer trace.Tracer, ffmpeg string) *Activities {
	return &Activities{artifacts: artifacts, objects: objects, tracer: tracer, ffmpeg: ffmpeg}
}

// traced stores the worker tracer the way the HTTP middleware does, so
// business spans started with otel.AddSpan join the activity trace.
func (a *Activities) traced(ctx context.Context) context.Context {
	return otel.InjectTracing(ctx, a.tracer)
}

// ResolveSource checks that the recording is a ready file in the run's project
// and that its bytes are in storage. The run API does not know what a file is,
// so this step is where the workflow decides whether the caller may convert it.
func (a *Activities) ResolveSource(ctx context.Context, in ResolveInput) (Source, error) {
	runworker.Progress(ctx, -1, "Loading the recording.")

	fileID, err := uuid.Parse(in.FileID)
	if err != nil {
		return Source{}, invalidInput(ctx, "fileId is not a file id", err)
	}

	art, err := a.artifacts.QueryByFileID(a.traced(ctx), fileID)
	if err != nil {
		if errors.Is(err, artifactbus.ErrNotFound) {
			return Source{}, permanent(ctx, errTypeSourceNotFound, "the recording does not exist", err)
		}
		return Source{}, fmt.Errorf("load source: %w", err)
	}

	ready := art.Status.Equal(artifactbus.MustParseStatus(artifactbus.StatusReady))
	switch {
	case art.OwnerID.String() != in.OwnerID, art.ProjectID.String() != in.ProjectID:
		return Source{}, permanent(ctx, errTypeSourceNotFound, "the recording is not in this project", nil)
	case !ready:
		return Source{}, permanent(ctx, errTypeSourceNotFound, "the recording has not finished uploading", nil)
	}

	if _, err := a.objects.HeadObject(ctx, art.Bucket, art.ObjectKey); err != nil {
		if objectstore.IsMissing(err) {
			return Source{}, permanent(ctx, errTypeSourceNotFound, "the recording is missing from storage", err)
		}
		return Source{}, fmt.Errorf("head source: %w", err)
	}

	name := art.Name
	if name == "" {
		name = "recording"
	}

	return Source{
		RunID:       in.RunID,
		OwnerID:     in.OwnerID,
		ProjectID:   in.ProjectID,
		FileID:      art.FileID.String(),
		Name:        name,
		ContentType: art.ContentType,
	}, nil
}

// ExtractAudio downloads the recording and writes 16 kHz mono 16-bit PCM WAV.
// An attempt that is stopped starts the download again when it continues.
func (a *Activities) ExtractAudio(ctx context.Context, src Source) (File, error) {
	fileID, err := uuid.Parse(src.FileID)
	if err != nil {
		return File{}, invalidInput(ctx, "fileId is not a file id", err)
	}

	art, err := a.artifacts.QueryByFileID(a.traced(ctx), fileID)
	if err != nil {
		if errors.Is(err, artifactbus.ErrNotFound) {
			return File{}, permanent(ctx, errTypeSourceNotFound, "the recording does not exist", err)
		}
		return File{}, fmt.Errorf("load source: %w", err)
	}

	dir, err := os.MkdirTemp("", "cloudchamber-audio-")
	if err != nil {
		return File{}, fmt.Errorf("extract audio: %w", err)
	}
	defer os.RemoveAll(dir)

	in := filepath.Join(dir, "source"+sourceExt(art.Name, art.ContentType))
	out := filepath.Join(dir, "audio.wav")

	if err := a.pullAndExtract(ctx, art.Bucket, art.ObjectKey, in, out); err != nil {
		switch {
		case errors.Is(err, audioextract.ErrNotFound):
			return File{}, permanent(ctx, "FFmpegNotFound", "ffmpeg was not found on the worker", err)
		case objectstore.IsMissing(err):
			return File{}, permanent(ctx, errTypeSourceNotFound, "the recording is missing from storage", err)
		default:
			return File{}, fmt.Errorf("extract audio: %w", err)
		}
	}

	body, err := os.ReadFile(out)
	if err != nil {
		return File{}, fmt.Errorf("extract audio: %w", err)
	}
	if len(body) == 0 {
		return File{}, invalidInput(ctx, "the recording has no audio", nil)
	}

	return a.publish(ctx, src, artifactbus.KindAudio, stem(src.Name)+".wav", "audio/wav", body)
}

// pullAndExtract downloads the source and encodes it. While it works, a pulse
// heartbeats and reports progress: the download is the first half of the bar
// and the encode is the second.
func (a *Activities) pullAndExtract(ctx context.Context, bucket, key, in, out string) error {
	var total int64
	if info, err := a.objects.HeadObject(ctx, bucket, key); err == nil {
		total = info.Size
	}

	var live audioProgress
	var downloaded atomic.Int64

	stop := pulse(ctx, func() (float64, string) {
		return live.sample(total, downloaded.Load())
	})
	defer stop()

	f, err := os.Create(in)
	if err != nil {
		return err
	}
	readErr := a.objects.GetObject(ctx, bucket, key, countingWriter{w: f, n: &downloaded})
	closeErr := f.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}

	live.startEncode()
	return audioextract.ExtractProgress(ctx, a.ffmpeg, in, out, false, live.setEncode)
}

// Transcribe writes the transcript. It is a stand-in: it does not read the audio.
func (a *Activities) Transcribe(ctx context.Context, in TranscribeInput) (File, error) {
	if err := simulate(ctx, "Writing the transcript."); err != nil {
		return File{}, err
	}

	body, err := json.Marshal(transcript(in.Source.Name))
	if err != nil {
		return File{}, err
	}

	return a.publish(ctx, in.Source, artifactbus.KindTranscript, stem(in.Source.Name)+".transcript.json", "application/json", body)
}

// ExtractKeyFrames writes one frame. It is a stand-in: it does not read the recording.
func (a *Activities) ExtractKeyFrames(ctx context.Context, src Source) (File, error) {
	if err := simulate(ctx, "Choosing the key frames."); err != nil {
		return File{}, err
	}

	body, err := framePNG()
	if err != nil {
		return File{}, err
	}

	return a.publish(ctx, src, artifactbus.KindFrames, stem(src.Name)+"-frame.png", "image/png", body)
}

// ComposeDocument writes the document. It is a stand-in: it does not read the transcript or the frames.
func (a *Activities) ComposeDocument(ctx context.Context, in ComposeInput) (File, error) {
	if err := simulate(ctx, "Writing the document."); err != nil {
		return File{}, err
	}

	body := []byte(documentMarkdown(in.Source.Name, in.KeyFramesFileID != ""))

	return a.publish(ctx, in.Source, artifactbus.KindDocument, stem(in.Source.Name)+".md", "text/markdown", body)
}

// publish stores the bytes and registers them as a file in the run's project.
// The file id is derived from the run and the kind of file, so a step that
// runs again replaces its own file instead of adding another.
func (a *Activities) publish(ctx context.Context, src Source, kind, name, contentType string, body []byte) (File, error) {
	ownerID, err := uuid.Parse(src.OwnerID)
	if err != nil {
		return File{}, invalidInput(ctx, "ownerId is not an id", err)
	}
	projectID, err := uuid.Parse(src.ProjectID)
	if err != nil {
		return File{}, invalidInput(ctx, "projectId is not an id", err)
	}
	if err := ctx.Err(); err != nil {
		return File{}, err
	}

	ctx = a.traced(ctx)
	fileID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(src.RunID+"/"+kind))
	key := fmt.Sprintf("outputs/%s/%s", ownerID, fileID)

	if err := a.objects.PutObject(ctx, "", key, contentType, body); err != nil {
		return File{}, err
	}
	if err := ctx.Err(); err != nil {
		return File{}, err
	}

	_, err = a.artifacts.Register(ctx, artifactbus.NewArtifact{
		FileID:      fileID,
		OwnerID:     ownerID,
		ProjectID:   projectID,
		Bucket:      a.objects.Bucket(),
		ObjectKey:   key,
		Kind:        artifactbus.MustParseKind(kind),
		ContentType: contentType,
		Name:        name,
		Status:      artifactbus.MustParseStatus(artifactbus.StatusReady),
	})
	if err != nil {
		return File{}, fmt.Errorf("register %s: %w", kind, err)
	}

	return File{FileID: fileID.String()}, nil
}

// =============================================================================

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

// pulse heartbeats and reports the sampled progress on every tick, for work
// that blocks in a call it cannot interrupt to report. Call stop when the work returns.
func pulse(ctx context.Context, sample func() (fraction float64, message string)) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Go(func() {
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for {
			fraction, message := sample()
			runworker.Progress(ctx, fraction, message)
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx)
			}
		}
	})

	return func() {
		close(done)
		wg.Wait()
	}
}

// audioProgress is how far the audio step is. The download is the first half
// of the bar and the encode is the second.
type audioProgress struct {
	mu        sync.Mutex
	encoding  bool
	fraction  float64
	measured  bool
	remaining time.Duration
}

func (p *audioProgress) startEncode() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.encoding = true
	p.measured = false
}

func (p *audioProgress) setEncode(sample audioextract.Progress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fraction = min(sample.Fraction, 1)
	p.measured = true
	p.remaining = sample.Remaining
}

// sample returns the fraction of the whole step and the sentence to show.
// A negative fraction means the amount is not known yet.
func (p *audioProgress) sample(total, downloaded int64) (float64, string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch {
	case !p.encoding && total > 0:
		return min(float64(downloaded)/float64(total), 1) / 2, "Downloading the recording."
	case !p.encoding:
		return -1, "Downloading the recording."
	case !p.measured:
		return 0.5, "Writing the audio."
	default:
		message := "Writing the audio."
		if extra := remainingText(p.remaining); extra != "" {
			message += " " + extra
		}
		return 0.5 + p.fraction/2, message
	}
}

func remainingText(left time.Duration) string {
	switch {
	case left <= 0:
		return ""
	case left < 45*time.Second:
		return "Less than a minute left."
	case left < 90*time.Second:
		return "About a minute left."
	default:
		return fmt.Sprintf("About %d min left.", int((left+30*time.Second)/time.Minute))
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
