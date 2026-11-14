package pipelinejobbus

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	// FactType is the message type of one business event.
	FactType = "Fact"
	// ProgressType is the message type of one progress update.
	ProgressType = "Progress"
	// StreamCategory is the category of every job stream.
	// A stream is named job-<job id>. The category is the prefix before the first hyphen,
	// and that shared lock keeps a write behind a consumer that has already moved on.
	StreamCategory = "job"
)

// StreamName returns the stream for one job.
func StreamName(jobID uuid.UUID) string {
	return StreamCategory + "-" + jobID.String()
}

// StreamIDOffset is the 1-based index of the job id inside StreamName.
// StreamName is "<category>-<uuid>".
func StreamIDOffset() int {
	return len(StreamCategory) + 2
}

// JobIDFromStream reads the job id from a job stream name.
func JobIDFromStream(stream string) (uuid.UUID, error) {
	id, ok := strings.CutPrefix(stream, StreamCategory+"-")
	if !ok {
		return uuid.UUID{}, fmt.Errorf("stream %q", stream)
	}
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("stream %q: %w", stream, err)
	}
	return parsed, nil
}
