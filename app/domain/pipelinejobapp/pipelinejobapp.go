// Package pipelinejobapp maintains the app layer api for pipeline jobs.
package pipelinejobapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/workflow"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
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
		msg := err.Error()
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

	job, err = a.reconcile(ctx, job)
	if err != nil {
		a.log.Error(ctx, "pipelinejob.reconcile", "err", err, "job_id", job.ID)
	}

	return a.appJob(ctx, job)
}

func (a *app) cancel(ctx context.Context, _ *http.Request) web.Encoder {
	job, err := mid.GetPipelineJob(ctx)
	if err != nil {
		return errs.Errorf(errs.Internal, "pipeline job missing in context: %s", err)
	}

	if err := a.temporal.CancelWorkflow(ctx, job.WorkflowID, job.RunID); err != nil {
		a.log.Error(ctx, "pipelinejob.cancel", "err", err, "job_id", job.ID)
		return errs.Errorf(errs.Internal, "cancel workflow: %s", err)
	}

	st, _ := pipelinejobbus.ParseStatus(pipelinejobbus.StatusCanceled)
	job, err = a.pipelineJobBus.Update(ctx, job, pipelinejobbus.UpdateJob{Status: &st, ClearError: true})
	if err != nil {
		a.log.Error(ctx, "pipelinejob.cancel.update", "err", err)
	}

	return a.appJob(ctx, job)
}

func (a *app) appJob(ctx context.Context, job pipelinejobbus.Job) Job {
	return toAppJob(job, a.fileNames(ctx, []pipelinejobbus.Job{job}))
}

func parseFileID(raw string) (uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
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

func (a *app) reconcile(ctx context.Context, job pipelinejobbus.Job) (pipelinejobbus.Job, error) {
	if a.temporal == nil || job.WorkflowID == "" {
		return job, nil
	}
	if job.Status.IsTerminal() {
		return job, nil
	}

	desc, err := a.temporal.DescribeWorkflowExecution(ctx, job.WorkflowID, job.RunID)
	if err != nil {
		return job, err
	}
	if desc == nil || desc.WorkflowExecutionInfo == nil {
		return job, nil
	}

	status := mapTemporalStatus(desc.WorkflowExecutionInfo.Status)
	uj := pipelinejobbus.UpdateJob{}
	changed := false

	if status != "" && status != job.Status.String() {
		st, err := pipelinejobbus.ParseStatus(status)
		if err == nil {
			uj.Status = &st
			changed = true
		}
	}

	if status == pipelinejobbus.StatusCompleted {
		var result workflow.ProcessSourceResult
		run := a.temporal.GetWorkflow(ctx, job.WorkflowID, job.RunID)
		if err := run.Get(ctx, &result); err == nil {
			if id, ok := parseFileID(result.AudioFileID); ok {
				uj.AudioFileID = &id
				changed = true
			}
			if id, ok := parseFileID(result.TranscriptFileID); ok {
				uj.TranscriptFileID = &id
				changed = true
			}
			if id, ok := parseFileID(result.KeyFramesFileID); ok {
				uj.KeyFramesFileID = &id
				changed = true
			}
			if id, ok := parseFileID(result.DocumentFileID); ok {
				uj.DocumentFileID = &id
				changed = true
			}
		}
		uj.ClearError = true
		changed = true
	}

	if status == pipelinejobbus.StatusFailed || status == pipelinejobbus.StatusTerminated || status == pipelinejobbus.StatusTimedOut {
		msg := status
		uj.ErrorMessage = &msg
		changed = true
	}

	if !changed {
		return job, nil
	}

	return a.pipelineJobBus.Update(ctx, job, uj)
}

func mapTemporalStatus(status enumspb.WorkflowExecutionStatus) string {
	switch status {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return pipelinejobbus.StatusRunning
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return pipelinejobbus.StatusCompleted
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
		return pipelinejobbus.StatusFailed
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return pipelinejobbus.StatusCanceled
	case enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return pipelinejobbus.StatusTerminated
	case enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return pipelinejobbus.StatusTimedOut
	default:
		return ""
	}
}

func midGetWriter(ctx context.Context) (http.ResponseWriter, error) {
	w := web.GetWriter(ctx)
	if w == nil {
		return nil, errs.Errorf(errs.Internal, "response writer missing in context")
	}
	return w, nil
}
