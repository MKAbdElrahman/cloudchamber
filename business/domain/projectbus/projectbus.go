// Package projectbus provides business access to projects.
package projectbus

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/google/uuid"
)

// Set of error variables for CRUD operations.
var (
	ErrNotFound = errors.New("project not found")
)

// Storer interface declares the behavior this package needs to persist and
// retrieve data.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Create(ctx context.Context, prj Project) error
	Update(ctx context.Context, prj Project) error
	Delete(ctx context.Context, prj Project) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Project, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, projectID uuid.UUID) (Project, error)
}

// ExtBusiness lists every public method an extension can wrap.
type ExtBusiness interface {
	// NewWithTx returns a transaction-bound copy. Extensions must bind their
	// dependencies to the transaction and wrap the returned business themselves.
	NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error)
	Create(ctx context.Context, np NewProject) (Project, error)
	Update(ctx context.Context, prj Project, up UpdateProject) (Project, error)
	Delete(ctx context.Context, prj Project) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Project, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, projectID uuid.UUID) (Project, error)
}

// Extension wraps a new layer of business logic around the existing logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages the set of APIs for project access.
type Business struct {
	storer Storer
}

// NewBusiness constructs a project business API for use.
func NewBusiness(storer Storer, extensions ...Extension) ExtBusiness {
	extBus := ExtBusiness(newBusiness(storer))

	for _, ext := range slices.Backward(extensions) {
		if ext != nil {
			extBus = ext(extBus)
		}
	}

	return extBus
}

func newBusiness(storer Storer) *Business {
	return &Business{storer: storer}
}

// NewWithTx constructs a new business value that will use the
// specified transaction in any store related calls.
func (b *Business) NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error) {
	storer, err := b.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	// Each extension rebuilds its own wrapper in NewWithTx.
	return newBusiness(storer), nil
}

// Create adds a new project.
func (b *Business) Create(ctx context.Context, np NewProject) (Project, error) {
	now := time.Now().UTC()

	prj := Project{
		ID:          uuid.New(),
		OwnerID:     np.OwnerID,
		Name:        np.Name,
		DateCreated: now,
		DateUpdated: now,
	}

	if err := b.storer.Create(ctx, prj); err != nil {
		return Project{}, fmt.Errorf("create: %w", err)
	}

	return prj, nil
}

// Update modifies a project.
func (b *Business) Update(ctx context.Context, prj Project, up UpdateProject) (Project, error) {
	if up.Name != nil {
		prj.Name = *up.Name
	}

	prj.DateUpdated = time.Now().UTC()

	if err := b.storer.Update(ctx, prj); err != nil {
		return Project{}, fmt.Errorf("update: %w", err)
	}

	return prj, nil
}

// Delete removes the specified project.
func (b *Business) Delete(ctx context.Context, prj Project) error {
	if err := b.storer.Delete(ctx, prj); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

// Query retrieves projects matching the filter.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, pg page.Page) ([]Project, error) {
	prjs, err := b.storer.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return prjs, nil
}

// Count returns the number of projects matching the filter.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.storer.Count(ctx, filter)
}

// QueryByID finds the project by its id.
func (b *Business) QueryByID(ctx context.Context, projectID uuid.UUID) (Project, error) {
	prj, err := b.storer.QueryByID(ctx, projectID)
	if err != nil {
		return Project{}, fmt.Errorf("query: projectID[%s]: %w", projectID, err)
	}

	return prj, nil
}
