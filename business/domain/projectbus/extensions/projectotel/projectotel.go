// Package projectotel provides an extension for projectbus that adds
// otel tracking.
package projectotel

import (
	"context"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/google/uuid"
)

// Extension provides a wrapper for otel functionality around the project bus.
type Extension struct {
	bus projectbus.ExtBusiness
}

// NewExtension constructs a new extension that wraps the project bus with otel.
func NewExtension() projectbus.Extension {
	return func(bus projectbus.ExtBusiness) projectbus.ExtBusiness {
		return &Extension{bus: bus}
	}
}

// NewWithTx constructs a new extension around the transaction-bound business.
func (ext *Extension) NewWithTx(tx sqldb.CommitRollbacker) (projectbus.ExtBusiness, error) {
	bus, err := ext.bus.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	next := Extension{
		bus: bus,
	}

	return &next, nil
}

// Create applies otel to project creation.
func (ext *Extension) Create(ctx context.Context, np projectbus.NewProject) (projectbus.Project, error) {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.create")
	defer span.End()

	prj, err := ext.bus.Create(ctx, np)
	if err != nil {
		return projectbus.Project{}, err
	}

	return prj, nil
}

// Update applies otel to a project update.
func (ext *Extension) Update(ctx context.Context, prj projectbus.Project, up projectbus.UpdateProject) (projectbus.Project, error) {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.update")
	defer span.End()

	prj, err := ext.bus.Update(ctx, prj, up)
	if err != nil {
		return projectbus.Project{}, err
	}

	return prj, nil
}

// Delete applies otel to project deletion.
func (ext *Extension) Delete(ctx context.Context, prj projectbus.Project) error {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.delete")
	defer span.End()

	if err := ext.bus.Delete(ctx, prj); err != nil {
		return err
	}

	return nil
}

// Query applies otel to a project query.
func (ext *Extension) Query(ctx context.Context, filter projectbus.QueryFilter, orderBy order.By, pg page.Page) ([]projectbus.Project, error) {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.query")
	defer span.End()

	return ext.bus.Query(ctx, filter, orderBy, pg)
}

// Count applies otel to a project count.
func (ext *Extension) Count(ctx context.Context, filter projectbus.QueryFilter) (int, error) {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.count")
	defer span.End()

	return ext.bus.Count(ctx, filter)
}

// QueryByID applies otel to a lookup by project id.
func (ext *Extension) QueryByID(ctx context.Context, projectID uuid.UUID) (projectbus.Project, error) {
	ctx, span := otel.AddSpan(ctx, "business.projectbus.querybyid")
	defer span.End()

	return ext.bus.QueryByID(ctx, projectID)
}
