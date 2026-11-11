package projectdb

import (
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/title"
	"github.com/google/uuid"
)

// ProjectDB is the database representation of a project row.
type ProjectDB struct {
	ID          uuid.UUID `db:"project_id"`
	OwnerID     uuid.UUID `db:"owner_id"`
	Name        string    `db:"name"`
	DateCreated time.Time `db:"date_created"`
	DateUpdated time.Time `db:"date_updated"`
}

// ToDBProject converts a business project into its database representation.
func ToDBProject(bus projectbus.Project) ProjectDB {
	return ProjectDB{
		ID:          bus.ID,
		OwnerID:     bus.OwnerID,
		Name:        bus.Name.String(),
		DateCreated: bus.DateCreated.UTC(),
		DateUpdated: bus.DateUpdated.UTC(),
	}
}

// ToBusProject converts a database row into a business project.
func ToBusProject(db ProjectDB) (projectbus.Project, error) {
	name, err := title.Parse(db.Name)
	if err != nil {
		return projectbus.Project{}, fmt.Errorf("parse name: %w", err)
	}

	return projectbus.Project{
		ID:          db.ID,
		OwnerID:     db.OwnerID,
		Name:        name,
		DateCreated: db.DateCreated.UTC(),
		DateUpdated: db.DateUpdated.UTC(),
	}, nil
}

// ToBusProjects converts database rows into business projects.
func ToBusProjects(dbs []ProjectDB) ([]projectbus.Project, error) {
	bus := make([]projectbus.Project, len(dbs))
	for i, db := range dbs {
		var err error
		bus[i], err = ToBusProject(db)
		if err != nil {
			return nil, err
		}
	}

	return bus, nil
}
