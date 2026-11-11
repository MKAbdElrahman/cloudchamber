// Package artifactbus provides business access to readable stored objects.
package artifactbus

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
)

const readTTL = 15 * time.Minute

var (
	// ErrNotFound is returned when the caller cannot read that file.
	ErrNotFound = errors.New("artifact not found")
)

// Storer persists artifacts.
type Storer interface {
	Upsert(ctx context.Context, art Artifact) error
	QueryByFileID(ctx context.Context, fileID uuid.UUID) (Artifact, error)
	QueryBySourceRef(ctx context.Context, sourceRef string) (Artifact, error)
	QueryBySourceRefs(ctx context.Context, sourceRefs []string) ([]Artifact, error)
}

// Presigner signs a short-lived read of one object. It does not fetch the bytes.
type Presigner interface {
	PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (url string, expiresAt time.Time, err error)
}

// ExtBusiness lists every public method an extension can wrap.
type ExtBusiness interface {
	Register(ctx context.Context, na NewArtifact) (Artifact, error)
	ReadURL(ctx context.Context, ownerID, fileID uuid.UUID) (Playback, error)
	QueryByFileID(ctx context.Context, fileID uuid.UUID) (Artifact, error)
	QueryBySourceRef(ctx context.Context, sourceRef string) (Artifact, error)
	QueryBySourceRefs(ctx context.Context, sourceRefs []string) ([]Artifact, error)
}

// Extension wraps a new layer of business logic around the existing logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages readable stored objects.
type Business struct {
	log       *logger.Logger
	storer    Storer
	presigner Presigner
}

// NewBusiness constructs an artifact business API for use.
func NewBusiness(log *logger.Logger, storer Storer, presigner Presigner, extensions ...Extension) ExtBusiness {
	b := ExtBusiness(&Business{
		log:       log,
		storer:    storer,
		presigner: presigner,
	})

	for _, ext := range slices.Backward(extensions) {
		if ext != nil {
			b = ext(b)
		}
	}

	return b
}

// Register records an object that is already in the bucket.
func (b *Business) Register(ctx context.Context, na NewArtifact) (Artifact, error) {
	now := time.Now()
	b.log.Info(ctx, "artifact.register", "file_id", na.FileID.String(), "kind", na.Kind.String())
	art := Artifact{
		FileID:      na.FileID,
		OwnerID:     na.OwnerID,
		Bucket:      na.Bucket,
		ObjectKey:   na.ObjectKey,
		SourceRef:   na.SourceRef,
		Kind:        na.Kind,
		ContentType: na.ContentType,
		Status:      na.Status,
		DateCreated: now,
		DateUpdated: now,
	}

	if err := b.storer.Upsert(ctx, art); err != nil {
		return Artifact{}, fmt.Errorf("upsert: %w", err)
	}

	return art, nil
}

// ReadURL checks that the caller owns a ready file and signs a short-lived read URL.
func (b *Business) ReadURL(ctx context.Context, ownerID, fileID uuid.UUID) (Playback, error) {
	art, err := b.storer.QueryByFileID(ctx, fileID)
	if err != nil {
		return Playback{}, err
	}

	ready := art.Status.Equal(MustParseStatus(StatusReady))
	switch {
	case art.OwnerID != ownerID, !ready:
		return Playback{}, ErrNotFound
	}

	url, expiresAt, err := b.presigner.PresignGet(ctx, art.Bucket, art.ObjectKey, readTTL)
	if err != nil {
		return Playback{}, fmt.Errorf("presign: %w", err)
	}

	return Playback{URL: url, ExpiresAt: expiresAt}, nil
}

// QueryByFileID returns the artifact for a file id.
func (b *Business) QueryByFileID(ctx context.Context, fileID uuid.UUID) (Artifact, error) {
	art, err := b.storer.QueryByFileID(ctx, fileID)
	if err != nil {
		return Artifact{}, fmt.Errorf("query by file id: %w", err)
	}
	return art, nil
}

// QueryBySourceRef returns the artifact stored at a source ref.
func (b *Business) QueryBySourceRef(ctx context.Context, sourceRef string) (Artifact, error) {
	art, err := b.storer.QueryBySourceRef(ctx, sourceRef)
	if err != nil {
		return Artifact{}, fmt.Errorf("query by source ref: %w", err)
	}
	return art, nil
}

// QueryBySourceRefs returns the artifacts stored at those source refs.
func (b *Business) QueryBySourceRefs(ctx context.Context, sourceRefs []string) ([]Artifact, error) {
	arts, err := b.storer.QueryBySourceRefs(ctx, sourceRefs)
	if err != nil {
		return nil, fmt.Errorf("query by source refs: %w", err)
	}
	return arts, nil
}

// FileIDFromObjectKey reads the file id from a source object key.
// The key is sources/{owner}/{fileID}.
func FileIDFromObjectKey(objectKey string) (uuid.UUID, error) {
	parts := strings.Split(objectKey, "/")
	if len(parts) < 3 || parts[0] != "sources" {
		return uuid.Nil, fmt.Errorf("unexpected object key %q", objectKey)
	}

	id, err := uuid.Parse(parts[2])
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse file id: %w", err)
	}

	return id, nil
}
