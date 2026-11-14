// Package runclose records runs whose workflow has already stopped.
package runclose

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
)

// Run checks open jobs until ctx is canceled.
func Run(ctx context.Context, log *logger.Logger, temporal client.Client, jobs pipelinejobbus.ExtBusiness) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := closeOnce(ctx, log, temporal, jobs); err != nil {
			log.Error(ctx, "pipeline.recordrun", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func closeOnce(ctx context.Context, log *logger.Logger, temporal client.Client, jobs pipelinejobbus.ExtBusiness) error {
	for _, value := range []string{pipelinejobbus.StatusSubmitted, pipelinejobbus.StatusRunning} {
		status := pipelinejobbus.MustParseStatus(value)
		open, err := openJobs(ctx, jobs, status)
		if err != nil {
			return err
		}
		for _, job := range open {
			if err := closeJob(ctx, temporal, jobs, job); err != nil {
				log.Error(ctx, "pipeline.recordrun", "jobID", job.ID, "err", err)
			}
		}
	}
	return nil
}

func openJobs(ctx context.Context, jobs pipelinejobbus.ExtBusiness, status pipelinejobbus.Status) ([]pipelinejobbus.Job, error) {
	filter := pipelinejobbus.QueryFilter{Status: &status}
	var all []pipelinejobbus.Job
	for number := 1; ; number++ {
		pg := page.MustParse(strconv.Itoa(number), "100")
		batch, err := jobs.Query(ctx, filter, pipelinejobbus.DefaultOrderBy, pg)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			return all, nil
		}
	}
}

func closeJob(ctx context.Context, temporal client.Client, jobs pipelinejobbus.ExtBusiness, job pipelinejobbus.Job) error {
	if job.WorkflowID == "" {
		return nil
	}
	desc, err := temporal.DescribeWorkflowExecution(ctx, job.WorkflowID, job.RunID)
	if err != nil {
		if _, ok := errors.AsType[*serviceerror.NotFound](err); ok {
			return nil
		}
		return err
	}
	info := desc.GetWorkflowExecutionInfo()
	if info == nil {
		return nil
	}

	var cause pipelinejobbus.Cause
	switch info.GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		cause = pipelinejobbus.RunCompleted{}
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		text := closeMessage(ctx, temporal, job)
		cause = pipelinejobbus.RunFailed{Note: &text}
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		cause = pipelinejobbus.RunCanceled{}
	default:
		return nil
	}

	_, err = jobs.RecordRun(ctx, job, cause)
	return err
}

func closeMessage(ctx context.Context, temporal client.Client, job pipelinejobbus.Job) string {
	getCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	text := "the run ended before it recorded an outcome"
	err := temporal.GetWorkflow(getCtx, job.WorkflowID, job.RunID).Get(getCtx, nil)
	if err != nil {
		if trimmed := strings.TrimSpace(err.Error()); trimmed != "" {
			text = trimmed
		}
	}
	if len(text) > 500 {
		return text[:500]
	}
	return text
}
