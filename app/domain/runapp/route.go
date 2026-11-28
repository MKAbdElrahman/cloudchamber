package runapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"github.com/google/uuid"
)

// Runs is what the handlers need from the run domain. They create runs, edit
// specs and read. They cannot write a status or a step.
type Runs interface {
	runbus.Commander
	runbus.Reader
}

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log        *logger.Logger
	Runs       Runs
	AuthClient authclient.Authenticator
	// Workflows are the workflows a caller may start a run of.
	Workflows *workflows.Registry
	// Nudge tells the controller a run changed. It must not block.
	Nudge func(runID uuid.UUID)
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.AuthClient)
	ruleAny := mid.Authorize(cfg.AuthClient, auth.RuleAny)
	ruleRun := mid.AuthorizeRun(cfg.AuthClient, cfg.Runs)

	api := newApp(cfg)

	app.HandlerFunc(http.MethodGet, version, "/workflows", api.queryWorkflows, authen, ruleAny)
	app.HandlerFunc(http.MethodGet, version, "/runs", api.query, authen, ruleAny)
	app.HandlerFunc(http.MethodPost, version, "/runs", api.create, authen, ruleAny)
	app.HandlerFunc(http.MethodGet, version, "/runs/{run_id}", api.queryByID, authen, ruleRun)
	app.HandlerFunc(http.MethodGet, version, "/runs/{run_id}/events", api.queryEvents, authen, ruleRun)
	app.HandlerFunc(http.MethodPost, version, "/runs/{run_id}/suspend", api.command(runbus.Suspend), authen, ruleRun)
	app.HandlerFunc(http.MethodPost, version, "/runs/{run_id}/resume", api.command(runbus.Resume), authen, ruleRun)
	app.HandlerFunc(http.MethodPost, version, "/runs/{run_id}/cancel", api.command(runbus.Cancel), authen, ruleRun)
	app.HandlerFunc(http.MethodPost, version, "/runs/{run_id}/retry", api.command(runbus.Retry), authen, ruleRun)
	app.HandlerFunc(http.MethodPost, version, "/runs/{run_id}/restart", api.command(runbus.Restart), authen, ruleRun)
}
