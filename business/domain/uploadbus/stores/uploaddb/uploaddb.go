// Package uploaddb contains PostgreSQL database access for uploads.
package uploaddb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus/stores/commondb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb/dialect"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/jmoiron/sqlx"
)

// Store manages database access for uploads.
type Store struct {
	log     *logger.Logger
	db      sqlx.ExtContext
	dialect dialect.Dialect
}

// NewStore constructs an upload store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log:     log,
		db:      db,
		dialect: dialect.Postgres{},
	}
}

// NewWithTx constructs a new Store value replacing the sqlx DB value with
// a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (uploadbus.Storer, error) {
	ec, err := sqldb.GetExtContext(tx)
	if err != nil {
		return nil, err
	}

	store := Store{
		log:     s.log,
		db:      ec,
		dialect: s.dialect,
	}

	return &store, nil
}

// Create inserts a new upload.
func (s *Store) Create(ctx context.Context, up uploadbus.Upload) error {
	const q = `
	INSERT INTO uploads
		(object_key, owner_id, bucket, object_key_ref, size_bytes, etag, status, tusd_upload_id, date_created, date_updated)
	VALUES
		(:object_key, :owner_id, :bucket, :object_key_ref, :size_bytes, :etag, :status, :tusd_upload_id, :date_created, :date_updated)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, commondb.ToDBUpload(up)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update modifies an existing upload.
func (s *Store) Update(ctx context.Context, up uploadbus.Upload) error {
	const q = `
	UPDATE uploads
	SET
		owner_id = :owner_id,
		bucket = :bucket,
		object_key_ref = :object_key_ref,
		size_bytes = :size_bytes,
		etag = :etag,
		status = :status,
		tusd_upload_id = :tusd_upload_id,
		date_updated = :date_updated
	WHERE
		object_key = :object_key`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, commondb.ToDBUpload(up)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Upsert inserts or updates an upload by object key (idempotent).
func (s *Store) Upsert(ctx context.Context, up uploadbus.Upload) error {
	const q = `
	INSERT INTO uploads
		(object_key, owner_id, bucket, object_key_ref, size_bytes, etag, status, tusd_upload_id, date_created, date_updated)
	VALUES
		(:object_key, :owner_id, :bucket, :object_key_ref, :size_bytes, :etag, :status, :tusd_upload_id, :date_created, :date_updated)
	ON CONFLICT (object_key) DO UPDATE SET
		owner_id = EXCLUDED.owner_id,
		bucket = EXCLUDED.bucket,
		object_key_ref = EXCLUDED.object_key_ref,
		size_bytes = EXCLUDED.size_bytes,
		etag = EXCLUDED.etag,
		status = EXCLUDED.status,
		tusd_upload_id = EXCLUDED.tusd_upload_id,
		date_updated = EXCLUDED.date_updated`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, commondb.ToDBUpload(up)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// MarkTerminated marks an upload as terminated if it exists (idempotent).
func (s *Store) MarkTerminated(ctx context.Context, objectKey string, now time.Time) error {
	data := struct {
		ObjectKey string    `db:"object_key"`
		Status    string    `db:"status"`
		Updated   time.Time `db:"date_updated"`
	}{
		ObjectKey: objectKey,
		Status:    uploadbus.StatusTerminated,
		Updated:   now.UTC(),
	}

	const q = `
	UPDATE uploads
	SET
		status = :status,
		date_updated = :date_updated
	WHERE
		object_key = :object_key`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes an upload.
func (s *Store) Delete(ctx context.Context, up uploadbus.Upload) error {
	data := struct {
		ObjectKey string `db:"object_key"`
	}{
		ObjectKey: up.ObjectKey,
	}

	const q = `DELETE FROM uploads WHERE object_key = :object_key`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query retrieves uploads matching the filter.
func (s *Store) Query(ctx context.Context, filter uploadbus.QueryFilter, orderBy order.By, pg page.Page) ([]uploadbus.Upload, error) {
	data := map[string]any{
		"offset":        (pg.Number() - 1) * pg.RowsPerPage(),
		"rows_per_page": pg.RowsPerPage(),
	}

	const q = `
	SELECT
		object_key, owner_id, bucket, object_key_ref, size_bytes, etag, status, tusd_upload_id, date_created, date_updated
	FROM
		uploads`

	buf := bytes.NewBufferString(q)
	commondb.ApplyFilter(filter, data, buf)

	clause, err := commondb.OrderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(clause)
	s.dialect.Paginate(buf)

	var dbUps []commondb.Upload
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, buf.String(), data, &dbUps); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return commondb.ToBusUploads(dbUps)
}

// Count returns the number of uploads matching the filter.
func (s *Store) Count(ctx context.Context, filter uploadbus.QueryFilter) (int, error) {
	data := map[string]any{}

	const q = `SELECT count(1) AS count FROM uploads`

	buf := bytes.NewBufferString(q)
	commondb.ApplyFilter(filter, data, buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("db: %w", err)
	}

	return count.Count, nil
}

// QueryByObjectKey retrieves an upload by object key.
func (s *Store) QueryByObjectKey(ctx context.Context, objectKey string) (uploadbus.Upload, error) {
	data := struct {
		ObjectKey string `db:"object_key"`
	}{
		ObjectKey: objectKey,
	}

	const q = `
	SELECT
		object_key, owner_id, bucket, object_key_ref, size_bytes, etag, status, tusd_upload_id, date_created, date_updated
	FROM
		uploads
	WHERE
		object_key = :object_key`

	var dbUp commondb.Upload
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbUp); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return uploadbus.Upload{}, fmt.Errorf("db: %w", uploadbus.ErrNotFound)
		}
		return uploadbus.Upload{}, fmt.Errorf("db: %w", err)
	}

	return commondb.ToBusUpload(dbUp)
}
