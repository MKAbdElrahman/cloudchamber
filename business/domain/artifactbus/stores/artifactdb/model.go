package artifactdb

import (
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/google/uuid"
)

type artifactDB struct {
	FileID      uuid.UUID `db:"file_id"`
	OwnerID     uuid.UUID `db:"owner_id"`
	Bucket      string    `db:"bucket"`
	ObjectKey   string    `db:"object_key"`
	SourceRef   string    `db:"source_ref"`
	Kind        string    `db:"kind"`
	ContentType string    `db:"content_type"`
	Status      string    `db:"status"`
	DateCreated time.Time `db:"date_created"`
	DateUpdated time.Time `db:"date_updated"`
}

func toDBArtifact(bus artifactbus.Artifact) artifactDB {
	return artifactDB{
		FileID:      bus.FileID,
		OwnerID:     bus.OwnerID,
		Bucket:      bus.Bucket,
		ObjectKey:   bus.ObjectKey,
		SourceRef:   bus.SourceRef,
		Kind:        bus.Kind.String(),
		ContentType: bus.ContentType,
		Status:      bus.Status.String(),
		DateCreated: bus.DateCreated.UTC(),
		DateUpdated: bus.DateUpdated.UTC(),
	}
}

func toBusArtifact(row artifactDB) (artifactbus.Artifact, error) {
	kind, err := artifactbus.ParseKind(row.Kind)
	if err != nil {
		return artifactbus.Artifact{}, fmt.Errorf("parse kind: %w", err)
	}
	status, err := artifactbus.ParseStatus(row.Status)
	if err != nil {
		return artifactbus.Artifact{}, fmt.Errorf("parse status: %w", err)
	}

	return artifactbus.Artifact{
		FileID:      row.FileID,
		OwnerID:     row.OwnerID,
		Bucket:      row.Bucket,
		ObjectKey:   row.ObjectKey,
		SourceRef:   row.SourceRef,
		Kind:        kind,
		ContentType: row.ContentType,
		Status:      status,
		DateCreated: row.DateCreated.In(time.Local),
		DateUpdated: row.DateUpdated.In(time.Local),
	}, nil
}

func toBusArtifacts(rows []artifactDB) ([]artifactbus.Artifact, error) {
	arts := make([]artifactbus.Artifact, len(rows))
	for i, row := range rows {
		art, err := toBusArtifact(row)
		if err != nil {
			return nil, err
		}
		arts[i] = art
	}
	return arts, nil
}
