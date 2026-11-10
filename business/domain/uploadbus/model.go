package uploadbus

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Set of known upload statuses.
const (
	StatusPending    = "pending"
	StatusReady      = "ready"
	StatusFailed     = "failed"
	StatusTerminated = "terminated"
)

var validStatuses = map[string]struct{}{
	StatusPending:    {},
	StatusReady:      {},
	StatusFailed:     {},
	StatusTerminated: {},
}

// Status represents the reconciled state of an upload.
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
	case StatusReady, StatusFailed, StatusTerminated:
		return true
	default:
		return false
	}
}

// Upload represents a finalized-or-pending upload owned by a user.
type Upload struct {
	ObjectKey    string
	OwnerID      uuid.UUID
	Bucket       string
	ObjectKeyRef string
	SourceRef    string
	SizeBytes    int64
	ETag         *string
	Status       Status
	TusdUploadID *string
	DateCreated  time.Time
	DateUpdated  time.Time
}

// NewUpload contains information needed to create an upload record.
type NewUpload struct {
	ObjectKey    string
	OwnerID      uuid.UUID
	Bucket       string
	ObjectKeyRef string
	SourceRef    string
	SizeBytes    int64
	ETag         *string
	Status       Status
	TusdUploadID *string
}

// UpdateUpload contains information that can be updated for an upload.
type UpdateUpload struct {
	SizeBytes    *int64
	ETag         *string
	ClearETag    bool
	Status       *Status
	TusdUploadID *string
	SourceRef    *string
}
