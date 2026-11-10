package pipelinejobapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"go.temporal.io/sdk/client"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log               *logger.Logger
	PipelineJobBus    pipelinejobbus.ExtBusiness
	UploadBus         uploadbus.ExtBusiness
	AuthClient        authclient.Authenticator
	TemporalClient    client.Client
	TemporalNamespace string
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.AuthClient)
	ruleUser := mid.Authorize(cfg.AuthClient, auth.RuleUserOnly)
	rulePipelineJob := mid.AuthorizePipeline(cfg.AuthClient, cfg.PipelineJobBus)

	api := newApp(cfg.Log, cfg.PipelineJobBus, cfg.UploadBus, cfg.TemporalClient, cfg.TemporalNamespace)

	app.HandlerFunc(http.MethodPost, version, "/pipeline/jobs", api.create, authen, ruleUser)
	app.HandlerFunc(http.MethodGet, version, "/pipeline/jobs", api.query, authen, ruleUser)
	app.HandlerFunc(http.MethodGet, version, "/pipeline/jobs/{job_id}", api.queryByID, authen, rulePipelineJob)
	app.HandlerFunc(http.MethodPost, version, "/pipeline/jobs/{job_id}/cancel", api.cancel, authen, rulePipelineJob)
}
