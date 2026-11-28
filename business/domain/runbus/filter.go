package runbus

import (
	"github.com/google/uuid"

	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
)

// QueryFilter holds the available fields a query can be filtered on.
// Pointer fields distinguish a value that was not provided.
type QueryFilter struct {
	ID                 *uuid.UUID
	OwnerID            *uuid.UUID
	ProjectID          *uuid.UUID
	Workflow           *workflowname.WorkflowName
	Phase              *phase.Phase
	IdempotencyKeyHash *string
}
