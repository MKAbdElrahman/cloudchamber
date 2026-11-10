package pipelinejobbus

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Set of known pipeline job statuses.
const (
	StatusSubmitted  = "submitted"
	StatusRunning    = "running"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCanceled   = "canceled"
	StatusTimedOut   = "timed_out"
	StatusTerminated = "terminated"
	StatusUnknown    = "unknown"
)

var validStatuses = map[string]struct{}{
	StatusSubmitted:  {},
	StatusRunning:    {},
	StatusCompleted:  {},
	StatusFailed:     {},
	StatusCanceled:   {},
	StatusTimedOut:   {},
	StatusTerminated: {},
	StatusUnknown:    {},
}

// Status represents the reconciled state of a pipeline job.
type Status struct {
	value string
}

// ParseStatus parses a status value.
func ParseStatus(value string) (Status, error) {
	if _, ok := validStatuses[value]; !ok {
		return Status{}, fmt.Errorf("invalid status %q", value)
	}
	return Status{value: value}, nil
}

// MustParseStatus parses a status value and panics on error.
func MustParseStatus(value string) Status {
	st, err := ParseStatus(value)
	if err != nil {
		panic(err)
	}
	return st
}

// String returns the status value.
func (s Status) String() string {
	return s.value
}

// Equal provides support for testing.
func (s Status) Equal(s2 Status) bool {
	return s.value == s2.value
}

// MarshalText provides support for logging and marshaling.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

// IsTerminal reports whether the status is a terminal state.
func (s Status) IsTerminal() bool {
	switch s.value {
	case StatusCompleted, StatusFailed, StatusCanceled, StatusTimedOut, StatusTerminated:
		return true
	default:
		return false
	}
}

// Job represents a pipeline job owned by a user.
type Job struct {
	ID                 uuid.UUID
	OwnerID            uuid.UUID
	WorkflowID         string
	RunID              string
	SourceRef          string
	IdempotencyKeyHash *string
	Status             Status
	ErrorMessage       *string
	AudioRef           *string
	TranscriptRef      *string
	KeyFramesRef       *string
	DocumentRef        *string
	DateCreated        time.Time
	DateUpdated        time.Time
}

// NewJob contains information needed to create a pipeline job.
type NewJob struct {
	ID                 uuid.UUID
	OwnerID            uuid.UUID
	WorkflowID         string
	RunID              string
	SourceRef          string
	IdempotencyKeyHash *string
	Status             Status
}

// UpdateJob contains information that can be updated for a pipeline job.
type UpdateJob struct {
	RunID         *string
	Status        *Status
	ErrorMessage  *string
	ClearError    bool
	AudioRef      *string
	TranscriptRef *string
	KeyFramesRef  *string
	DocumentRef   *string
}
