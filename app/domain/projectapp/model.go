package projectapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/title"
	"github.com/google/uuid"
)

// Project is one project returned to the client.
type Project struct {
	ID          string `json:"id"`
	OwnerID     string `json:"ownerID"`
	Name        string `json:"name"`
	DateCreated string `json:"dateCreated"`
	DateUpdated string `json:"dateUpdated"`
}

// Encode implements the encoder interface.
func (app Project) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}

func toAppProject(prj projectbus.Project) Project {
	return Project{
		ID:          prj.ID.String(),
		OwnerID:     prj.OwnerID.String(),
		Name:        prj.Name.String(),
		DateCreated: prj.DateCreated.Format(time.RFC3339),
		DateUpdated: prj.DateUpdated.Format(time.RFC3339),
	}
}

func toAppProjects(prjs []projectbus.Project) []Project {
	app := make([]Project, len(prjs))
	for i, prj := range prjs {
		app[i] = toAppProject(prj)
	}

	return app
}

// =============================================================================

// NewProject is the data needed to add a project. The owner is the caller.
type NewProject struct {
	Name string `json:"name"`
}

// Decode implements the decoder interface.
func (app *NewProject) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

func toBusNewProject(ctx context.Context, app NewProject) (projectbus.NewProject, error) {
	var fieldErrs errs.FieldErrors

	ownerID, err := mid.GetUserID(ctx)
	if err != nil {
		fieldErrs.Add("ownerID", err)
	}

	name, err := title.Parse(app.Name)
	if err != nil {
		fieldErrs.Add("name", err)
	}

	if len(fieldErrs) > 0 {
		return projectbus.NewProject{}, fmt.Errorf("validate: %w", fieldErrs.ToError())
	}

	return projectbus.NewProject{
		OwnerID: ownerID,
		Name:    name,
	}, nil
}

// =============================================================================

// UpdateProject is the data needed to change a project.
type UpdateProject struct {
	Name *string `json:"name"`
}

// Decode implements the decoder interface.
func (app *UpdateProject) Decode(data []byte) error {
	return json.Unmarshal(data, app)
}

func toBusUpdateProject(app UpdateProject) (projectbus.UpdateProject, error) {
	var fieldErrs errs.FieldErrors

	var name *title.Title
	if app.Name != nil {
		parsed, err := title.Parse(*app.Name)
		if err != nil {
			fieldErrs.Add("name", err)
		}
		name = &parsed
	}

	if len(fieldErrs) > 0 {
		return projectbus.UpdateProject{}, fmt.Errorf("validate: %w", fieldErrs.ToError())
	}

	return projectbus.UpdateProject{Name: name}, nil
}

// ProjectFile is one file in a project.
type ProjectFile struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
	DateCreated string `json:"dateCreated"`
}

// ProjectFiles is the file list for one project.
type ProjectFiles struct {
	Items []ProjectFile `json:"items"`
}

// Encode implements the encoder interface.
func (app ProjectFiles) Encode() ([]byte, string, error) {
	data, err := json.Marshal(app)
	return data, "application/json", err
}

func toAppFiles(arts []artifactbus.Artifact) ProjectFiles {
	items := make([]ProjectFile, len(arts))
	for i, art := range arts {
		projectID := ""
		if art.ProjectID != uuid.Nil {
			projectID = art.ProjectID.String()
		}
		items[i] = ProjectFile{
			ID:          art.FileID.String(),
			ProjectID:   projectID,
			Name:        art.Name,
			Kind:        art.Kind.String(),
			ContentType: art.ContentType,
			DateCreated: art.DateCreated.UTC().Format(time.RFC3339),
		}
	}
	return ProjectFiles{Items: items}
}
