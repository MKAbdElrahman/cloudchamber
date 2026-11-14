package pipelinejobapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/messagedb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
	"go.temporal.io/sdk/client"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log            *logger.Logger
	PipelineJobBus pipelinejobbus.ExtBusiness
	ArtifactBus    artifactbus.ExtBusiness
	AuthClient     authclient.Authenticator
	TemporalClient client.Client
	Hub            *messagedb.Hub
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.AuthClient)
	keep := mid.AuthenticateKeep(cfg.AuthClient)
	ruleAny := mid.Authorize(cfg.AuthClient, auth.RuleAny)
	rulePipelineJob := mid.AuthorizePipeline(cfg.AuthClient, cfg.PipelineJobBus)
	rulePipelineKeep := mid.AuthorizePipelineKeep(cfg.AuthClient, cfg.PipelineJobBus)

	api := newApp(cfg.Log, cfg.PipelineJobBus, cfg.ArtifactBus, cfg.TemporalClient, cfg.Hub)

	app.HandlerFunc(http.MethodGet, version, "/pipeline/jobs", api.query, authen, ruleAny)
	app.HandlerFunc(http.MethodGet, version, "/pipeline/jobs/{job_id}", api.queryByID, authen, rulePipelineJob)
	app.HandlerFunc(http.MethodGet, version, "/pipeline/jobs/{job_id}/facts", api.followJob, keep, rulePipelineKeep)
	app.HandlerFunc(http.MethodPost, version, "/pipeline/jobs", api.create, authen, ruleAny)
	app.HandlerFunc(http.MethodPost, version, "/pipeline/jobs/{job_id}/cancel", api.cancel, authen, rulePipelineJob)
	app.HandlerFunc(http.MethodGet, version, "/pipeline/facts", api.followProject, keep)
}
