package runapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

type queryParams struct {
	Page      string
	Rows      string
	OrderBy   string
	ProjectID string
	Workflow  string
	Phase     string
}

func parseQueryParams(r *http.Request) queryParams {
	values := r.URL.Query()

	return queryParams{
		Page:      values.Get("page"),
		Rows:      values.Get("rows"),
		OrderBy:   values.Get("orderBy"),
		ProjectID: values.Get("project_id"),
		Workflow:  values.Get("workflow"),
		Phase:     values.Get("phase"),
	}
}

func parseFilter(qp queryParams) (runbus.QueryFilter, error) {
	var fieldErrs errs.FieldErrors
	var filter runbus.QueryFilter

	if qp.ProjectID != "" {
		id, err := uuid.Parse(qp.ProjectID)
		switch err {
		case nil:
			filter.ProjectID = &id
		default:
			fieldErrs.Add("project_id", err)
		}
	}

	if qp.Workflow != "" {
		name, err := workflowname.Parse(qp.Workflow)
		switch err {
		case nil:
			filter.Workflow = &name
		default:
			fieldErrs.Add("workflow", err)
		}
	}

	if qp.Phase != "" {
		phase, err := phase.Parse(qp.Phase)
		switch err {
		case nil:
			filter.Phase = &phase
		default:
			fieldErrs.Add("phase", err)
		}
	}

	if fieldErrs != nil {
		return runbus.QueryFilter{}, fieldErrs.ToError()
	}

	return filter, nil
}
