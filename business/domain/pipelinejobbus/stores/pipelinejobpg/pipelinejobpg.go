// Package pipelinejobpg contains PostgreSQL database access for pipeline jobs.
package pipelinejobpg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus/stores/commondb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/messagedb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb/dbarray"
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

// Begin starts a transaction on the underlying database.
func (s *Store) Begin(ctx context.Context) (sqldb.CommitRollbacker, error) {
	db, ok := s.db.(*sqlx.DB)
	if !ok {
		return nil, fmt.Errorf("db: transaction already started")
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	return tx, nil
}

// InTx reports whether this store is already bound to a transaction.
func (s *Store) InTx() bool {
	_, ok := s.db.(*sqlx.Tx)
	return ok
}

// Create inserts a new pipeline job.
func (s *Store) Create(ctx context.Context, job pipelinejobbus.Job) error {
	const q = `
	INSERT INTO pipeline_jobs
		(job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, failure_message,
		 audio_file_id, transcript_file_id, key_frames_file_id, document_file_id, date_created, date_updated)
	VALUES
		(:job_id, :owner_id, :project_id, :workflow_id, :run_id, :input_file_id, :idempotency_key_hash, :status, :failure_message,
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
		owner_id = :owner_id,
		project_id = :project_id,
		workflow_id = :workflow_id,
		run_id = :run_id,
		input_file_id = :input_file_id,
		idempotency_key_hash = :idempotency_key_hash,
		status = :status,
		failure_message = :failure_message,
		date_created = :date_created,
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
		job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, failure_message,
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
		job_id, owner_id, project_id, workflow_id, run_id, input_file_id, idempotency_key_hash, status, failure_message,
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

// AppendFact writes one fact to the job stream and returns it with its positions.
func (s *Store) AppendFact(ctx context.Context, job pipelinejobbus.Job, fact pipelinejobbus.Fact) (pipelinejobbus.Fact, error) {
	if fact.ID == uuid.Nil {
		fact.ID = uuid.New()
	}
	return s.writeFact(ctx, job, fact, false)
}

// AppendStableFact writes one fact under a caller-chosen id.
// Writing that id again returns the fact already stored.
func (s *Store) AppendStableFact(ctx context.Context, job pipelinejobbus.Job, fact pipelinejobbus.Fact) (pipelinejobbus.Fact, error) {
	if fact.ID == uuid.Nil {
		return pipelinejobbus.Fact{}, fmt.Errorf("db: fact id is required")
	}
	return s.writeFact(ctx, job, fact, true)
}

func (s *Store) writeFact(ctx context.Context, job pipelinejobbus.Job, fact pipelinejobbus.Fact, stable bool) (pipelinejobbus.Fact, error) {
	requested := fact.Time
	fact.Time = time.Time{}
	data, meta, err := commondb.ToDBFact(fact)
	if err != nil {
		return pipelinejobbus.Fact{}, fmt.Errorf("db: %w", err)
	}

	var written messagedb.Written
	if stable {
		written, err = messagedb.WriteStable(ctx, s.db, fact.ID, pipelinejobbus.StreamName(job.ID), pipelinejobbus.FactType, data, meta)
	} else {
		written, err = messagedb.Write(ctx, s.db, fact.ID, pipelinejobbus.StreamName(job.ID), pipelinejobbus.FactType, data, meta)
	}
	if err != nil {
		return pipelinejobbus.Fact{}, fmt.Errorf("db: %w", err)
	}
	if !requested.IsZero() {
		if _, err := s.db.ExecContext(ctx, `UPDATE message_store.messages SET time = $1 AT TIME ZONE 'utc' WHERE id = $2::uuid`, requested.UTC(), written.ID.String()); err != nil {
			return pipelinejobbus.Fact{}, fmt.Errorf("db: %w", err)
		}
		written.Time = requested.UTC()
	}
	return applyWritten(fact, written), nil
}

func applyWritten(fact pipelinejobbus.Fact, written messagedb.Written) pipelinejobbus.Fact {
	fact.ID = written.ID
	fact.Position = written.Position
	fact.GlobalPosition = written.GlobalPosition
	fact.Time = written.Time
	return fact
}

// AppendProgress writes one progress update. It does not change the job row.
func (s *Store) AppendProgress(ctx context.Context, job pipelinejobbus.Job, progress pipelinejobbus.Progress) (pipelinejobbus.Progress, error) {
	if progress.ID == uuid.Nil {
		progress.ID = uuid.New()
	}
	data, meta, err := commondb.ToDBProgress(progress)
	if err != nil {
		return pipelinejobbus.Progress{}, fmt.Errorf("db: %w", err)
	}
	written, err := messagedb.Write(ctx, s.db, progress.ID, pipelinejobbus.StreamName(job.ID), pipelinejobbus.ProgressType, data, meta)
	if err != nil {
		return pipelinejobbus.Progress{}, fmt.Errorf("db: %w", err)
	}
	progress.ID = written.ID
	progress.Position = written.Position
	progress.GlobalPosition = written.GlobalPosition
	progress.Time = written.Time
	return progress, nil
}

// StreamFacts returns facts of one job after a stream position.
func (s *Store) StreamFacts(ctx context.Context, jobID uuid.UUID, after int64, limit int) ([]pipelinejobbus.Fact, error) {
	if limit <= 0 {
		limit = 1000
	}
	return s.facts(ctx, `
		SELECT `+messagedb.MessageColumns+`
		FROM message_store.messages
		WHERE stream_name = $1 AND position > $2 AND type = $3
		ORDER BY position
		LIMIT $4`, pipelinejobbus.StreamName(jobID), after, pipelinejobbus.FactType, limit)
}

// StreamLog returns business events and progress updates after a stream position.
func (s *Store) StreamLog(ctx context.Context, jobID uuid.UUID, after int64, limit int) (pipelinejobbus.Log, error) {
	if limit <= 0 {
		limit = 1000
	}
	return s.readLog(ctx, `
		SELECT `+messagedb.MessageColumns+`
		FROM message_store.messages
		WHERE stream_name = $1 AND position > $2 AND type IN ($3, $4)
		ORDER BY position
		LIMIT $5`, pipelinejobbus.StreamName(jobID), after, pipelinejobbus.FactType, pipelinejobbus.ProgressType, limit)
}

// ProjectFacts returns facts of one project after a global position.
// The job row owns the owner and the project. The stream name is job-<job id>.
func (s *Store) ProjectFacts(ctx context.Context, filter pipelinejobbus.QueryFilter, after int64, limit int) ([]pipelinejobbus.Fact, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.facts(ctx, `
		SELECT `+messagedb.MessageColumns+`
		FROM message_store.messages
		JOIN pipeline_jobs AS job
		  ON job.job_id = `+jobIDSQL()+`
		WHERE message_store.category(stream_name) = $1
		  AND type = $2
		  AND global_position > $3
		  AND job.project_id = $4::uuid
		  AND ($5 = '' OR job.owner_id = NULLIF($5, '')::uuid)
		ORDER BY global_position
		LIMIT $6`, pipelinejobbus.StreamCategory, pipelinejobbus.FactType, after, filter.ProjectID.String(), ownerText(filter.OwnerID), limit)
}

// ProjectLog returns business events and progress updates after a global position.
func (s *Store) ProjectLog(ctx context.Context, filter pipelinejobbus.QueryFilter, after int64, limit int) (pipelinejobbus.Log, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.readLog(ctx, `
		SELECT `+messagedb.MessageColumns+`
		FROM message_store.messages
		JOIN pipeline_jobs AS job
		  ON job.job_id = `+jobIDSQL()+`
		WHERE message_store.category(stream_name) = $1
		  AND type IN ($2, $3)
		  AND global_position > $4
		  AND job.project_id = $5::uuid
		  AND ($6 = '' OR job.owner_id = NULLIF($6, '')::uuid)
		ORDER BY global_position
		LIMIT $7`, pipelinejobbus.StreamCategory, pipelinejobbus.FactType, pipelinejobbus.ProgressType, after, filter.ProjectID.String(), ownerText(filter.OwnerID), limit)
}

// ProjectCursor returns the newest global position for a project.
func (s *Store) ProjectCursor(ctx context.Context, filter pipelinejobbus.QueryFilter) (int64, error) {
	var cursor int64
	err := sqlx.GetContext(ctx, s.db, &cursor, `
		SELECT COALESCE(MAX(global_position), 0)
		FROM message_store.messages
		JOIN pipeline_jobs AS job
		  ON job.job_id = `+jobIDSQL()+`
		WHERE message_store.category(stream_name) = $1
		  AND type IN ($2, $3)
		  AND job.project_id = $4::uuid
		  AND ($5 = '' OR job.owner_id = NULLIF($5, '')::uuid)`,
		pipelinejobbus.StreamCategory, pipelinejobbus.FactType, pipelinejobbus.ProgressType, filter.ProjectID.String(), ownerText(filter.OwnerID))
	if err != nil {
		return 0, fmt.Errorf("db: %w", err)
	}
	return cursor, nil
}

// HeadFacts returns the latest copy of each sentence for the given jobs.
func (s *Store) HeadFacts(ctx context.Context, jobIDs []uuid.UUID) ([]pipelinejobbus.Fact, error) {
	if len(jobIDs) == 0 {
		return nil, nil
	}
	streams := make([]string, len(jobIDs))
	for i, id := range jobIDs {
		streams[i] = pipelinejobbus.StreamName(id)
	}
	return s.facts(ctx, `
		SELECT DISTINCT ON (stream_name, COALESCE(data->>'fact', ''))
			`+messagedb.MessageColumns+`
		FROM message_store.messages
		WHERE stream_name = ANY($1) AND type = $2
		ORDER BY stream_name, COALESCE(data->>'fact', ''), position DESC`, dbarray.Array(streams), pipelinejobbus.FactType)
}

// HeadProgress returns the latest progress update of each stage for the given jobs.
func (s *Store) HeadProgress(ctx context.Context, jobIDs []uuid.UUID) ([]pipelinejobbus.Progress, error) {
	if len(jobIDs) == 0 {
		return nil, nil
	}
	streams := make([]string, len(jobIDs))
	for i, id := range jobIDs {
		streams[i] = pipelinejobbus.StreamName(id)
	}
	return s.progressOnly(ctx, `
		SELECT DISTINCT ON (stream_name, COALESCE(data->>'stage', ''))
			`+messagedb.MessageColumns+`
		FROM message_store.messages
		WHERE stream_name = ANY($1) AND type = $2
		ORDER BY stream_name, COALESCE(data->>'stage', ''), position DESC`, dbarray.Array(streams), pipelinejobbus.ProgressType)
}

func (s *Store) facts(ctx context.Context, query string, args ...any) ([]pipelinejobbus.Fact, error) {
	msgs, err := messagedb.Select(ctx, s.db, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	facts, err := commondb.ToBusFacts(msgs)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	return facts, nil
}

func (s *Store) readLog(ctx context.Context, query string, args ...any) (pipelinejobbus.Log, error) {
	msgs, err := messagedb.Select(ctx, s.db, query, args...)
	if err != nil {
		return pipelinejobbus.Log{}, fmt.Errorf("db: %w", err)
	}
	log, err := commondb.ToBusLog(msgs)
	if err != nil {
		return pipelinejobbus.Log{}, fmt.Errorf("db: %w", err)
	}
	return log, nil
}

func (s *Store) progressOnly(ctx context.Context, query string, args ...any) ([]pipelinejobbus.Progress, error) {
	msgs, err := messagedb.Select(ctx, s.db, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	out := make([]pipelinejobbus.Progress, 0, len(msgs))
	for _, msg := range msgs {
		update, err := commondb.ToBusProgress(msg)
		if err != nil {
			return nil, fmt.Errorf("db: %w", err)
		}
		out = append(out, update)
	}
	return out, nil
}

func ownerText(owner *uuid.UUID) string {
	if owner == nil {
		return ""
	}
	return owner.String()
}

func jobIDSQL() string {
	return fmt.Sprintf("substring(stream_name from %d)::uuid", pipelinejobbus.StreamIDOffset())
}
