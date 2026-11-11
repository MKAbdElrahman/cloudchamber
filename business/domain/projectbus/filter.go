package projectbus

import (
	"github.com/MKAbdElrahman/cloudchamber/business/types/title"
	"github.com/google/uuid"
)

// QueryFilter holds the available fields a query can be filtered on.
// Pointer fields distinguish a value that was not provided.
type QueryFilter struct {
	ID      *uuid.UUID
	OwnerID *uuid.UUID
	Name    *title.Title
}
