// Package runapp maintains the app layer api for runs.
//
// The handlers are declarative. Creating a run stores what the caller wants. A
// command edits that. Neither talks to the workflow engine: the controller
// reads the spec and does the work, and the response shows the status it last recorded.
package runapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/query"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
)

type app struct {
	log       *logger.Logger
	runs      Runs
	workflows *workflows.Registry
	nudge     func(runID uuid.UUID)
}

func newApp(cfg Config) *app {
	registry := cfg.Workflows
	if registry == nil {
		registry = workflows.NewRegistry()
	}

	nudge := cfg.Nudge
	if nudge == nil {
		nudge = func(uuid.UUID) {}
	}

	return &app{log: cfg.Log, runs: cfg.Runs, workflows: registry, nudge: nudge}
}

// queryWorkflows lists the workflows a caller can start a run of. It is the same for every caller.
func (a *app) queryWorkflows(context.Context, *http.Request) web.Encoder {
	return fromWorkflowsResponse(a.workflows.List())
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var app NewRun
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	nr, err := toBusNewRun(ctx, app)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	// The workflow checks its own input, so a request it would fail on is
	// refused here and no run is stored.
	req := workflows.Request{OwnerID: nr.OwnerID, ProjectID: nr.ProjectID, Input: nr.Input}
	if err := a.workflows.Validate(ctx, nr.Workflow.String(), req); err != nil {
		if inputErr, ok := errors.AsType[*workflows.InputError](err); ok {
			return errs.NewFieldErrors(inputField(inputErr.Field), errors.New(inputErr.Message))
		}
		return errs.Errorf(errs.Internal, "validate: workflow[%s]: %s", nr.Workflow, err)
	}

	key := strings.TrimSpace(app.IdempotencyKey)
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}
	if key != "" {
		sum := sha256.Sum256([]byte(nr.OwnerID.String() + ":" + key))
		nr.IdempotencyKeyHash = hex.EncodeToString(sum[:])
	}

	run, err := a.runs.Create(ctx, nr)
	if err != nil {
		switch {
		case errors.Is(err, runbus.ErrIdempotencyKey):
			return a.existing(ctx, nr)
		case errors.Is(err, runbus.ErrProject):
			return errs.NewFieldErrors("projectId", errors.New("projectId must be a project you own"))
		}
		return errs.Errorf(errs.Internal, "create: %s", err)
	}

	a.nudge(run.ID)

	return fromBusRunResponse(run)
}

// inputField names a field of the request body. A workflow names fields of
// its own input, which sits under "input" in the body.
func inputField(field string) string {
	switch field {
	case "workflow", "input":
		return field
	default:
		return "input." + field
	}
}

// existing returns the run a repeated create already made.
func (a *app) existing(ctx context.Context, nr runbus.NewRun) web.Encoder {
	found, err := a.runs.Query(ctx, runbus.QueryFilter{
		OwnerID:            &nr.OwnerID,
		IdempotencyKeyHash: &nr.IdempotencyKeyHash,
	}, runbus.DefaultOrderBy, page.MustParse("1", "1"))
	if err != nil || len(found) == 0 {
		return errs.Errorf(errs.Internal, "create: idempotency key: %v", err)
	}
	return fromBusRunResponse(found[0])
}

// command returns the handler for one user command.
func (a *app) command(command runbus.Command) web.HandlerFunc {
	return func(ctx context.Context, _ *http.Request) web.Encoder {
		run, err := mid.GetRun(ctx)
		if err != nil {
			return errs.Errorf(errs.Internal, "run missing in context: %s", err)
		}

		actorID, err := mid.GetUserID(ctx)
		if err != nil {
			return errs.New(errs.Unauthenticated, err)
		}

		changed, err := a.runs.Command(ctx, run, command, actorID)
		if err != nil {
			switch {
			case errors.Is(err, runbus.ErrNotAllowed):
				return errs.New(errs.FailedPrecondition, errors.New("the run is not in a state that allows this"))
			case errors.Is(err, runbus.ErrConflict):
				return errs.New(errs.Aborted, errors.New("the run changed while this was being applied, try again"))
			default:
				return errs.Errorf(errs.Internal, "command: runID[%s]: %s", run.ID, err)
			}
		}

		a.nudge(changed.ID)

		return fromBusRunResponse(changed)
	}
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, runbus.DefaultOrderBy)
	if err != nil {
		return errs.NewFieldErrors("order", err)
	}

	runs, err := a.runs.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}

	total, err := a.runs.Count(ctx, filter)
	if err != nil {
		return errs.Errorf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(fromBusRunsResponse(runs), total, pg)
}

func (a *app) queryByID(ctx context.Context, _ *http.Request) web.Encoder {
	run, err := mid.GetRun(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "querybyid: %s", err)
	}

	return fromBusRunResponse(run)
}

func (a *app) queryEvents(ctx context.Context, _ *http.Request) web.Encoder {
	run, err := mid.GetRun(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "run missing in context: %s", err)
	}

	events, err := a.runs.QueryEvents(ctx, run.ID)
	if err != nil {
		return errs.Errorf(errs.Internal, "events: runID[%s]: %s", run.ID, err)
	}

	return fromBusEventsResponse(events)
}
