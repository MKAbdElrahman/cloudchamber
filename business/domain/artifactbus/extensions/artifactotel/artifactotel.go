// Package artifactotel provides an extension for artifactbus that adds
// otel tracking.
package artifactotel

import (
	"context"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
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

// NewWithTx constructs a new extension around the transaction-bound business.
func (ext *Extension) NewWithTx(tx sqldb.CommitRollbacker) (artifactbus.ExtBusiness, error) {
	bus, err := ext.bus.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	next := Extension{
		bus: bus,
	}

	return &next, nil
}

// Register applies otel to recording a ready object.
func (ext *Extension) Register(ctx context.Context, na artifactbus.NewArtifact) (artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.register")
	defer span.End()

	art, err := ext.bus.Register(ctx, na)
	if err != nil {
		return artifactbus.Artifact{}, err
	}

	return art, nil
}

// ReadURL applies otel to the short-lived read URL.
func (ext *Extension) ReadURL(ctx context.Context, ownerID, fileID uuid.UUID) (artifactbus.Playback, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.readurl")
	defer span.End()

	playback, err := ext.bus.ReadURL(ctx, ownerID, fileID)
	if err != nil {
		return artifactbus.Playback{}, err
	}

	return playback, nil
}

// QueryByFileID applies otel to a lookup by file id.
func (ext *Extension) QueryByFileID(ctx context.Context, fileID uuid.UUID) (artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.querybyfileid")
	defer span.End()

	return ext.bus.QueryByFileID(ctx, fileID)
}

// QueryByFileIDs applies otel to a lookup of several file ids.
func (ext *Extension) QueryByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]artifactbus.Artifact, error) {
	ctx, span := otel.AddSpan(ctx, "business.artifactbus.querybyfileids")
	defer span.End()

	return ext.bus.QueryByFileIDs(ctx, fileIDs)
}
