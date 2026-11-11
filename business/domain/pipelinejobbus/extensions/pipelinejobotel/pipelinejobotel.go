// Package pipelinejobotel provides an extension for pipelinejobbus that adds
// otel tracking.
package pipelinejobotel

import (
	"context"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
)

// Extension provides a wrapper for otel functionality around the pipeline job bus.
type Extension struct {
	bus pipelinejobbus.ExtBusiness
}

// NewExtension constructs a new extension that wraps the pipeline job bus with otel.
func NewExtension() pipelinejobbus.Extension {
	return func(bus pipelinejobbus.ExtBusiness) pipelinejobbus.ExtBusiness {
		return &Extension{
			bus: bus,
		}
	}
}

// NewWithTx constructs a new extension around the transaction-bound business.
func (ext *Extension) NewWithTx(tx sqldb.CommitRollbacker) (pipelinejobbus.ExtBusiness, error) {
	bus, err := ext.bus.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	next := Extension{
		bus: bus,
	}

	return &next, nil
}

// Create applies otel to recording a pipeline job.
func (ext *Extension) Create(ctx context.Context, nj pipelinejobbus.NewJob) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.create")
	defer span.End()

	job, err := ext.bus.Create(ctx, nj)
	if err != nil {
		return pipelinejobbus.Job{}, err
	}

	return job, nil
}

// Update applies otel to a pipeline job update.
func (ext *Extension) Update(ctx context.Context, job pipelinejobbus.Job, uj pipelinejobbus.UpdateJob) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.update")
	defer span.End()

	updated, err := ext.bus.Update(ctx, job, uj)
	if err != nil {
		return pipelinejobbus.Job{}, err
	}

	return updated, nil
}

// Delete applies otel to removing a pipeline job.
func (ext *Extension) Delete(ctx context.Context, job pipelinejobbus.Job) error {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.delete")
	defer span.End()

	if err := ext.bus.Delete(ctx, job); err != nil {
		return err
	}

	return nil
}

// Query applies otel to a pipeline job query.
func (ext *Extension) Query(ctx context.Context, filter pipelinejobbus.QueryFilter, orderBy order.By, pg page.Page) ([]pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.query")
	defer span.End()

	return ext.bus.Query(ctx, filter, orderBy, pg)
}

// Count applies otel to a pipeline job count.
func (ext *Extension) Count(ctx context.Context, filter pipelinejobbus.QueryFilter) (int, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.count")
	defer span.End()

	return ext.bus.Count(ctx, filter)
}

// QueryByID applies otel to a lookup by job id.
func (ext *Extension) QueryByID(ctx context.Context, jobID uuid.UUID) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.querybyid")
	defer span.End()

	return ext.bus.QueryByID(ctx, jobID)
}
