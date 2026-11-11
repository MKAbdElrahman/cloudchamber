// Package artifactdb contains PostgreSQL database access for artifacts.
package artifactdb

import (
	"context"
	"errors"
	"fmt"

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

// NewWithTx constructs a new Store value replacing the sqlx DB
// value with a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (artifactbus.Storer, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log: s.log,
		db:  ec,
	}

	return &store, nil
}

const artifactColumns = `
	file_id, owner_id, project_id, bucket, object_key, kind, content_type, name, status, date_created, date_updated`

// Upsert inserts or replaces an artifact by file id.
func (s *Store) Upsert(ctx context.Context, art artifactbus.Artifact) error {
	const q = `
	INSERT INTO artifacts
		(file_id, owner_id, project_id, bucket, object_key, kind, content_type, name, status, date_created, date_updated)
	VALUES
		(:file_id, :owner_id, :project_id, :bucket, :object_key, :kind, :content_type, :name, :status, :date_created, :date_updated)
	ON CONFLICT (file_id) DO UPDATE SET
		owner_id = EXCLUDED.owner_id,
		project_id = COALESCE(EXCLUDED.project_id, artifacts.project_id),
		bucket = EXCLUDED.bucket,
		object_key = EXCLUDED.object_key,
		kind = EXCLUDED.kind,
		content_type = EXCLUDED.content_type,
		name = EXCLUDED.name,
		status = EXCLUDED.status,
		date_updated = EXCLUDED.date_updated`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBArtifact(art)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes an artifact row.
func (s *Store) Delete(ctx context.Context, fileID uuid.UUID) error {
	data := struct {
		FileID uuid.UUID `db:"file_id"`
	}{
		FileID: fileID,
	}
	const q = `DELETE FROM artifacts WHERE file_id = :file_id`
	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}

// QueryByProject retrieves the files in one project, newest first.
func (s *Store) QueryByProject(ctx context.Context, ownerID, projectID uuid.UUID) ([]artifactbus.Artifact, error) {
	data := struct {
		OwnerID   uuid.UUID `db:"owner_id"`
		ProjectID uuid.UUID `db:"project_id"`
	}{
		OwnerID:   ownerID,
		ProjectID: projectID,
	}
	q := `SELECT ` + artifactColumns + `
		FROM artifacts
		WHERE owner_id = :owner_id AND project_id = :project_id
		ORDER BY date_created DESC`

	var rows []artifactDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	return toBusArtifacts(rows)
}

// QueryByFileID retrieves an artifact by file id.
func (s *Store) QueryByFileID(ctx context.Context, fileID uuid.UUID) (artifactbus.Artifact, error) {
	data := struct {
		FileID uuid.UUID `db:"file_id"`
	}{
		FileID: fileID,
	}

	q := `SELECT ` + artifactColumns + ` FROM artifacts WHERE file_id = :file_id`

	var row artifactDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &row); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return artifactbus.Artifact{}, fmt.Errorf("db: %w", artifactbus.ErrNotFound)
		}
		return artifactbus.Artifact{}, fmt.Errorf("db: %w", err)
	}

	return toBusArtifact(row)
}

// QueryByFileIDs retrieves artifacts for the given file ids.
func (s *Store) QueryByFileIDs(ctx context.Context, fileIDs []uuid.UUID) ([]artifactbus.Artifact, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	data := map[string]any{
		"file_ids": fileIDs,
	}
	q := `SELECT ` + artifactColumns + ` FROM artifacts WHERE file_id IN (:file_ids)`

	var rows []artifactDB
	if err := sqldb.NamedQuerySliceUsingIn(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	return toBusArtifacts(rows)
}
