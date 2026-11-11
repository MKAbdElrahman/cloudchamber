//go:build !crud && !reporting

// Package build manages different build options.
package build

import (
	"github.com/MKAbdElrahman/cloudchamber/app/domain/artifactapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/auditapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/checkapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/homeapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelinejobapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/productapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/rawapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/tranapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/tusdapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/userapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/vproductapp"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mux"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Routes binds all the routes for the cloudchamber service.
func Routes() all {
	return all{}
}

type all struct{}

// Add implements the RouterAdder interface.
func (all) Add(app *web.App, cfg mux.Config) {
	checkapp.Routes(app, checkapp.Config{
		Build: cfg.Build,
		Log:   cfg.Log,
		DB:    cfg.DB,
	})

	homeapp.Routes(app, homeapp.Config{
		Log:        cfg.Log,
		HomeBus:    cfg.BusConfig.HomeBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	productapp.Routes(app, productapp.Config{
		Log:        cfg.Log,
		ProductBus: cfg.BusConfig.ProductBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	rawapp.Routes(app)

	tranapp.Routes(app, tranapp.Config{
		Log:        cfg.Log,
		DB:         cfg.DB,
		UserBus:    cfg.BusConfig.UserBus,
		ProductBus: cfg.BusConfig.ProductBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	userapp.Routes(app, userapp.Config{
		Log:        cfg.Log,
		UserBus:    cfg.BusConfig.UserBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	auditapp.Routes(app, auditapp.Config{
		Log:        cfg.Log,
		AuditBus:   cfg.BusConfig.AuditBus,
		AuthClient: cfg.CloudChamberConfig.AuthClient,
	})

	vproductapp.Routes(app, vproductapp.Config{
		Log:         cfg.Log,
		UserBus:     cfg.BusConfig.UserBus,
		VProductBus: cfg.BusConfig.VProductBus,
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
		AuthClient:  cfg.CloudChamberConfig.AuthClient,
	})

	tusdapp.Routes(app, tusdapp.Config{
		AuthClient:  cfg.CloudChamberConfig.AuthClient,
		UploadBus:   cfg.BusConfig.UploadBus,
		ArtifactBus: cfg.BusConfig.ArtifactBus,
		ObjectStore: cfg.CloudChamberConfig.ObjectStore,
	})
}
