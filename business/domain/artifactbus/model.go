package artifactbus

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	KindSource     = "source"
	KindAudio      = "audio"
	KindTranscript = "transcript"
	KindFrames     = "frames"
	KindDocument   = "document"
)

var validKinds = map[string]struct{}{
	KindSource:     {},
	KindAudio:      {},
	KindTranscript: {},
	KindFrames:     {},
	KindDocument:   {},
}

// Kind is the role of a stored object.
type Kind struct {
	value string
}

// ParseKind parses a kind value.
func ParseKind(value string) (Kind, error) {
	if _, ok := validKinds[value]; !ok {
		return Kind{}, fmt.Errorf("invalid kind %q", value)
	}
	return Kind{value: value}, nil
}

// MustParseKind parses a kind value and panics on error.
func MustParseKind(value string) Kind {
	kind, err := ParseKind(value)
	if err != nil {
		panic(err)
	}
	return kind
}

// String returns the kind value.
func (k Kind) String() string {
	return k.value
}

// Equal reports whether two kinds are the same.
func (k Kind) Equal(other Kind) bool {
	return k.value == other.value
}

// MarshalText provides support for logging and marshaling.
func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k.value), nil
}

const (
	StatusReady = "ready"
)

var validStatuses = map[string]struct{}{
	StatusReady: {},
}

// Status is whether a stored object can be read.
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
	status, err := ParseStatus(value)
	if err != nil {
		panic(err)
	}
	return status
}

// String returns the status value.
func (s Status) String() string {
	return s.value
}

// Equal reports whether two statuses are the same.
func (s Status) Equal(other Status) bool {
	return s.value == other.value
}

// MarshalText provides support for logging and marshaling.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

// Artifact is one durable object a person can read.
type Artifact struct {
	FileID      uuid.UUID
	OwnerID     uuid.UUID
	Bucket      string
	ObjectKey   string
	Kind        Kind
	ContentType string
	Name        string
	Status      Status
	DateCreated time.Time
	DateUpdated time.Time
}

// NewArtifact is the data required to register an object that is already in the bucket.
type NewArtifact struct {
	FileID      uuid.UUID
	OwnerID     uuid.UUID
	Bucket      string
	ObjectKey   string
	Kind        Kind
	ContentType string
	Name        string
	Status      Status
}

// Playback is a short-lived way to read one object.
type Playback struct {
	URL       string
	ExpiresAt time.Time
}
