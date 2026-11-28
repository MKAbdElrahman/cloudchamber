// Package runotel provides an extension for runbus that adds
// otel tracking.
package runotel

import (
	"context"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
)

// Extension provides a wrapper for otel functionality around the run bus.
type Extension struct {
	bus runbus.ExtBusiness
}

// NewExtension constructs a new extension that wraps the run bus with otel.
func NewExtension() runbus.Extension {
	return func(bus runbus.ExtBusiness) runbus.ExtBusiness {
		return &Extension{bus: bus}
	}
}

// NewWithTx constructs a new extension around the transaction-bound business.
func (ext *Extension) NewWithTx(tx sqldb.CommitRollbacker) (runbus.ExtBusiness, error) {
	bus, err := ext.bus.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	next := Extension{
		bus: bus,
	}

	return &next, nil
}

// Create applies otel to run creation.
func (ext *Extension) Create(ctx context.Context, nr runbus.NewRun) (runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.create")
	defer span.End()

	run, err := ext.bus.Create(ctx, nr)
	if err != nil {
		return runbus.Run{}, err
	}

	return run, nil
}

// Command applies otel to a command on a run.
func (ext *Extension) Command(ctx context.Context, run runbus.Run, command runbus.Command, actorID uuid.UUID) (runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.command")
	defer span.End()

	run, err := ext.bus.Command(ctx, run, command, actorID)
	if err != nil {
		return runbus.Run{}, err
	}

	return run, nil
}

// Delete applies otel to run deletion.
func (ext *Extension) Delete(ctx context.Context, run runbus.Run) error {
	ctx, span := otel.AddSpan(ctx, "business.runbus.delete")
	defer span.End()

	if err := ext.bus.Delete(ctx, run); err != nil {
		return err
	}

	return nil
}

// Query applies otel to a run query.
func (ext *Extension) Query(ctx context.Context, filter runbus.QueryFilter, orderBy order.By, page page.Page) ([]runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.query")
	defer span.End()

	runs, err := ext.bus.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, err
	}

	return runs, nil
}

// Count applies otel to a run count.
func (ext *Extension) Count(ctx context.Context, filter runbus.QueryFilter) (int, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.count")
	defer span.End()

	count, err := ext.bus.Count(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// QueryByID applies otel to a run query by id.
func (ext *Extension) QueryByID(ctx context.Context, runID uuid.UUID) (runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.querybyid")
	defer span.End()

	run, err := ext.bus.QueryByID(ctx, runID)
	if err != nil {
		return runbus.Run{}, err
	}

	return run, nil
}

// QueryEvents applies otel to a query of a run's events.
func (ext *Extension) QueryEvents(ctx context.Context, runID uuid.UUID) ([]runbus.Event, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.queryevents")
	defer span.End()

	events, err := ext.bus.QueryEvents(ctx, runID)
	if err != nil {
		return nil, err
	}

	return events, nil
}

// Unsettled applies otel to the query for runs the controller has work for.
func (ext *Extension) Unsettled(ctx context.Context, limit int) ([]runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.unsettled")
	defer span.End()

	runs, err := ext.bus.Unsettled(ctx, limit)
	if err != nil {
		return nil, err
	}

	return runs, nil
}

// RecordStatus applies otel to recording what the controller observed.
func (ext *Extension) RecordStatus(ctx context.Context, run runbus.Run, status runbus.Status) (runbus.Run, error) {
	ctx, span := otel.AddSpan(ctx, "business.runbus.recordstatus")
	defer span.End()

	run, err := ext.bus.RecordStatus(ctx, run, status)
	if err != nil {
		return runbus.Run{}, err
	}

	return run, nil
}

// ClearSteps applies otel to clearing a run's steps.
func (ext *Extension) ClearSteps(ctx context.Context, run runbus.Run) error {
	ctx, span := otel.AddSpan(ctx, "business.runbus.clearsteps")
	defer span.End()

	if err := ext.bus.ClearSteps(ctx, run); err != nil {
		return err
	}

	return nil
}

// ReportStep applies otel to a step report.
func (ext *Extension) ReportStep(ctx context.Context, runID uuid.UUID, report runbus.StepReport) error {
	ctx, span := otel.AddSpan(ctx, "business.runbus.reportstep")
	defer span.End()

	if err := ext.bus.ReportStep(ctx, runID, report); err != nil {
		return err
	}

	return nil
}
