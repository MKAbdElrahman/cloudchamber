package media2docapp

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// files is the file store the workers' business packages use. It joins two
// things those packages may not reach themselves: the artifact domain, which
// records files, and the object store, which holds their bytes. Joining
// domains is the app layer's job.
type files struct {
	artifacts artifactbus.ExtBusiness
	objects   *objectstore.Client
	tracer    trace.Tracer
}

// NewFiles constructs the file store for the workers' business packages.
// objects may be nil for a caller that only checks a recording and never
// reads or writes bytes, such as the API.
func NewFiles(artifacts artifactbus.ExtBusiness, objects *objectstore.Client, tracer trace.Tracer) media2docbus.Files {
	return &files{artifacts: artifacts, objects: objects, tracer: tracer}
}

// traced stores the tracer the way the HTTP middleware does, so business
// spans started with otel.AddSpan join the caller's trace.
func (f *files) traced(ctx context.Context) context.Context {
	if f.tracer == nil {
		return ctx
	}
	return otel.InjectTracing(ctx, f.tracer)
}

// QuerySource returns the recording, or media2docbus.ErrSourceNotFound.
func (f *files) QuerySource(ctx context.Context, fileID uuid.UUID) (media2docbus.Source, error) {
	art, err := f.artifact(ctx, fileID)
	if err != nil {
		return media2docbus.Source{}, err
	}
	return sourceFromArtifact(art), nil
}

// SourceSize returns the size of the recording's bytes.
func (f *files) SourceSize(ctx context.Context, fileID uuid.UUID) (int64, error) {
	art, err := f.artifact(ctx, fileID)
	if err != nil {
		return 0, err
	}
	info, err := f.objects.HeadObject(ctx, art.Bucket, art.ObjectKey)
	if err != nil {
		return 0, missing(err)
	}
	return info.Size, nil
}

// ReadSource writes the recording's bytes to dst.
func (f *files) ReadSource(ctx context.Context, fileID uuid.UUID, dst io.Writer) error {
	art, err := f.artifact(ctx, fileID)
	if err != nil {
		return err
	}
	return missing(f.objects.GetObject(ctx, art.Bucket, art.ObjectKey, dst))
}

// SaveOutput stores the bytes and records them as a ready file in the output's project.
func (f *files) SaveOutput(ctx context.Context, fileID uuid.UUID, out media2docbus.Output) error {
	kind, err := artifactbus.ParseKind(out.Kind)
	if err != nil {
		return fmt.Errorf("output kind: %w", err)
	}

	ctx = f.traced(ctx)
	key := fmt.Sprintf("outputs/%s/%s", out.OwnerID, fileID)

	if err := f.objects.PutObject(ctx, "", key, out.ContentType, out.Body); err != nil {
		return fmt.Errorf("store %s: %w", out.Kind, err)
	}
	// A step that was told to stop does not record a file it may not have finished.
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err = f.artifacts.Register(ctx, artifactbus.NewArtifact{
		FileID:      fileID,
		OwnerID:     out.OwnerID,
		ProjectID:   out.ProjectID,
		Bucket:      f.objects.Bucket(),
		ObjectKey:   key,
		Kind:        kind,
		ContentType: out.ContentType,
		Name:        out.Name,
		Status:      artifactbus.MustParseStatus(artifactbus.StatusReady),
	})
	if err != nil {
		return fmt.Errorf("register %s: %w", out.Kind, err)
	}
	return nil
}

func (f *files) artifact(ctx context.Context, fileID uuid.UUID) (artifactbus.Artifact, error) {
	art, err := f.artifacts.QueryByFileID(f.traced(ctx), fileID)
	if err != nil {
		if errors.Is(err, artifactbus.ErrNotFound) {
			return artifactbus.Artifact{}, media2docbus.ErrSourceNotFound
		}
		return artifactbus.Artifact{}, fmt.Errorf("load file: %w", err)
	}
	return art, nil
}

// missing names an object that is not in storage in the workers' terms.
func missing(err error) error {
	if err != nil && objectstore.IsMissing(err) {
		return fmt.Errorf("%w: %w", media2docbus.ErrSourceMissing, err)
	}
	return err
}

// sourceFromArtifact converts a file of the artifact domain into the workers' recording.
func sourceFromArtifact(art artifactbus.Artifact) media2docbus.Source {
	return media2docbus.Source{
		FileID:      art.FileID,
		OwnerID:     art.OwnerID,
		ProjectID:   art.ProjectID,
		Name:        art.Name,
		ContentType: art.ContentType,
		Ready:       art.Status.Equal(artifactbus.MustParseStatus(artifactbus.StatusReady)),
	}
}
