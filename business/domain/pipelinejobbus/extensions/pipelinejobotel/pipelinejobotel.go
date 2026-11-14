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

// StreamFacts applies otel to a job stream read.
func (ext *Extension) StreamFacts(ctx context.Context, jobID uuid.UUID, after int64, limit int) ([]pipelinejobbus.Fact, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.streamfacts")
	defer span.End()

	return ext.bus.StreamFacts(ctx, jobID, after, limit)
}

// StreamLog applies otel to a job stream read of events and progress.
func (ext *Extension) StreamLog(ctx context.Context, jobID uuid.UUID, after int64, limit int) (pipelinejobbus.Log, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.streamlog")
	defer span.End()

	return ext.bus.StreamLog(ctx, jobID, after, limit)
}

// ProjectFacts applies otel to a project stream read.
func (ext *Extension) ProjectFacts(ctx context.Context, filter pipelinejobbus.QueryFilter, after int64, limit int) ([]pipelinejobbus.Fact, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.projectfacts")
	defer span.End()

	return ext.bus.ProjectFacts(ctx, filter, after, limit)
}

// ProjectLog applies otel to a project stream read of events and progress.
func (ext *Extension) ProjectLog(ctx context.Context, filter pipelinejobbus.QueryFilter, after int64, limit int) (pipelinejobbus.Log, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.projectlog")
	defer span.End()

	return ext.bus.ProjectLog(ctx, filter, after, limit)
}

// ProjectCursor applies otel to a project cursor read.
func (ext *Extension) ProjectCursor(ctx context.Context, filter pipelinejobbus.QueryFilter) (int64, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.projectcursor")
	defer span.End()

	return ext.bus.ProjectCursor(ctx, filter)
}

// AttachHeadFacts applies otel to a head-fact read.
func (ext *Extension) AttachHeadFacts(ctx context.Context, jobs []pipelinejobbus.Job) ([]pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.attachheadfacts")
	defer span.End()

	return ext.bus.AttachHeadFacts(ctx, jobs)
}

// RecordProgress appends one progress update. The job row stays as it was.
func (ext *Extension) RecordProgress(ctx context.Context, job pipelinejobbus.Job, stage pipelinejobbus.Stage, fraction *float64, headline *string) error {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.recordprogress")
	defer span.End()

	return ext.bus.RecordProgress(ctx, job, stage, fraction, headline)
}

// Record applies otel to one step sentence.
func (ext *Extension) Record(ctx context.Context, job pipelinejobbus.Job, cause pipelinejobbus.Cause) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.record")
	defer span.End()

	updated, err := ext.bus.Record(ctx, job, cause)
	if err != nil {
		return pipelinejobbus.Job{}, err
	}

	return updated, nil
}

// Submit applies otel to the accepted fact.
func (ext *Extension) Submit(ctx context.Context, nj pipelinejobbus.NewJob) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.submit")
	defer span.End()

	return ext.bus.Submit(ctx, nj)
}

// RecordStarted applies otel to the started fact.
func (ext *Extension) RecordStarted(ctx context.Context, job pipelinejobbus.Job, runID string) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.recordstarted")
	defer span.End()

	return ext.bus.RecordStarted(ctx, job, runID)
}

// RecordRun applies otel to the run-level fact.
func (ext *Extension) RecordRun(ctx context.Context, job pipelinejobbus.Job, cause pipelinejobbus.Cause) (pipelinejobbus.Job, error) {
	ctx, span := otel.AddSpan(ctx, "business.pipelinejobbus.recordrun")
	defer span.End()

	updated, err := ext.bus.RecordRun(ctx, job, cause)
	if err != nil {
		return pipelinejobbus.Job{}, err
	}

	return updated, nil
}
