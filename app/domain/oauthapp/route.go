package oauthapp

import (
	"net/http"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/web"
)

// Config contains all the configuration for Google sign-in.
type Config struct {
	Auth               *auth.Auth
	Log                *logger.Logger
	UserBus            userbus.ExtBusiness
	TokenKey           string
	GoogleKey          string
	GoogleSecret       string
	GoogleCallbackBase string
	BubbleURL          string
	SessionSecret      string
}

// Routes adds the routes for Google sign-in.
func Routes(app *web.App, cfg Config) {
	api := newApp(cfg)

	app.HandlerFunc(http.MethodGet, "", "/api/auth/{provider}", api.authenticate)
	app.HandlerFunc(http.MethodGet, "", "/api/logout/{provider}", api.logout)
	app.HandlerFunc(http.MethodGet, "", "/api/auth/{provider}/callback", api.authCallback)
}
