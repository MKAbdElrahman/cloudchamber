package tusdapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Config contains all the mandatory systems required by handlers.
type Config struct {
	AuthClient  authclient.Authenticator
	UploadBus   uploadbus.ExtBusiness
	ObjectStore *objectstore.Client
}

// Routes adds specific routes for this group.
func Routes(app *web.App, cfg Config) {
	const version = "v1"

	api := newApp(cfg.AuthClient, cfg.UploadBus, cfg.ObjectStore)

	app.HandlerFuncNoMid(http.MethodPost, version, "/internal/tusd/hooks", api.handle)
}
