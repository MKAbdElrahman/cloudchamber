// Package rundb contains the PostgreSQL implementation of the run store.
package rundb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb/dialect"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// PostgreSQL error codes for a broken unique constraint and a broken foreign key.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// columns lists the run columns every read selects. JSON columns are read as text.
const columns = `
	run_id, workflow, owner_id, project_id, idempotency_key_hash, generation,
	CAST(input AS text) AS input, suspend, cancel, attempt, resume_from,
	phase, observed_generation, observed_attempt, execution_id, reason, message,
	CAST(steps AS text) AS steps, CAST(result AS text) AS result,
	cancel_requested_at, date_closed, date_created, date_updated`

// Store manages run database access on PostgreSQL.
type Store struct {
	log     *logger.Logger
	db      sqlx.ExtContext
	dialect dialect.Dialect
}

// NewStore constructs the api for data access.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log:     log,
		db:      db,
		dialect: dialect.Postgres{},
	}
}

// NewWithTx constructs a store bound to the transaction.
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (runbus.Storer, error) {
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

// write is a run row and the event recorded with it, in one statement.
type write struct {
	runDB
	ExpectedGeneration int64         `db:"expected_generation"`
	ActorID            uuid.NullUUID `db:"actor_id"`
	EventType          string        `db:"event_type"`
	EventMessage       string        `db:"event_message"`
	EventDate          time.Time     `db:"event_date"`
}

func toWrite(run runbus.Run, event runbus.Event) write {
	ev := toDBEvent(event)
	return write{
		runDB:        toDBRun(run),
		ActorID:      ev.ActorID,
		EventType:    ev.Type,
		EventMessage: ev.Message,
		EventDate:    ev.DateCreated,
	}
}

// recordEvent is the tail of every write: it stores the event for the row the
// write touched, and returns that row's id. An empty event type stores nothing.
const recordEvent = `,
	recorded AS (
		INSERT INTO run_events (run_id, actor_id, type, message, date_created)
		SELECT run_id, CAST(:actor_id AS uuid), CAST(:event_type AS text), CAST(:event_message AS text), CAST(:event_date AS timestamp)
		FROM written
		WHERE CAST(:event_type AS text) <> ''
	)
	SELECT run_id FROM written`

// Create adds a run and its first event to the database.
func (s *Store) Create(ctx context.Context, run runbus.Run, event runbus.Event) error {
	const q = `
	WITH written AS (
		INSERT INTO runs
			(run_id, workflow, owner_id, project_id, idempotency_key_hash, generation,
			 input, suspend, cancel, attempt, resume_from,
			 phase, observed_generation, observed_attempt, date_created, date_updated)
		VALUES
			(:run_id, :workflow, :owner_id, :project_id, :idempotency_key_hash, :generation,
			 CAST(:input AS jsonb), :suspend, :cancel, :attempt, :resume_from,
			 :phase, :observed_generation, :observed_attempt, :date_created, :date_updated)
		RETURNING run_id
	)` + recordEvent

	if err := s.exec(ctx, q, toWrite(run, event)); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case uniqueViolation:
				return fmt.Errorf("db: %w", runbus.ErrIdempotencyKey)
			case foreignKeyViolation:
				return fmt.Errorf("db: %w", runbus.ErrProject)
			}
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// UpdateSpec writes the spec columns when the stored generation is the one the
// caller read. It returns ErrConflict when another request changed the run first.
func (s *Store) UpdateSpec(ctx context.Context, run runbus.Run, generation int64, event runbus.Event) error {
	const q = `
	WITH written AS (
		UPDATE runs
		SET
			suspend = :suspend,
			cancel = :cancel,
			attempt = :attempt,
			resume_from = :resume_from,
			generation = :generation,
			date_updated = :date_updated
		WHERE
			run_id = :run_id AND generation = :expected_generation
		RETURNING run_id
	)` + recordEvent

	data := toWrite(run, event)
	data.ExpectedGeneration = generation

	if err := s.exec(ctx, q, data); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return fmt.Errorf("db: %w", runbus.ErrConflict)
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// UpdateStatus writes the status columns. It never moves the observed
// generation backward, and it leaves the steps to the workers.
func (s *Store) UpdateStatus(ctx context.Context, run runbus.Run, event runbus.Event) error {
	const q = `
	WITH written AS (
		UPDATE runs
		SET
			phase = :phase,
			observed_generation = :observed_generation,
			observed_attempt = :observed_attempt,
			execution_id = :execution_id,
			reason = :reason,
			message = :message,
			result = CAST(:result AS jsonb),
			cancel_requested_at = :cancel_requested_at,
			date_closed = :date_closed,
			date_updated = :date_updated
		WHERE
			run_id = :run_id AND observed_generation <= :observed_generation
		RETURNING run_id
	)` + recordEvent

	if err := s.exec(ctx, q, toWrite(run, event)); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return fmt.Errorf("db: %w", runbus.ErrConflict)
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// exec runs one write and reports ErrDBNotFound when it touched no row.
func (s *Store) exec(ctx context.Context, query string, data write) error {
	var touched struct {
		ID uuid.UUID `db:"run_id"`
	}
	return sqldb.NamedQueryStruct(ctx, s.log, s.db, query, data, &touched)
}

// UpsertStep stores one step under its key. A step already stored keeps the time it started.
func (s *Store) UpsertStep(ctx context.Context, runID uuid.UUID, step runbus.Step) error {
	body, err := json.Marshal(toDBStep(step))
	if err != nil {
		return fmt.Errorf("encode step: %w", err)
	}

	data := struct {
		ID          uuid.UUID `db:"run_id"`
		Key         string    `db:"key"`
		Step        string    `db:"step"`
		DateUpdated time.Time `db:"date_updated"`
	}{
		ID:          runID,
		Key:         step.Key,
		Step:        string(body),
		DateUpdated: step.DateUpdated.UTC(),
	}

	const q = `
	UPDATE runs
	SET
		steps = jsonb_set(
			steps,
			ARRAY[CAST(:key AS text)],
			CAST(:step AS jsonb) || jsonb_strip_nulls(jsonb_build_object(
				'dateStarted', steps #> ARRAY[CAST(:key AS text), 'dateStarted']
			))
		),
		date_updated = :date_updated
	WHERE
		run_id = :run_id
	RETURNING run_id`

	var touched struct {
		ID uuid.UUID `db:"run_id"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &touched); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return fmt.Errorf("db: %w", runbus.ErrNotFound)
		}
		return fmt.Errorf("db: %w", err)
	}

	return nil
}

// ClearSteps empties the steps of a run.
func (s *Store) ClearSteps(ctx context.Context, runID uuid.UUID, now time.Time) error {
	data := struct {
		ID          uuid.UUID `db:"run_id"`
		DateUpdated time.Time `db:"date_updated"`
	}{
		ID:          runID,
		DateUpdated: now.UTC(),
	}

	const q = `
	UPDATE runs
	SET
		steps = CAST('{}' AS jsonb),
		date_updated = :date_updated
	WHERE
		run_id = :run_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a run. Its events go with it.
func (s *Store) Delete(ctx context.Context, run runbus.Run) error {
	data := struct {
		ID uuid.UUID `db:"run_id"`
	}{
		ID: run.ID,
	}

	const q = `
	DELETE FROM
		runs
	WHERE
		run_id = :run_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query gets the runs matching the filter.
func (s *Store) Query(ctx context.Context, filter runbus.QueryFilter, orderBy order.By, pg page.Page) ([]runbus.Run, error) {
	data := map[string]any{
		"offset":        (pg.Number() - 1) * pg.RowsPerPage(),
		"rows_per_page": pg.RowsPerPage(),
	}

	buf := bytes.NewBufferString("SELECT" + columns + " FROM runs")
	s.applyFilter(filter, data, buf)

	clause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(clause)
	s.dialect.Paginate(buf)

	var rows []runDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, buf.String(), data, &rows); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusRuns(rows)
}

// Count returns the number of runs matching the filter.
func (s *Store) Count(ctx context.Context, filter runbus.QueryFilter) (int, error) {
	data := map[string]any{}

	buf := bytes.NewBufferString("SELECT count(1) AS count FROM runs")
	s.applyFilter(filter, data, buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("db: %w", err)
	}

	return count.Count, nil
}

// QueryByID gets the run with the given id.
func (s *Store) QueryByID(ctx context.Context, runID uuid.UUID) (runbus.Run, error) {
	data := struct {
		ID uuid.UUID `db:"run_id"`
	}{
		ID: runID,
	}

	const q = "SELECT" + columns + " FROM runs WHERE run_id = :run_id"

	var row runDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &row); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return runbus.Run{}, fmt.Errorf("db: %w", runbus.ErrNotFound)
		}
		return runbus.Run{}, fmt.Errorf("db: %w", err)
	}

	return toBusRun(row)
}

// QueryUnsettled gets runs whose asked-for attempt has not closed, least recently updated first.
func (s *Store) QueryUnsettled(ctx context.Context, limit int) ([]runbus.Run, error) {
	data := struct {
		Limit int `db:"limit"`
	}{
		Limit: limit,
	}

	const q = "SELECT" + columns + `
	FROM
		runs
	WHERE
		phase IN ('pending', 'running', 'paused') OR observed_attempt < attempt
	ORDER BY
		date_updated
	LIMIT :limit`

	var rows []runDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusRuns(rows)
}

// QueryEvents gets the events of a run, oldest first.
func (s *Store) QueryEvents(ctx context.Context, runID uuid.UUID) ([]runbus.Event, error) {
	data := struct {
		ID uuid.UUID `db:"run_id"`
	}{
		ID: runID,
	}

	const q = `
	SELECT
		event_id, run_id, actor_id, type, message, date_created
	FROM
		run_events
	WHERE
		run_id = :run_id
	ORDER BY
		event_id`

	var rows []eventDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, q, data, &rows); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusEvents(rows), nil
}
