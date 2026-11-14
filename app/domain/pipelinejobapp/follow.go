package pipelinejobapp

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
)

const (
	followWait      = 8 * time.Second
	projectFactRows = 100
	jobFactRows     = 1000
	jobStreamRows   = 5000
)

func (a *app) followProject(ctx context.Context, r *http.Request) web.Encoder {
	projectID, err := uuid.Parse(r.URL.Query().Get("project_id"))
	if err != nil {
		return errs.NewFieldErrors("project_id", errors.New("project_id is required"))
	}
	after, enc := parseAfter(r)
	if enc != nil {
		return enc
	}

	owner, enc := a.ownerFilter(ctx)
	if enc != nil {
		return enc
	}

	return a.follow(ctx, after, true, func() (pipelinejobbus.Log, error) {
		return a.pipelineJobBus.ProjectLog(ctx, pipelinejobbus.QueryFilter{
			ProjectID: &projectID,
			OwnerID:   owner,
		}, after, projectFactRows)
	})
}

func (a *app) followJob(ctx context.Context, r *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}
	after, enc := parseAfter(r)
	if enc != nil {
		return enc
	}

	return a.follow(ctx, after, false, func() (pipelinejobbus.Log, error) {
		return a.pipelineJobBus.StreamLog(ctx, job.ID, after, jobFactRows)
	})
}

func (a *app) follow(ctx context.Context, after int64, global bool, read func() (pipelinejobbus.Log, error)) web.Encoder {
	log, err := read()
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}
	if len(log.Facts) == 0 && len(log.Progress) == 0 && a.hub != nil {
		a.hub.Wait(ctx, followWait)
		if ctx.Err() != nil {
			return errs.New(errs.Canceled, ctx.Err())
		}
		log, err = read()
		if err != nil {
			return errs.Errorf(errs.Internal, "query: %s", err)
		}
	}
	projections, err := a.projections(ctx, log)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}
	return factPage(log, after, global, projections)
}

// projections are the folded rows for the jobs in this page.
// The row is the fold of the whole log, so the screen does not fold the page itself.
func (a *app) projections(ctx context.Context, log pipelinejobbus.Log) ([]JobProjection, error) {
	seen := make(map[uuid.UUID]struct{}, len(log.Facts)+len(log.Progress))
	out := make([]JobProjection, 0)
	ids := make([]uuid.UUID, 0, len(log.Facts)+len(log.Progress))
	for _, fact := range log.Facts {
		ids = append(ids, fact.JobID)
	}
	for _, update := range log.Progress {
		ids = append(ids, update.JobID)
	}
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		job, err := a.pipelineJobBus.QueryByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, toAppProjection(job))
	}
	return out, nil
}

func (a *app) ownerFilter(ctx context.Context) (*uuid.UUID, web.Encoder) {
	if a.isAdmin(ctx) {
		return nil, nil
	}
	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		return nil, errs.New(errs.Unauthenticated, err)
	}
	return &ownerID, nil
}

func (a *app) isAdmin(ctx context.Context) bool {
	for _, name := range mid.GetClaims(ctx).Roles {
		if name == role.Admin.String() {
			return true
		}
	}
	return false
}

func parseAfter(r *http.Request) (int64, web.Encoder) {
	raw := r.URL.Query().Get("after")
	if raw == "" {
		return 0, nil
	}
	after, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || after < 0 {
		return 0, errs.NewFieldErrors("after", errors.New("after must be a non-negative position"))
	}
	return after, nil
}

func factPage(log pipelinejobbus.Log, after int64, global bool, projections []JobProjection) FactPage {
	page := FactPage{
		Facts:       toAppFacts(log.Facts),
		Progress:    toAppProgress(log.Progress),
		Cursor:      after,
		Projections: projections,
	}
	lastFact, factOK := lastPosition(log.Facts, global)
	lastProgress, progressOK := lastProgressPosition(log.Progress, global)
	switch {
	case factOK && progressOK:
		page.Cursor = max(lastFact, lastProgress)
	case factOK:
		page.Cursor = lastFact
	case progressOK:
		page.Cursor = lastProgress
	}
	return page
}

func lastPosition(facts []pipelinejobbus.Fact, global bool) (int64, bool) {
	if len(facts) == 0 {
		return 0, false
	}
	last := facts[len(facts)-1]
	if global {
		return last.GlobalPosition, true
	}
	return last.Position, true
}

func lastProgressPosition(updates []pipelinejobbus.Progress, global bool) (int64, bool) {
	if len(updates) == 0 {
		return 0, false
	}
	last := updates[len(updates)-1]
	if global {
		return last.GlobalPosition, true
	}
	return last.Position, true
}
