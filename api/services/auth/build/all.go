// Package build manages different build options.
package build

import (
	"github.com/MKAbdElrahman/cloudchamber/app/domain/authapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/checkapp"
	"github.com/MKAbdElrahman/cloudchamber/app/domain/oauthapp"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mux"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Routes binds all the routes for the auth service.
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

	authapp.Routes(app, authapp.Config{
		UserBus: cfg.BusConfig.UserBus,
		Auth:    cfg.AuthConfig.Auth,
	})

	oauthapp.Routes(app, oauthapp.Config{
		Auth:               cfg.AuthConfig.Auth,
		Log:                cfg.Log,
		UserBus:            cfg.BusConfig.UserBus,
		TokenKey:           cfg.AuthConfig.ActiveKID,
		GoogleKey:          cfg.AuthConfig.GoogleKey,
		GoogleSecret:       cfg.AuthConfig.GoogleSecret,
		GoogleCallbackBase: cfg.AuthConfig.GoogleCallbackBase,
		BubbleURL:          cfg.AuthConfig.BubbleURL,
		SessionSecret:      cfg.AuthConfig.SessionSecret,
	})
}
