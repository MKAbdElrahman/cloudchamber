// Package artifactdb contains PostgreSQL database access for artifacts.
package artifactdb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Store manages database access for artifacts.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs an artifact store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{log: log, db: db}
}

type artifactRow struct {
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

func toDBArtifact(bus artifactbus.Artifact) artifactRow {
	return artifactRow{
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

func toBusArtifact(row artifactRow) (artifactbus.Artifact, error) {
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

func toBusArtifacts(rows []artifactRow) ([]artifactbus.Artifact, error) {
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

const artifactColumns = `
	file_id, owner_id, bucket, object_key, source_ref, kind, content_type, status, date_created, date_updated`

// Upsert inserts or replaces an artifact by file id.
func (s *Store) Upsert(ctx context.Context, art artifactbus.Artifact) error {
	const q = `
	INSERT INTO artifacts
		(file_id, owner_id, bucket, object_key, source_ref, kind, content_type, status, date_created, date_updated)
	VALUES
		(:file_id, :owner_id, :bucket, :object_key, :source_ref, :kind, :content_type, :status, :date_created, :date_updated)
	ON CONFLICT (file_id) DO UPDATE SET
		owner_id = EXCLUDED.owner_id,
		bucket = EXCLUDED.bucket,
		object_key = EXCLUDED.object_key,
		source_ref = EXCLUDED.source_ref,
		kind = EXCLUDED.kind,
		content_type = EXCLUDED.content_type,
		status = EXCLUDED.status,
		date_updated = EXCLUDED.date_updated`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBArtifact(art)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// QueryByFileID retrieves an artifact by file id.
func (s *Store) QueryByFileID(ctx context.Context, fileID uuid.UUID) (artifactbus.Artifact, error) {
	data := struct {
		FileID uuid.UUID `db:"file_id"`
	}{
		FileID: fileID,
	}

	q := `SELECT ` + artifactColumns + ` FROM artifacts WHERE file_id = :file_id`

	var row artifactRow
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &row); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return artifactbus.Artifact{}, fmt.Errorf("db: %w", artifactbus.ErrNotFound)
		}
		return artifactbus.Artifact{}, fmt.Errorf("db: %w", err)
	}

	return toBusArtifact(row)
}

// QueryBySourceRef retrieves an artifact by its internal source ref.
func (s *Store) QueryBySourceRef(ctx context.Context, sourceRef string) (artifactbus.Artifact, error) {
	data := struct {
		SourceRef string `db:"source_ref"`
	}{
		SourceRef: sourceRef,
	}

	q := `SELECT ` + artifactColumns + ` FROM artifacts WHERE source_ref = :source_ref`

	var row artifactRow
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &row); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return artifactbus.Artifact{}, fmt.Errorf("db: %w", artifactbus.ErrNotFound)
		}
		return artifactbus.Artifact{}, fmt.Errorf("db: %w", err)
	}

	return toBusArtifact(row)
}

// QueryBySourceRefs retrieves artifacts for the given source refs.
func (s *Store) QueryBySourceRefs(ctx context.Context, sourceRefs []string) ([]artifactbus.Artifact, error) {
	if len(sourceRefs) == 0 {
		return nil, nil
	}

	data := map[string]any{
		"source_refs": sourceRefs,
	}
	q := `SELECT ` + artifactColumns + ` FROM artifacts WHERE source_ref IN (:source_refs)`

	var rows []artifactRow
	if err := sqldb.NamedQuerySliceUsingIn(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	return toBusArtifacts(rows)
}
