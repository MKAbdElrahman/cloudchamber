// Package renderbus holds the rules of the render worker: how a run's document is composed.
package renderbus

import (
	"context"
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/google/uuid"
)

// Business manages the render worker's step.
type Business struct {
	files media2docbus.Files
}

// NewBusiness constructs the render business.
func NewBusiness(files media2docbus.Files) *Business {
	return &Business{files: files}
}

// Compose writes the run's document and publishes it. withFrames says
// whether the run produced key frames for it to include.
// It is a stand-in: it does not read the transcript or the frames.
func (b *Business) Compose(ctx context.Context, runID uuid.UUID, src media2docbus.Source, withFrames bool) (uuid.UUID, error) {
	return media2docbus.Publish(ctx, b.files, media2docbus.Output{
		RunID:       runID,
		OwnerID:     src.OwnerID,
		ProjectID:   src.ProjectID,
		Kind:        media2docbus.KindDocument,
		Name:        media2docbus.Stem(src.Name) + ".md",
		ContentType: "text/markdown",
		Body:        []byte(markdown(src.Name, withFrames)),
	})
}

func markdown(name string, withFrames bool) string {
	frame := ""
	if withFrames {
		frame = "\n\n> Frame at 4:18. A slide with three boxes: the recording, the transcript, and this document.\n"
	}
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
`, name, frame, media2docbus.Stem(name))
}
