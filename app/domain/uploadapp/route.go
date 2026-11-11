package uploadapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	AuthClient  authclient.Authenticator
	UploadBus   uploadbus.ExtBusiness
	ArtifactBus artifactbus.ExtBusiness
	ProjectBus  projectbus.ExtBusiness
	ObjectStore *objectstore.Client
	DB          sqldb.Beginner
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	api := newApp(cfg.AuthClient, cfg.UploadBus, cfg.ArtifactBus, cfg.ProjectBus, cfg.ObjectStore, cfg.DB)

	// The same application middleware as every other route. mid.Otel stores the
	// tracer that business spans use. The hook still does its own authentication.
	app.HandlerFunc(http.MethodPost, version, "/internal/tusd/hooks", api.handle)
}
