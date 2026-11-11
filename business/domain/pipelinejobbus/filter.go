package pipelinejobbus

import "github.com/google/uuid"

// QueryFilter holds the available fields a query can be filtered on.
type QueryFilter struct {
	ID                 *uuid.UUID
	OwnerID            *uuid.UUID
	ProjectID          *uuid.UUID
	WorkflowID         *string
	IdempotencyKeyHash *string
	Status             *Status
}
