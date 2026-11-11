package artifactapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mid"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	Log         *logger.Logger
	ArtifactBus artifactbus.ExtBusiness
	UploadBus   uploadbus.ExtBusiness
	ObjectStore *objectstore.Client
	AuthClient  authclient.Authenticator
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	authen := mid.Authenticate(cfg.AuthClient)
	ruleAny := mid.Authorize(cfg.AuthClient, auth.RuleAny)

	api := newApp(cfg.Log, cfg.ArtifactBus, cfg.UploadBus, cfg.ObjectStore)

	app.HandlerFunc(http.MethodGet, version, "/files/{file_id}/url", api.readURL, authen, ruleAny)
	app.HandlerFunc(http.MethodDelete, version, "/files/{file_id}", api.delete, authen, ruleAny)
}
