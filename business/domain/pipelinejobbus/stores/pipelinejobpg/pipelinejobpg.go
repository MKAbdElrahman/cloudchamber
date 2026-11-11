// Package pipelinejobpg contains PostgreSQL database access for pipeline jobs.
package pipelinejobpg

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus/stores/commondb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb/dialect"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Store manages database access for pipeline jobs.
type Store struct {
	log     *logger.Logger
	db      sqlx.ExtContext
	dialect dialect.Dialect
}

// NewStore constructs a pipeline job store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log:     log,
		db:      db,
		dialect: dialect.Postgres{},
	}
}

// NewWithTx constructs a new Store value replacing the sqlx DB value with
// a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (pipelinejobbus.Storer, error) {
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

// Create inserts a new pipeline job.
func (s *Store) Create(ctx context.Context, job pipelinejobbus.Job) error {
	const q = `
	INSERT INTO pipeline_jobs
		(job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, error_message, stopped_step,
		 audio_file_id, transcript_file_id, key_frames_file_id, document_file_id, date_created, date_updated)
	VALUES
		(:job_id, :owner_id, :project_id, :workflow_id, :run_id, :input_file_id, :idempotency_key_hash, :status, :error_message, :stopped_step,
		 :audio_file_id, :transcript_file_id, :key_frames_file_id, :document_file_id, :date_created, :date_updated)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, commondb.ToDBJob(job)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update modifies an existing pipeline job.
func (s *Store) Update(ctx context.Context, job pipelinejobbus.Job) error {
	const q = `
	UPDATE pipeline_jobs
	SET
		project_id = :project_id,
		workflow_id = :workflow_id,
		run_id = :run_id,
		input_file_id = :input_file_id,
		idempotency_key_hash = :idempotency_key_hash,
		status = :status,
		error_message = :error_message,
		stopped_step = :stopped_step,
		audio_file_id = :audio_file_id,
		transcript_file_id = :transcript_file_id,
		key_frames_file_id = :key_frames_file_id,
		document_file_id = :document_file_id,
		date_updated = :date_updated
	WHERE
		job_id = :job_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, commondb.ToDBJob(job)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a pipeline job.
func (s *Store) Delete(ctx context.Context, job pipelinejobbus.Job) error {
	data := struct {
		ID string `db:"job_id"`
	}{
		ID: job.ID.String(),
	}

	const q = `DELETE FROM pipeline_jobs WHERE job_id = :job_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query retrieves pipeline jobs matching the filter.
func (s *Store) Query(ctx context.Context, filter pipelinejobbus.QueryFilter, orderBy order.By, pg page.Page) ([]pipelinejobbus.Job, error) {
	data := map[string]any{
		"offset":        (pg.Number() - 1) * pg.RowsPerPage(),
		"rows_per_page": pg.RowsPerPage(),
	}

	const q = `
	SELECT
		job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, error_message, stopped_step,
		audio_file_id, transcript_file_id, key_frames_file_id, document_file_id, date_created, date_updated
	FROM
		pipeline_jobs`

	buf := bytes.NewBufferString(q)
	commondb.ApplyFilter(filter, data, buf)

	clause, err := commondb.OrderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(clause)
	s.dialect.Paginate(buf)

	var dbJobs []commondb.Job
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, buf.String(), data, &dbJobs); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return commondb.ToBusJobs(dbJobs)
}

// Count returns the number of pipeline jobs matching the filter.
func (s *Store) Count(ctx context.Context, filter pipelinejobbus.QueryFilter) (int, error) {
	data := map[string]any{}

	const q = `SELECT count(1) AS count FROM pipeline_jobs`

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

// QueryByID retrieves a pipeline job by ID.
func (s *Store) QueryByID(ctx context.Context, jobID uuid.UUID) (pipelinejobbus.Job, error) {
	data := struct {
		ID string `db:"job_id"`
	}{
		ID: jobID.String(),
	}

	const q = `
	SELECT
		job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, error_message, stopped_step,
		audio_file_id, transcript_file_id, key_frames_file_id, document_file_id, date_created, date_updated
	FROM
		pipeline_jobs
	WHERE
		job_id = :job_id`

	var dbJob commondb.Job
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &dbJob); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return pipelinejobbus.Job{}, fmt.Errorf("db: %w", pipelinejobbus.ErrNotFound)
		}
		return pipelinejobbus.Job{}, fmt.Errorf("db: %w", err)
	}

	return commondb.ToBusJob(dbJob)
}
