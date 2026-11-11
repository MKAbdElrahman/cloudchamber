//go:build crud

package build

import (
	"github.com/MKAbdElrahman/cloudchamber/app/domain/artifactapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/auditapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/checkapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/homeapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelinejobapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/productapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/projectapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/tranapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/tusdapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/userapp"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mux"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Routes binds the crud routes for the cloudchamber service.
func Routes() crud {
	return crud{}
}

type crud struct{}

// Add implements the RouterAdder interface.
func (crud) Add(app *web.App, cfg mux.Config) {
	checkapp.Routes(app, checkapp.Config{
		Build: cfg.Build,
		Log:   cfg.Log,
		DB:    cfg.DB,
	})

	homeapp.Routes(app, homeapp.Config{
		HomeBus:    cfg.BusConfig.HomeBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	productapp.Routes(app, productapp.Config{
		ProductBus: cfg.BusConfig.ProductBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	projectapp.Routes(app, projectapp.Config{
		ProjectBus:  cfg.BusConfig.ProjectBus,
		ArtifactBus: cfg.BusConfig.ArtifactBus,
		AuthClient:  cfg.CloudChamberConfig.AuthClient,
	})

	pipelinejobapp.Routes(app, pipelinejobapp.Config{
		Log:               cfg.Log,
		PipelineJobBus:    cfg.BusConfig.PipelineJobBus,
		ArtifactBus:       cfg.BusConfig.ArtifactBus,
		AuthClient:        cfg.CloudChamberConfig.AuthClient,
		TemporalClient:    cfg.CloudChamberConfig.TemporalClient,
		TemporalNamespace: cfg.CloudChamberConfig.TemporalNamespace,
	})

	artifactapp.Routes(app, artifactapp.Config{
		Log:         cfg.Log,
		ArtifactBus: cfg.BusConfig.ArtifactBus,
		UploadBus:   cfg.BusConfig.UploadBus,
		ObjectStore: cfg.CloudChamberConfig.ObjectStore,
		AuthClient:  cfg.CloudChamberConfig.AuthClient,
	})

	tusdapp.Routes(app, tusdapp.Config{
		AuthClient:  cfg.CloudChamberConfig.AuthClient,
		UploadBus:   cfg.BusConfig.UploadBus,
		ArtifactBus: cfg.BusConfig.ArtifactBus,
		ProjectBus:  cfg.BusConfig.ProjectBus,
		ObjectStore: cfg.CloudChamberConfig.ObjectStore,
		DB:          sqldb.NewBeginner(cfg.DB),
	})

	tranapp.Routes(app, tranapp.Config{
		UserBus:    cfg.BusConfig.UserBus,
		ProductBus: cfg.BusConfig.ProductBus,
		Log:        cfg.Log,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
		DB:         cfg.DB,
	})

	userapp.Routes(app, userapp.Config{
		UserBus:    cfg.BusConfig.UserBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	auditapp.Routes(app, auditapp.Config{
		Log:        cfg.Log,
		AuditBus:   cfg.BusConfig.AuditBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})
}
