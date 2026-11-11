// Package projectapp maintains the app layer api for the project domain.
package projectapp

import (
	"context"
	"errors"
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/query"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

type app struct {
	projectBus  projectbus.ExtBusiness
	artifactBus artifactbus.ExtBusiness
}

func newApp(projectBus projectbus.ExtBusiness, artifactBus artifactbus.ExtBusiness) *app {
	return &app{projectBus: projectBus, artifactBus: artifactBus}
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var app NewProject
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	np, err := toBusNewProject(ctx, app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	prj, err := a.projectBus.Create(ctx, np)
	if err != nil {
		return errs.Errorf(errs.Internal, "create: prj[%+v]: %s", prj, err)
	}

	return toAppProject(prj)
}

func (a *app) update(ctx context.Context, r *http.Request) web.Encoder {
	var app UpdateProject
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	up, err := toBusUpdateProject(app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	prj, err := mid.GetProject(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "project missing in context: %s", err)
	}

	updated, err := a.projectBus.Update(ctx, prj, up)
	if err != nil {
		return errs.Errorf(errs.Internal, "update: projectID[%s] up[%+v]: %s", prj.ID, app, err)
	}

	return toAppProject(updated)
}

func (a *app) delete(ctx context.Context, _ *http.Request) web.Encoder {
	prj, err := mid.GetProject(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "projectID missing in context: %s", err)
	}

	if err := a.projectBus.Delete(ctx, prj); err != nil {
		if errors.Is(err, projectbus.ErrNotEmpty) {
			return errs.New(errs.FailedPrecondition, errors.New("remove the files and jobs in this project first"))
		}
		return errs.Errorf(errs.Internal, "delete: projectID[%s]: %s", prj.ID, err)
	}

	return nil
}

func (a *app) query(ctx context.Context, r *http.Request) web.Encoder {
	qp := parseQueryParams(r)

	pg, err := page.Parse(qp.Page, qp.Rows)
	if err != nil {
		return errs.NewFieldErrors("page", err)
	}

	filter, err := parseFilter(qp)
	if err != nil {
		return err.(*errs.Error)
	}

	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}
	filter.OwnerID = &ownerID

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, projectbus.DefaultOrderBy)
	if err != nil {
		return errs.NewFieldErrors("order", err)
	}

	prjs, err := a.projectBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}

	total, err := a.projectBus.Count(ctx, filter)
	if err != nil {
		return errs.Errorf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppProjects(prjs), total, pg)
}

func (a *app) queryByID(ctx context.Context, _ *http.Request) web.Encoder {
	prj, err := mid.GetProject(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "querybyid: %s", err)
	}

	return toAppProject(prj)
}

func (a *app) queryFiles(ctx context.Context, _ *http.Request) web.Encoder {
	prj, err := mid.GetProject(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "project missing in context: %s", err)
	}

	arts, err := a.artifactBus.QueryByProject(ctx, prj.OwnerID, prj.ID)
	if err != nil {
		return errs.Errorf(errs.Internal, "query files: %s", err)
	}

	return toAppFiles(arts)
}
