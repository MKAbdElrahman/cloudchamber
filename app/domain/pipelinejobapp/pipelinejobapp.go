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
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/messagedb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
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
	hub            *messagedb.Hub
}

func newApp(log *logger.Logger, pipelineJobBus pipelinejobbus.ExtBusiness, artifactBus artifactbus.ExtBusiness, temporal client.Client, hub *messagedb.Hub) *app {
	return &app{
		log:            log,
		pipelineJobBus: pipelineJobBus,
		artifactBus:    artifactBus,
		temporal:       temporal,
		hub:            hub,
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

		existing, found, err := a.jobByIdempotency(ctx, ownerID, hash)
		if err != nil {
			return errs.Errorf(errs.Internal, "query: %s", err)
		}
		if found {
			return a.created(ctx, existing)
		}

		jobID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(ownerID.String()+":"+key))
	} else {
		jobID = uuid.New()
	}

	workflowID := pipelinejobbus.WorkflowID(jobID)

	nj := pipelinejobbus.NewJob{
		ID:                 jobID,
		OwnerID:            ownerID,
		ProjectID:          art.ProjectID,
		InputFileID:        art.FileID,
		IdempotencyKeyHash: idempotencyHash,
	}

	job, err := a.pipelineJobBus.Submit(ctx, nj)
	if err != nil {
		// Race on idempotency key: return existing.
		if idempotencyHash != nil {
			existing, found, qerr := a.jobByIdempotency(ctx, ownerID, *idempotencyHash)
			if qerr == nil && found {
				return a.created(ctx, existing)
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
		note := "could not start the run"
		if _, rerr := a.pipelineJobBus.RecordRun(ctx, job, pipelinejobbus.RunFailed{Note: &note}); rerr != nil {
			a.log.Error(ctx, "pipelinejob.create.record-failed", "err", rerr)
		}
		return errs.Errorf(errs.Internal, "start workflow: %s", err)
	}

	started, err := a.pipelineJobBus.RecordStarted(ctx, job, run.GetRunID())
	if err != nil {
		a.log.Error(ctx, "pipelinejob.create.record-started", "err", err)
	} else {
		job = started
	}

	return a.created(ctx, job)
}

func (a *app) created(ctx context.Context, job pipelinejobbus.Job) web.Encoder {
	jobs, err := a.pipelineJobBus.AttachHeadFacts(ctx, []pipelinejobbus.Job{job})
	if err != nil {
		a.log.Error(ctx, "pipelinejob.create.facts", "err", err)
	} else if len(jobs) == 1 {
		job = jobs[0]
	}
	return CreatedJob{Job: a.appJob(ctx, job)}
}

func (a *app) jobByIdempotency(ctx context.Context, ownerID uuid.UUID, hash string) (pipelinejobbus.Job, bool, error) {
	existing, err := a.pipelineJobBus.Query(ctx, pipelinejobbus.QueryFilter{
		OwnerID:            &ownerID,
		IdempotencyKeyHash: &hash,
	}, pipelinejobbus.DefaultOrderBy, page.MustParse("1", "1"))
	if err != nil {
		return pipelinejobbus.Job{}, false, err
	}
	if len(existing) == 0 {
		return pipelinejobbus.Job{}, false, nil
	}
	return existing[0], true, nil
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

	ownerID, enc := a.ownerFilter(ctx)
	if enc != nil {
		return enc
	}
	if ownerID != nil {
		filter.OwnerID = ownerID
	}

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, pipelinejobbus.DefaultOrderBy)
	if err != nil {
		return errs.NewFieldErrors("order", err)
	}

	var cursor int64
	if filter.ProjectID != nil {
		cursor, err = a.pipelineJobBus.ProjectCursor(ctx, filter)
		if err != nil {
			return errs.Errorf(errs.Internal, "query: %s", err)
		}
	}

	jobs, err := a.pipelineJobBus.Query(ctx, filter, orderBy, pg)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}
	jobs, err = a.pipelineJobBus.AttachHeadFacts(ctx, jobs)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}

	total, err := a.pipelineJobBus.Count(ctx, filter)
	if err != nil {
		return errs.Errorf(errs.Internal, "count: %s", err)
	}

	return JobList{
		Result:      query.NewResult(toAppJobs(jobs, a.fileNames(ctx, jobs)), total, pg),
		FactsCursor: cursor,
	}
}

func (a *app) queryByID(ctx context.Context, _ *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}
	log, err := a.pipelineJobBus.StreamLog(ctx, job.ID, 0, jobStreamRows)
	if err != nil {
		return errs.Errorf(errs.Internal, "query: %s", err)
	}
	job.Facts = log.Facts
	job.Progress = log.Progress

	return a.appJob(ctx, job)
}

func (a *app) cancel(ctx context.Context, _ *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}

	// The request asks the run to stop. The workflow records the run on the way out.
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
