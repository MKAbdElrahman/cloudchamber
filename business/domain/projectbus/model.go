package projectbus

import (
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/types/title"
	"github.com/google/uuid"
)

// Project is one project owned by one user.
type Project struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Name        title.Title
	DateCreated time.Time
	DateUpdated time.Time
}

// NewProject is what we require when adding a project.
type NewProject struct {
	OwnerID uuid.UUID
	Name    title.Title
}

// UpdateProject is the set of fields that may change. A nil field is left as it is.
type UpdateProject struct {
	Name *title.Title
}
