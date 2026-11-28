package rundb

import (
	"bytes"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
)

func (s *Store) applyFilter(filter runbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string

	if filter.ID != nil {
		data["run_id"] = *filter.ID
		wc = append(wc, "run_id = :run_id")
	}

	if filter.OwnerID != nil {
		data["owner_id"] = *filter.OwnerID
		wc = append(wc, "owner_id = :owner_id")
	}

	if filter.ProjectID != nil {
		data["project_id"] = *filter.ProjectID
		wc = append(wc, "project_id = :project_id")
	}

	if filter.Workflow != nil {
		data["workflow"] = filter.Workflow.String()
		wc = append(wc, "workflow = :workflow")
	}

	if filter.Phase != nil {
		data["phase"] = filter.Phase.String()
		wc = append(wc, "phase = :phase")
	}

	if filter.IdempotencyKeyHash != nil {
		data["idempotency_key_hash"] = *filter.IdempotencyKeyHash
		wc = append(wc, "idempotency_key_hash = :idempotency_key_hash")
	}

	if len(wc) > 0 {
		buf.WriteString(" WHERE ")
		buf.WriteString(strings.Join(wc, " AND "))
	}
}
