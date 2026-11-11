package projectapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log         *logger.Logger
	ProjectBus  projectbus.ExtBusiness
	ArtifactBus artifactbus.ExtBusiness
	AuthClient  authclient.Authenticator
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.AuthClient)
	ruleAny := mid.Authorize(cfg.AuthClient, auth.RuleAny)
	ruleAuthorizeProject := mid.AuthorizeProject(cfg.AuthClient, cfg.ProjectBus)

	api := newApp(cfg.ProjectBus, cfg.ArtifactBus)

	app.HandlerFunc(http.MethodGet, version, "/projects", api.query, authen, ruleAny)
	app.HandlerFunc(http.MethodGet, version, "/projects/{project_id}", api.queryByID, authen, ruleAuthorizeProject)
	app.HandlerFunc(http.MethodGet, version, "/projects/{project_id}/files", api.queryFiles, authen, ruleAuthorizeProject)
	app.HandlerFunc(http.MethodPost, version, "/projects", api.create, authen, ruleAny)
	app.HandlerFunc(http.MethodPut, version, "/projects/{project_id}", api.update, authen, ruleAuthorizeProject)
	app.HandlerFunc(http.MethodDelete, version, "/projects/{project_id}", api.delete, authen, ruleAuthorizeProject)
}
