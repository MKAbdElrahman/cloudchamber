package pipelinejobbus

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Stage names the activity a progress update is about.
const (
	StageSource     = "source"
	StageAudio      = "audio"
	StageTranscript = "transcript"
	StageKeyFrames  = "keyFrames"
	StageDocument   = "document"
)

// Stage is one pipeline activity. It is not a business event.
type Stage struct {
	value string
}

// ParseStage parses a stage name.
func ParseStage(value string) (Stage, error) {
	switch value {
	case StageSource, StageAudio, StageTranscript, StageKeyFrames, StageDocument:
		return Stage{value: value}, nil
	default:
		return Stage{}, fmt.Errorf("invalid stage %q", value)
	}
}

// String returns the stage name.
func (s Stage) String() string {
	return s.value
}

// The five activities a progress update can name.
var (
	SourceStage     = Stage{value: StageSource}
	AudioStage      = Stage{value: StageAudio}
	TranscriptStage = Stage{value: StageTranscript}
	KeyFramesStage  = Stage{value: StageKeyFrames}
	DocumentStage   = Stage{value: StageDocument}
)

// Progress is a live report while an activity runs.
// It carries the fraction and the headline. It is not a cause and the fold does not read it.
type Progress struct {
	ID             uuid.UUID
	JobID          uuid.UUID
	Position       int64
	GlobalPosition int64
	Time           time.Time
	Stage          Stage
	Fraction       *float64
	Headline       *string
}

// Log is one page of the job stream: business events and progress updates, in stream order.
type Log struct {
	Facts    []Fact
	Progress []Progress
}
