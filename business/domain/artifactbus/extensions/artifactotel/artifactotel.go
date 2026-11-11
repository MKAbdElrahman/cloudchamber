// Package artifactotel provides an extension for artifactbus that adds
// otel tracking.
package artifactotel

import (
	"context"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Extension provides a wrapper for otel functionality around the artifact bus.
type Extension struct {
	bus artifactbus.ExtBusiness
}

// NewExtension constructs a new extension that wraps the artifact bus with otel.
func NewExtension() artifactbus.Extension {
	return func(bus artifactbus.ExtBusiness) artifactbus.ExtBusiness {
		return &Extension{bus: bus}
	}
}

// Register records the span for storing a ready object.
func (ext *Extension) Register(ctx context.Context, na artifactbus.NewArtifact) (artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.register",
		attribute.String("file.id", na.FileID.String()),
		attribute.String("enduser.id", na.OwnerID.String()),
		attribute.String("artifact.kind", na.Kind.String()),
		attribute.String("object.bucket", na.Bucket),
		attribute.String("object.key", na.ObjectKey),
	)
	defer span.End()

	art, err := ext.bus.Register(ctx, na)
	finish(span, err)
	if err != nil {
		return artifactbus.Artifact{}, err
	}

	return art, nil
}

// ReadURL records the ownership check and the short-lived URL decision.
func (ext *Extension) ReadURL(ctx context.Context, ownerID, fileID uuid.UUID) (artifactbus.Playback, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.readurl",
		attribute.String("enduser.id", ownerID.String()),
		attribute.String("file.id", fileID.String()),
	)
	defer span.End()

	playback, err := ext.bus.ReadURL(ctx, ownerID, fileID)
	if err != nil {
		finish(span, err)
		return artifactbus.Playback{}, err
	}

	span.SetAttributes(attribute.String("playback.expires_at", playback.ExpiresAt.UTC().Format(time.RFC3339)))
	finish(span, nil)
	return playback, nil
}

// QueryByFileID records a lookup by file id.
func (ext *Extension) QueryByFileID(ctx context.Context, fileID uuid.UUID) (artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.querybyfileid",
		attribute.String("file.id", fileID.String()),
	)
	defer span.End()

	art, err := ext.bus.QueryByFileID(ctx, fileID)
	finish(span, err)
	if err != nil {
		return artifactbus.Artifact{}, err
	}

	return art, nil
}

// QueryBySourceRef records a lookup by internal source ref.
func (ext *Extension) QueryBySourceRef(ctx context.Context, sourceRef string) (artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.querybysourceref")
	defer span.End()

	art, err := ext.bus.QueryBySourceRef(ctx, sourceRef)
	finish(span, err)
	if err != nil {
		return artifactbus.Artifact{}, err
	}

	return art, nil
}

// QueryBySourceRefs records a lookup of several internal source refs.
func (ext *Extension) QueryBySourceRefs(ctx context.Context, sourceRefs []string) ([]artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.querybysourcerefs",
		attribute.Int("artifact.count", len(sourceRefs)),
	)
	defer span.End()

	arts, err := ext.bus.QueryBySourceRefs(ctx, sourceRefs)
	finish(span, err)
	if err != nil {
		return nil, err
	}

	return arts, nil
}

func finish(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetStatus(codes.Ok, "")
}
