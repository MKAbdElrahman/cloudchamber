// Package pipelinejobbus provides business access to pipeline job domain.
package pipelinejobbus

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
)

// Set of error variables for CRUD operations.
var (
	ErrNotFound = errors.New("pipeline job not found")
)

// Storer interface declares the behavior this package needs to persist and
// retrieve data.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Create(ctx context.Context, job Job) error
	Update(ctx context.Context, job Job) error
	Delete(ctx context.Context, job Job) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error)
}

// ExtBusiness interface provides support for extensions that wrap extra functionality
// around the core business logic.
type ExtBusiness interface {
	// NewWithTx returns a transaction-bound copy. Extensions must bind their
	// dependencies to the transaction and wrap the returned business themselves.
	NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error)
	Create(ctx context.Context, nj NewJob) (Job, error)
	Update(ctx context.Context, job Job, uj UpdateJob) (Job, error)
	Delete(ctx context.Context, job Job) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error)
}

// Extension is a function that wraps a new layer of business logic
// around the existing business logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages the set of APIs for pipeline job access.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs a pipeline job business API for use.
func NewBusiness(log *logger.Logger, storer Storer, extensions ...Extension) ExtBusiness {
	b := newBusiness(log, storer)

	extBus := ExtBusiness(b)

	for _, ext := range slices.Backward(extensions) {
		if ext != nil {
			extBus = ext(extBus)
		}
	}

	return extBus
}

func newBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// NewWithTx constructs a new business value that will use the
// specified transaction in any store related calls.
func (b *Business) NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error) {
	storer, err := b.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	// Each extension rebuilds its own wrapper in NewWithTx.
	return newBusiness(b.log, storer), nil
}

// Create adds a new pipeline job to the system.
func (b *Business) Create(ctx context.Context, nj NewJob) (Job, error) {
	now := time.Now().UTC()

	job := Job{
		ID:                 nj.ID,
		OwnerID:            nj.OwnerID,
		ProjectID:          nj.ProjectID,
		WorkflowID:         nj.WorkflowID,
		RunID:              nj.RunID,
		InputFileID:        nj.InputFileID,
		IdempotencyKeyHash: nj.IdempotencyKeyHash,
		Status:             nj.Status,
		DateCreated:        now,
		DateUpdated:        now,
	}

	if err := b.storer.Create(ctx, job); err != nil {
		return Job{}, fmt.Errorf("create: %w", err)
	}

	return job, nil
}

// Update modifies information about a pipeline job.
func (b *Business) Update(ctx context.Context, job Job, uj UpdateJob) (Job, error) {
	if uj.RunID != nil {
		job.RunID = *uj.RunID
	}
	if uj.Status != nil {
		job.Status = *uj.Status
	}
	if uj.ClearFailure {
		job.FailureMessage = nil
	}
	if uj.FailureMessage != nil {
		job.FailureMessage = uj.FailureMessage
	}
	if uj.StoppedActivityType != nil {
		job.StoppedActivityType = *uj.StoppedActivityType
	}
	if uj.AudioFileID != nil {
		job.AudioFileID = *uj.AudioFileID
	}
	if uj.TranscriptFileID != nil {
		job.TranscriptFileID = *uj.TranscriptFileID
	}
	if uj.KeyFramesFileID != nil {
		job.KeyFramesFileID = *uj.KeyFramesFileID
	}
	if uj.DocumentFileID != nil {
		job.DocumentFileID = *uj.DocumentFileID
	}

	job.DateUpdated = time.Now().UTC()

	if err := b.storer.Update(ctx, job); err != nil {
		return Job{}, fmt.Errorf("update: %w", err)
	}

	return job, nil
}

// Delete removes the specified pipeline job.
func (b *Business) Delete(ctx context.Context, job Job) error {
	if err := b.storer.Delete(ctx, job); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Query retrieves a list of existing pipeline jobs.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error) {
	jobs, err := b.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return jobs, nil
}

// Count returns the total number of pipeline jobs.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.storer.Count(ctx, filter)
}

// QueryByID finds the pipeline job by the specified ID.
func (b *Business) QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error) {
	job, err := b.storer.QueryByID(ctx, jobID)
	if err != nil {
		return Job{}, fmt.Errorf("query: jobID[%s]: %w", jobID, err)
	}
	return job, nil
}
