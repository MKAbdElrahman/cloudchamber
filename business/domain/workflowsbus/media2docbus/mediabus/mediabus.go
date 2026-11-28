// Package mediabus holds the rules of the media worker: which recording a
// run may convert, and how its audio is extracted.
package mediabus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/mediabus/audioextract"
	"github.com/google/uuid"
)

// Set of error variables for audio extraction.
var (
	// ErrNoAudio is returned when a recording has no audio to extract.
	ErrNoAudio = errors.New("the recording has no audio")
	// ErrFFmpegNotFound is returned when the ffmpeg binary is not on the worker.
	ErrFFmpegNotFound = audioextract.ErrNotFound
)

// Progress receives how far a step has got, from 0 to 1, and what it is doing.
// A negative fraction means the amount is not known.
type Progress func(fraction float64, message string)

// Business manages the media worker's steps.
type Business struct {
	files  media2docbus.Files
	ffmpeg string
}

// NewBusiness constructs the media business. ffmpeg is the binary used to
// extract audio, and an empty path searches PATH.
func NewBusiness(files media2docbus.Files, ffmpeg string) *Business {
	return &Business{files: files, ffmpeg: ffmpeg}
}

// CheckSource returns the recording when the owner may convert it in the project.
// It does not look at the recording's bytes, so it is cheap enough to run on a request.
func (b *Business) CheckSource(ctx context.Context, ownerID, projectID, fileID uuid.UUID) (media2docbus.Source, error) {
	src, err := b.files.QuerySource(ctx, fileID)
	if err != nil {
		return media2docbus.Source{}, fmt.Errorf("checksource: %w", err)
	}
	if err := src.Eligible(ownerID, projectID); err != nil {
		return media2docbus.Source{}, fmt.Errorf("checksource: %w", err)
	}
	return src, nil
}

// ResolveSource is CheckSource for a run that is starting: it also confirms
// the recording's bytes are in storage.
func (b *Business) ResolveSource(ctx context.Context, ownerID, projectID, fileID uuid.UUID) (media2docbus.Source, error) {
	src, err := b.CheckSource(ctx, ownerID, projectID, fileID)
	if err != nil {
		return media2docbus.Source{}, err
	}
	if _, err := b.files.SourceSize(ctx, fileID); err != nil {
		return media2docbus.Source{}, fmt.Errorf("resolvesource: %w", err)
	}
	if src.Name == "" {
		src.Name = "recording"
	}
	return src, nil
}

// ExtractAudio downloads the recording, writes it as 16 kHz mono 16-bit PCM
// WAV, and publishes that as the run's audio file. The download is the first
// half of the progress and the encode is the second.
func (b *Business) ExtractAudio(ctx context.Context, runID uuid.UUID, src media2docbus.Source, progress Progress) (uuid.UUID, error) {
	dir, err := os.MkdirTemp("", "cloudchamber-audio-")
	if err != nil {
		return uuid.Nil, fmt.Errorf("extractaudio: %w", err)
	}
	defer os.RemoveAll(dir)

	in := filepath.Join(dir, "source"+sourceExt(src.Name, src.ContentType))
	out := filepath.Join(dir, "audio.wav")

	if err := b.download(ctx, src.FileID, in, progress); err != nil {
		return uuid.Nil, fmt.Errorf("extractaudio: download: %w", err)
	}

	progress(0.5, "Writing the audio.")
	err = audioextract.ExtractProgress(ctx, b.ffmpeg, in, out, false, func(sample audioextract.Progress) {
		progress(0.5+min(sample.Fraction, 1)/2, encodeMessage(sample.Remaining))
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("extractaudio: encode: %w", err)
	}

	body, err := os.ReadFile(out)
	if err != nil {
		return uuid.Nil, fmt.Errorf("extractaudio: %w", err)
	}
	if len(body) == 0 {
		return uuid.Nil, fmt.Errorf("extractaudio: %w", ErrNoAudio)
	}

	return media2docbus.Publish(ctx, b.files, media2docbus.Output{
		RunID:       runID,
		OwnerID:     src.OwnerID,
		ProjectID:   src.ProjectID,
		Kind:        media2docbus.KindAudio,
		Name:        media2docbus.Stem(src.Name) + ".wav",
		ContentType: "audio/wav",
		Body:        body,
	})
}

// download writes the recording to path and reports how much has arrived.
func (b *Business) download(ctx context.Context, fileID uuid.UUID, path string, progress Progress) error {
	const message = "Downloading the recording."

	// The size is only for the progress report. Without it the amount is unknown.
	total, _ := b.files.SourceSize(ctx, fileID)
	progress(-1, message)

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	var got atomic.Int64
	w := writerFunc(func(p []byte) (int, error) {
		n, err := f.Write(p)
		if n > 0 && total > 0 {
			progress(min(float64(got.Add(int64(n)))/float64(total), 1)/2, message)
		}
		return n, err
	})

	return errors.Join(b.files.ReadSource(ctx, fileID, w), f.Close())
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

var _ io.Writer = writerFunc(nil)

// encodeMessage is the sentence for the encode, with how long is left when ffmpeg can tell.
func encodeMessage(left time.Duration) string {
	switch {
	case left <= 0:
		return "Writing the audio."
	case left < 45*time.Second:
		return "Writing the audio. Less than a minute left."
	case left < 90*time.Second:
		return "Writing the audio. About a minute left."
	default:
		return fmt.Sprintf("Writing the audio. About %d min left.", int((left+30*time.Second)/time.Minute))
	}
}

// sourceExt is the extension ffmpeg should see for the recording.
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
