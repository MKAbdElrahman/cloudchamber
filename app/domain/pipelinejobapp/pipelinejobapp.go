// Package pipelinejobapp maintains the app layer api for pipeline jobs.
package pipelinejobapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	workflow "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/query"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/types/role"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type app struct {
	log            *logger.Logger
	pipelineJobBus pipelinejobbus.ExtBusiness
	artifactBus    artifactbus.ExtBusiness
	temporal       client.Client
	temporalNS     string
}

func newApp(log *logger.Logger, pipelineJobBus pipelinejobbus.ExtBusiness, artifactBus artifactbus.ExtBusiness, temporal client.Client, temporalNS string) *app {
	return &app{
		log:            log,
		pipelineJobBus: pipelineJobBus,
		artifactBus:    artifactBus,
		temporal:       temporal,
		temporalNS:     temporalNS,
	}
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var app NewJob
	if err := web.Decode(r, &app); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		return errs.New(errs.Unauthenticated, err)
	}

	fileID, err := uuid.Parse(strings.TrimSpace(app.FileID))
	if err != nil {
		return errs.NewFieldErrors("fileId", errors.New("fileId is required"))
	}

	art, err := a.artifactBus.QueryByFileID(ctx, fileID)
	if err != nil {
		if errors.Is(err, artifactbus.ErrNotFound) {
			return errs.NewFieldErrors("fileId", errors.New("fileId must reference a ready file you own"))
		}
		return errs.Errorf(errs.Internal, "resolve fileId: %s", err)
	}
	ready := art.Status.Equal(artifactbus.MustParseStatus(artifactbus.StatusReady))
	switch {
	case art.OwnerID != ownerID, !ready:
		return errs.NewFieldErrors("fileId", errors.New("fileId must reference a ready file you own"))
	case art.ProjectID == uuid.Nil:
		return errs.NewFieldErrors("fileId", errors.New("fileId must reference a file in a project you own"))
	}
	key := strings.TrimSpace(app.IdempotencyKey)
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	}

	var (
		jobID           uuid.UUID
		idempotencyHash *string
	)

	if key != "" {
		h := sha256.Sum256([]byte(ownerID.String() + ":" + key))
		hash := hex.EncodeToString(h[:])
		idempotencyHash = &hash

		filter := pipelinejobbus.QueryFilter{
			OwnerID:            &ownerID,
			IdempotencyKeyHash: &hash,
		}
		existing, err := a.pipelineJobBus.Query(ctx, filter, pipelinejobbus.DefaultOrderBy, page.MustParse("1", "1"))
		if err != nil {
			return errs.Errorf(errs.Internal, "query idempotency: %s", err)
		}
		if len(existing) > 0 {
			return a.respondCreated(ctx, existing[0])
		}

		jobID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(ownerID.String()+":"+key))
	} else {
		jobID = uuid.New()
	}

	workflowID := "cloudchamber-job-" + jobID.String()

	status, _ := pipelinejobbus.ParseStatus(pipelinejobbus.StatusSubmitted)
	nj := pipelinejobbus.NewJob{
		ID:                 jobID,
		OwnerID:            ownerID,
		ProjectID:          art.ProjectID,
		WorkflowID:         workflowID,
		InputFileID:        art.FileID,
		IdempotencyKeyHash: idempotencyHash,
		Status:             status,
	}

	job, err := a.pipelineJobBus.Create(ctx, nj)
	if err != nil {
		// Race on idempotency key: return existing.
		if idempotencyHash != nil {
			filter := pipelinejobbus.QueryFilter{
				OwnerID:            &ownerID,
				IdempotencyKeyHash: idempotencyHash,
			}
			existing, qerr := a.pipelineJobBus.Query(ctx, filter, pipelinejobbus.DefaultOrderBy, page.MustParse("1", "1"))
			if qerr == nil && len(existing) > 0 {
				return a.respondCreated(ctx, existing[0])
			}
		}
		return errs.Errorf(errs.Internal, "create: %s", err)
	}

	opts := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                workflow.TaskQueueWorkflow,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}

	run, err := a.temporal.ExecuteWorkflow(ctx, opts, workflow.ProcessSourceWorkflow, workflow.ProcessSourceInput{
		JobID:   jobID.String(),
		OwnerID: ownerID.String(),
		FileID:  art.FileID.String(),
	})
	if err != nil {
		a.log.Error(ctx, "pipelinejob.create.start", "err", err)
		msg := "could not start the run"
		st, _ := pipelinejobbus.ParseStatus(pipelinejobbus.StatusFailed)
		if _, uerr := a.pipelineJobBus.Update(ctx, job, pipelinejobbus.UpdateJob{Status: &st, ErrorMessage: &msg}); uerr != nil {
			a.log.Error(ctx, "pipelinejob.create.update-failed", "err", uerr)
		}
		return errs.Errorf(errs.Internal, "start workflow: %s", err)
	}

	runID := run.GetRunID()
	st, _ := pipelinejobbus.ParseStatus(pipelinejobbus.StatusRunning)
	job, err = a.pipelineJobBus.Update(ctx, job, pipelinejobbus.UpdateJob{RunID: &runID, Status: &st, ClearError: true})
	if err != nil {
		a.log.Error(ctx, "pipelinejob.create.update-running", "err", err)
	}

	return a.respondCreated(ctx, job)
}

func (a *app) respondCreated(ctx context.Context, job pipelinejobbus.Job) web.Encoder {
	w, err := midGetWriter(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "response writer: %s", err)
	}
	if err := WriteResponse(w, a.appJob(ctx, job)); err != nil {
		return errs.Errorf(errs.Internal, "write response: %s", err)
	}
	return web.NewNoResponse()
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, pipelinejobbus.DefaultOrderBy)
	if err != nil {
		return errs.NewFieldErrors("order", err)
	}

	claims := mid.GetClaims(ctx)
	isAdmin := false
	for _, r := range claims.Roles {
		if r == role.Admin.String() {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		ownerID, err := mid.GetUserID(ctx)
		if err != nil {
			return errs.New(errs.Unauthenticated, err)
		}
		filter.OwnerID = &ownerID
	}

	jobs, err := a.pipelineJobBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}

	total, err := a.pipelineJobBus.Count(ctx, filter)
	if err != nil {
		return errs.Errorf(errs.Internal, "count: %s", err)
	}

	return query.NewResult(toAppJobs(jobs, a.fileNames(ctx, jobs)), total, pg)
}

func (a *app) queryByID(ctx context.Context, _ *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}

	activity, err := a.activityInFlight(ctx, job)
	if err != nil {
		a.log.Error(ctx, "pipelinejob.activity", "err", err, "job_id", job.ID)
	}

	appJob := a.appJob(ctx, job)
	appJob.Activity = activity
	return appJob
}

func (a *app) cancel(ctx context.Context, _ *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}

	// The request asks the run to stop. The activity that is running, or
	// MarkCanceled when none is, stores the job before that run closes.
	if err := a.temporal.CancelWorkflow(ctx, job.WorkflowID, job.RunID); err != nil {
		a.log.Error(ctx, "pipelinejob.cancel", "err", err, "job_id", job.ID)
		return errs.Errorf(errs.Internal, "cancel workflow: %s", err)
	}

	return a.appJob(ctx, job)
}

func (a *app) appJob(ctx context.Context, job pipelinejobbus.Job) Job {
	return toAppJob(job, a.fileNames(ctx, []pipelinejobbus.Job{job}))
}

func (a *app) fileNames(ctx context.Context, jobs []pipelinejobbus.Job) map[uuid.UUID]string {
	ids := make([]uuid.UUID, 0, len(jobs))
	for _, job := range jobs {
		if job.InputFileID != uuid.Nil {
			ids = append(ids, job.InputFileID)
		}
		if job.DocumentFileID != uuid.Nil {
			ids = append(ids, job.DocumentFileID)
		}
	}

	found, err := a.artifactBus.QueryByFileIDs(ctx, ids)
	if err != nil {
		a.log.Error(ctx, "pipelinejob.file-names", "err", err)
		return map[uuid.UUID]string{}
	}

	names := make(map[uuid.UUID]string, len(found))
	for _, art := range found {
		names[art.FileID] = art.Name
	}
	return names
}

func midGetWriter(ctx context.Context) (http.ResponseWriter, error) {
	w := web.GetWriter(ctx)
	if w == nil {
		return nil, errs.Errorf(errs.Internal, "response writer missing in context")
	}
	return w, nil
}
