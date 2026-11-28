// Package media2docbus holds the business rules of the media2doc
// workflow. This file is what its workers share: the recording they read, the
// files they produce, and the store those files live in.
//
// Each worker has its own package under this one, with the rules of its steps:
// mediabus, asrbus, framesbus and renderbus. None of them knows about the
// workflow engine. An activity in the app layer calls them and reports progress.
//
// The workers read and write files through the Files interface. The file
// store belongs to another domain, and a business domain does not import
// another, so the app layer supplies the implementation.
package media2docbus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

// Set of error variables for a recording that cannot be converted.
var (
	ErrSourceNotFound     = errors.New("the recording does not exist")
	ErrSourceNotInProject = errors.New("the recording is not in this project")
	ErrSourceNotReady     = errors.New("the recording has not finished uploading")
	ErrSourceMissing      = errors.New("the recording is missing from storage")
)

// Set of kinds of file a step produces.
const (
	KindAudio      = "audio"
	KindTranscript = "transcript"
	KindFrames     = "frames"
	KindDocument   = "document"
)

// Source is the recording a run converts.
type Source struct {
	FileID      uuid.UUID
	OwnerID     uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	ContentType string
	Ready       bool
}

// Output is one file a step produced.
type Output struct {
	RunID       uuid.UUID
	OwnerID     uuid.UUID
	ProjectID   uuid.UUID
	Kind        string
	Name        string
	ContentType string
	Body        []byte
}

// Files is the file store as the workers need it.
type Files interface {
	// QuerySource returns the recording, or ErrSourceNotFound.
	QuerySource(ctx context.Context, fileID uuid.UUID) (Source, error)
	// SourceSize returns the size of the recording's bytes, or ErrSourceMissing when they are not in storage.
	SourceSize(ctx context.Context, fileID uuid.UUID) (int64, error)
	// ReadSource writes the recording's bytes to dst, or returns ErrSourceMissing.
	ReadSource(ctx context.Context, fileID uuid.UUID, dst io.Writer) error
	// SaveOutput stores the output's bytes under fileID and records it as a
	// ready file in the output's project. Saving the same fileID again replaces it.
	SaveOutput(ctx context.Context, fileID uuid.UUID, out Output) error
}

// Eligible reports whether the owner may convert the recording in the project.
func (s Source) Eligible(ownerID, projectID uuid.UUID) error {
	switch {
	case s.OwnerID != ownerID, s.ProjectID != projectID:
		return ErrSourceNotInProject
	case !s.Ready:
		return ErrSourceNotReady
	default:
		return nil
	}
}

// OutputFileID is the id of the file a run produces for one kind. It is derived
// from the run and the kind, so a step that runs again replaces its own file
// instead of adding another.
func OutputFileID(runID uuid.UUID, kind string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(runID.String()+"/"+kind))
}

// Publish saves an output under its derived id and returns that id.
func Publish(ctx context.Context, files Files, out Output) (uuid.UUID, error) {
	fileID := OutputFileID(out.RunID, out.Kind)
	if err := files.SaveOutput(ctx, fileID, out); err != nil {
		return uuid.Nil, fmt.Errorf("publish %s: %w", out.Kind, err)
	}
	return fileID, nil
}

// Stem is a file name without its extension. An empty name is "recording".
func Stem(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "recording"
	}
	if before, _, ok := strings.CutLast(trimmed, "."); ok && before != "" {
		return before
	}
	return trimmed
}
