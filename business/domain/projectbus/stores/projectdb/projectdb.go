// Package projectdb contains the PostgreSQL implementation of the project store.
package projectdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb/dialect"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Store manages project database access on PostgreSQL.
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
func (s *Store) NewWithTx(tx sqldb.CommitRollbacker) (projectbus.Storer, error) {
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

// Create adds a project to the database.
func (s *Store) Create(ctx context.Context, prj projectbus.Project) error {
	const q = `
	INSERT INTO projects
		(project_id, owner_id, name, date_created, date_updated)
	VALUES
		(:project_id, :owner_id, :name, :date_created, :date_updated)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, ToDBProject(prj)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Update modifies a project.
func (s *Store) Update(ctx context.Context, prj projectbus.Project) error {
	const q = `
	UPDATE
		projects
	SET
		"name" = :name,
		date_updated = :date_updated
	WHERE
		project_id = :project_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, ToDBProject(prj)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Delete removes a project.
func (s *Store) Delete(ctx context.Context, prj projectbus.Project) error {
	data := struct {
		ID string `db:"project_id"`
	}{
		ID: prj.ID.String(),
	}

	const q = `
	DELETE FROM
		projects
	WHERE
		project_id = :project_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

// Query gets the projects matching the filter.
func (s *Store) Query(ctx context.Context, filter projectbus.QueryFilter, orderBy order.By, pg page.Page) ([]projectbus.Project, error) {
	data := map[string]any{
		"offset":        (pg.Number() - 1) * pg.RowsPerPage(),
		"rows_per_page": pg.RowsPerPage(),
	}

	const q = `
	SELECT
		project_id, owner_id, name, date_created, date_updated
	FROM
		projects`

	buf := bytes.NewBufferString(q)
	ApplyFilter(filter, data, buf)

	clause, err := OrderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(clause)
	s.dialect.Paginate(buf)

	var rows []ProjectDB
	if err := sqldb.NamedQuerySlice(ctx, s.log, s.db, buf.String(), data, &rows); err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return ToBusProjects(rows)
}

// Count returns the number of projects matching the filter.
func (s *Store) Count(ctx context.Context, filter projectbus.QueryFilter) (int, error) {
	data := map[string]any{}

	const q = `
	SELECT
		count(1) AS count
	FROM
		projects`

	buf := bytes.NewBufferString(q)
	ApplyFilter(filter, data, buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("db: %w", err)
	}

	return count.Count, nil
}

// QueryByID finds the project identified by the given id.
func (s *Store) QueryByID(ctx context.Context, projectID uuid.UUID) (projectbus.Project, error) {
	data := struct {
		ID string `db:"project_id"`
	}{
		ID: projectID.String(),
	}

	const q = `
	SELECT
		project_id, owner_id, name, date_created, date_updated
	FROM
		projects
	WHERE
		project_id = :project_id`

	var row ProjectDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &row); err != nil {
		if errors.Is(err, sqldb.ErrDBNotFound) {
			return projectbus.Project{}, fmt.Errorf("db: %w", projectbus.ErrNotFound)
		}
		return projectbus.Project{}, fmt.Errorf("db: %w", err)
	}

	return ToBusProject(row)
}
