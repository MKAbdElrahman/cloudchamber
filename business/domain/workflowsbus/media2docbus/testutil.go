package media2docbus

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
)

// TestFiles is a file store in memory, for testing the workers' business packages.
type TestFiles struct {
	mu      sync.Mutex
	Sources map[uuid.UUID]Source
	Bytes   map[uuid.UUID][]byte
	Outputs map[uuid.UUID]Output
}

// NewTestFiles constructs an empty in-memory file store.
func NewTestFiles() *TestFiles {
	return &TestFiles{
		Sources: make(map[uuid.UUID]Source),
		Bytes:   make(map[uuid.UUID][]byte),
		Outputs: make(map[uuid.UUID]Output),
	}
}

// AddSource stores a recording and its bytes. Nil bytes leave the recording with none in storage.
func (f *TestFiles) AddSource(src Source, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Sources[src.FileID] = src
	if body != nil {
		f.Bytes[src.FileID] = body
	}
}

// QuerySource implements Files.
func (f *TestFiles) QuerySource(_ context.Context, fileID uuid.UUID) (Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	src, ok := f.Sources[fileID]
	if !ok {
		return Source{}, ErrSourceNotFound
	}
	return src, nil
}

// SourceSize implements Files.
func (f *TestFiles) SourceSize(_ context.Context, fileID uuid.UUID) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, ok := f.Bytes[fileID]
	if !ok {
		return 0, ErrSourceMissing
	}
	return int64(len(body)), nil
}

// ReadSource implements Files.
func (f *TestFiles) ReadSource(_ context.Context, fileID uuid.UUID, dst io.Writer) error {
	f.mu.Lock()
	body, ok := f.Bytes[fileID]
	f.mu.Unlock()
	if !ok {
		return ErrSourceMissing
	}
	_, err := dst.Write(body)
	return err
}

// SaveOutput implements Files.
func (f *TestFiles) SaveOutput(_ context.Context, fileID uuid.UUID, out Output) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Outputs[fileID] = out
	return nil
}
