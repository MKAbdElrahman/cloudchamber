package artifactapp

import (
	"encoding/json"
	"time"

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
