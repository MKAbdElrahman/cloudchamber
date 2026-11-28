// Package framesbus holds the rules of the frames worker: which recordings
// have key frames, and how a run's key frames are produced.
package framesbus

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/google/uuid"
)

// Business manages the frames worker's step.
type Business struct {
	files media2docbus.Files
}

// NewBusiness constructs the frames business.
func NewBusiness(files media2docbus.Files) *Business {
	return &Business{files: files}
}

// HasPicture reports whether a recording of this content type has frames to
// choose from. A recording that is only audio has none, and its run has no key-frame step.
func HasPicture(contentType string) bool {
	return !strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "audio/")
}

// ExtractKeyFrames writes the run's key frame and publishes it.
// It is a stand-in: it does not read the recording.
func (b *Business) ExtractKeyFrames(ctx context.Context, runID uuid.UUID, src media2docbus.Source) (uuid.UUID, error) {
	body, err := framePNG()
	if err != nil {
		return uuid.Nil, fmt.Errorf("extractkeyframes: %w", err)
	}

	return media2docbus.Publish(ctx, b.files, media2docbus.Output{
		RunID:       runID,
		OwnerID:     src.OwnerID,
		ProjectID:   src.ProjectID,
		Kind:        media2docbus.KindFrames,
		Name:        media2docbus.Stem(src.Name) + "-frame.png",
		ContentType: "image/png",
		Body:        body,
	})
}

// framePNG is a small image standing in for one extracted video frame.
func framePNG() ([]byte, error) {
	const width, height = 64, 36
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(20 + x*3), G: uint8(40 + y*4), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
