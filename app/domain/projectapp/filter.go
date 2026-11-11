package projectapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/title"
	"github.com/google/uuid"
)

type queryParams struct {
	Page    string
	Rows    string
	OrderBy string
	ID      string
	Name    string
}

func parseQueryParams(r *http.Request) queryParams {
	values := r.URL.Query()

	return queryParams{
		Page:    values.Get("page"),
		Rows:    values.Get("rows"),
		OrderBy: values.Get("orderBy"),
		ID:      values.Get("project_id"),
		Name:    values.Get("name"),
	}
}

func parseFilter(qp queryParams) (projectbus.QueryFilter, error) {
	var fieldErrs errs.FieldErrors
	var filter projectbus.QueryFilter

	if qp.ID != "" {
		id, err := uuid.Parse(qp.ID)
		switch err {
		case nil:
			filter.ID = &id
		default:
			fieldErrs.Add("project_id", err)
		}
	}

	if qp.Name != "" {
		name, err := title.Parse(qp.Name)
		switch err {
		case nil:
			filter.Name = &name
		default:
			fieldErrs.Add("name", err)
		}
	}

	if fieldErrs != nil {
		return projectbus.QueryFilter{}, fieldErrs.ToError()
	}

	return filter, nil
}
