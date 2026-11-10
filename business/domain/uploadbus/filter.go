package uploadbus

import "github.com/google/uuid"

// QueryFilter holds the available fields a query can be filtered on.
type QueryFilter struct {
	ObjectKey *string
	OwnerID   *uuid.UUID
	Status    *Status
	SourceRef *string
}
