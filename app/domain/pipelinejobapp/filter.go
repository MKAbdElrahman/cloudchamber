package pipelinejobapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/google/uuid"
)

type queryParams struct {
	Page      string
	Rows      string
	OrderBy   string
	JobID     string
	ProjectID string
	Status    string
}

func parseQueryParams(r *http.Request) queryParams {
	values := r.URL.Query()

	return queryParams{
		Page:      values.Get("page"),
		Rows:      values.Get("rows"),
		OrderBy:   values.Get("orderBy"),
		JobID:     values.Get("job_id"),
		ProjectID: values.Get("project_id"),
		Status:    values.Get("status"),
	}
}

func parseFilter(qp queryParams) (pipelinejobbus.QueryFilter, error) {
	var fieldErrors errs.FieldErrors
	var filter pipelinejobbus.QueryFilter

	if qp.JobID != "" {
		id, err := uuid.Parse(qp.JobID)
		switch err {
		case nil:
			filter.ID = &id
		default:
			fieldErrors.Add("job_id", err)
		}
	}

	if qp.ProjectID != "" {
		id, err := uuid.Parse(qp.ProjectID)
		switch err {
		case nil:
			filter.ProjectID = &id
		default:
			fieldErrors.Add("project_id", err)
		}
	}

	if qp.Status != "" {
		status, err := pipelinejobbus.ParseStatus(qp.Status)
		switch err {
		case nil:
			filter.Status = &status
		default:
			fieldErrors.Add("status", err)
		}
	}

	if fieldErrors != nil {
		return pipelinejobbus.QueryFilter{}, fieldErrors.ToError()
	}

	return filter, nil
}
