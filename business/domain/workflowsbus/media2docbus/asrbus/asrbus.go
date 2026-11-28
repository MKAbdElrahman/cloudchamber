// Package asrbus holds the rules of the ASR worker: how a run's transcript is produced.
package asrbus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/google/uuid"
)

// Business manages the ASR worker's step.
type Business struct {
	files media2docbus.Files
}

// NewBusiness constructs the ASR business.
func NewBusiness(files media2docbus.Files) *Business {
	return &Business{files: files}
}

// Transcribe writes the transcript of the run's audio and publishes it.
// It is a stand-in: it does not read the audio.
func (b *Business) Transcribe(ctx context.Context, runID uuid.UUID, src media2docbus.Source) (uuid.UUID, error) {
	body, err := json.Marshal(transcript(src.Name))
	if err != nil {
		return uuid.Nil, fmt.Errorf("transcribe: %w", err)
	}

	return media2docbus.Publish(ctx, b.files, media2docbus.Output{
		RunID:       runID,
		OwnerID:     src.OwnerID,
		ProjectID:   src.ProjectID,
		Kind:        media2docbus.KindTranscript,
		Name:        media2docbus.Stem(src.Name) + ".transcript.json",
		ContentType: "application/json",
		Body:        body,
	})
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
