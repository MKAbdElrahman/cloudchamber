package artifactapp

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
)

// Playback is the short-lived read ticket returned to the browser.
type Playback struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expiresAt"`
}

// Encode implements the encoder interface.
func (p Playback) Encode() ([]byte, string, error) {
	data, err := json.Marshal(p)
	return data, "application/json", err
}

func fromBusPlaybackResponse(bus artifactbus.Playback) Playback {
	return Playback{
		URL:       bus.URL,
		ExpiresAt: bus.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

// PlaybackReport is what the browser observed while reading the object.
type PlaybackReport struct {
	Outcome    string `json:"outcome"`
	ReadyState int    `json:"readyState"`
	WaitedMs   int    `json:"waitedMs"`
	MediaError *int   `json:"mediaError,omitempty"`
}

// Decode implements the decoder interface.
func (p *PlaybackReport) Decode(data []byte) error {
	return json.Unmarshal(data, p)
}

// Validate checks the browser report is one of the known outcomes.
func (p PlaybackReport) Validate() error {
	switch p.Outcome {
	case "loaded", "error":
		return nil
	default:
		return errs.NewFieldErrors("outcome", errors.New("outcome must be loaded or error"))
	}
}
