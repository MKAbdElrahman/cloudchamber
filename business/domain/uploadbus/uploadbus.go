// Package uploadbus provides business access to upload domain.
package uploadbus

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
)

// Set of error variables for CRUD operations.
var (
	ErrNotFound = errors.New("upload not found")
)

// Storer interface declares the behavior this package needs to persist and
// retrieve data.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Create(ctx context.Context, up Upload) error
	Update(ctx context.Context, up Upload) error
	Delete(ctx context.Context, up Upload) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Upload, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByObjectKey(ctx context.Context, objectKey string) (Upload, error)
	Upsert(ctx context.Context, up Upload) error
	MarkTerminated(ctx context.Context, objectKey string, now time.Time) error
}

// ExtBusiness interface provides support for extensions that wrap extra functionality
// around the core business logic.
type ExtBusiness interface {
	NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error)
	Create(ctx context.Context, nu NewUpload) (Upload, error)
	Update(ctx context.Context, up Upload, uu UpdateUpload) (Upload, error)
	Delete(ctx context.Context, up Upload) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Upload, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByObjectKey(ctx context.Context, objectKey string) (Upload, error)
	UpsertReady(ctx context.Context, nu NewUpload) (Upload, error)
	MarkTerminated(ctx context.Context, objectKey string) error
}

// Extension is a function that wraps a new layer of business logic
// around the existing business logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages the set of APIs for upload access.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs an upload business API for use.
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

	return newBusiness(b.log, storer), nil
}

// Create adds a new upload to the system.
func (b *Business) Create(ctx context.Context, nu NewUpload) (Upload, error) {
	now := time.Now().UTC()

	up := Upload{
		ObjectKey:    nu.ObjectKey,
		OwnerID:      nu.OwnerID,
		Bucket:       nu.Bucket,
		ObjectKeyRef: nu.ObjectKeyRef,
		SizeBytes:    nu.SizeBytes,
		ETag:         nu.ETag,
		Status:       nu.Status,
		TusdUploadID: nu.TusdUploadID,
		DateCreated:  now,
		DateUpdated:  now,
	}

	if err := b.storer.Create(ctx, up); err != nil {
		return Upload{}, fmt.Errorf("create: %w", err)
	}

	return up, nil
}

// Update modifies information about an upload.
func (b *Business) Update(ctx context.Context, up Upload, uu UpdateUpload) (Upload, error) {
	if uu.SizeBytes != nil {
		up.SizeBytes = *uu.SizeBytes
	}
	if uu.ETag != nil {
		up.ETag = uu.ETag
	}
	if uu.ClearETag {
		up.ETag = nil
	}
	if uu.Status != nil {
		up.Status = *uu.Status
	}
	if uu.TusdUploadID != nil {
		up.TusdUploadID = uu.TusdUploadID
	}

	up.DateUpdated = time.Now().UTC()

	if err := b.storer.Update(ctx, up); err != nil {
		return Upload{}, fmt.Errorf("update: %w", err)
	}

	return up, nil
}

// Delete removes the specified upload.
func (b *Business) Delete(ctx context.Context, up Upload) error {
	if err := b.storer.Delete(ctx, up); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Query retrieves a list of existing uploads.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Upload, error) {
	ups, err := b.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return ups, nil
}

// Count returns the total number of uploads.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.storer.Count(ctx, filter)
}

// QueryByObjectKey finds an upload by its object key.
func (b *Business) QueryByObjectKey(ctx context.Context, objectKey string) (Upload, error) {
	up, err := b.storer.QueryByObjectKey(ctx, objectKey)
	if err != nil {
		return Upload{}, fmt.Errorf("query: objectKey[%s]: %w", objectKey, err)
	}
	return up, nil
}

// UpsertReady records a verified, finalized upload as ready (idempotent).
func (b *Business) UpsertReady(ctx context.Context, nu NewUpload) (Upload, error) {
	now := time.Now().UTC()

	status := nu.Status
	if status.String() == "" {
		status = MustParseStatus(StatusReady)
	}

	up := Upload{
		ObjectKey:    nu.ObjectKey,
		OwnerID:      nu.OwnerID,
		Bucket:       nu.Bucket,
		ObjectKeyRef: nu.ObjectKeyRef,
		SizeBytes:    nu.SizeBytes,
		ETag:         nu.ETag,
		Status:       status,
		TusdUploadID: nu.TusdUploadID,
		DateCreated:  now,
		DateUpdated:  now,
	}

	if err := b.storer.Upsert(ctx, up); err != nil {
		return Upload{}, fmt.Errorf("upsert: %w", err)
	}

	return b.QueryByObjectKey(ctx, nu.ObjectKey)
}

// MarkTerminated marks an upload as terminated if it exists (idempotent).
func (b *Business) MarkTerminated(ctx context.Context, objectKey string) error {
	if err := b.storer.MarkTerminated(ctx, objectKey, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark terminated: %w", err)
	}
	return nil
}
