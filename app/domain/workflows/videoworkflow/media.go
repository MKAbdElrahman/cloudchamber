package videoworkflow

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"
)

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

// stem is the file name without its extension.
func stem(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "recording"
	}
	if before, _, ok := strings.CutLast(trimmed, "."); ok && before != "" {
		return before
	}
	return trimmed
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
