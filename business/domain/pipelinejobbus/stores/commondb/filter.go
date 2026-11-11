package commondb

import (
	"bytes"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

// ApplyFilter appends a WHERE clause to buf based on filter.
func ApplyFilter(filter pipelinejobbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string

	if filter.ID != nil {
		data["job_id"] = *filter.ID
		wc = append(wc, "job_id = :job_id")
	}

	if filter.OwnerID != nil {
		data["owner_id"] = *filter.OwnerID
		wc = append(wc, "owner_id = :owner_id")
	}

	if filter.ProjectID != nil {
		data["project_id"] = *filter.ProjectID
		wc = append(wc, "project_id = :project_id")
	}

	if filter.WorkflowID != nil {
		data["workflow_id"] = *filter.WorkflowID
		wc = append(wc, "workflow_id = :workflow_id")
	}

	if filter.IdempotencyKeyHash != nil {
		data["idempotency_key_hash"] = *filter.IdempotencyKeyHash
		wc = append(wc, "idempotency_key_hash = :idempotency_key_hash")
	}

	if filter.Status != nil {
		data["status"] = filter.Status.String()
		wc = append(wc, "status = :status")
	}

	if len(wc) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(wc, " AND "))
	}
}
