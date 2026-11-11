package apitest

import (
	"net/http/httptest"
	"testing"

	authbuild "github.com/MKAbdElrahman/cloudchamber/api/services/auth/build"
	apibuild "github.com/MKAbdElrahman/cloudchamber/api/services/cloudchamber/build"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/auth"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient/http"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mux"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/dbtest"
)

// New initialized the system to run a test.
func New(t *testing.T, testName string) *Test {
	db := dbtest.New(t, testName)

	// -------------------------------------------------------------------------

	auth := auth.New(auth.Config{
		Log:       db.Log,
		UserBus:   db.BusDomain.User,
		KeyLookup: &KeyStore{},
	})

	// -------------------------------------------------------------------------

	server := httptest.NewServer(mux.WebAPI(mux.Config{
		Log: db.Log,
		DB:  db.DB,
		BusConfig: mux.BusConfig{
			UserBus: db.BusDomain.User,
		},
		AuthConfig: mux.AuthConfig{
			Auth: auth,
		},
	}, authbuild.Routes()))

	authClient, err := http.New(db.Log, server.URL)
	if err != nil {
		t.Fatal("could not create authentication client")
	}

	// -------------------------------------------------------------------------

	mux := mux.WebAPI(mux.Config{
		Log: db.Log,
		DB:  db.DB,
		BusConfig: mux.BusConfig{
			AuditBus:    db.BusDomain.Audit,
			UserBus:     db.BusDomain.User,
			ProductBus:  db.BusDomain.Product,
			VProductBus: db.BusDomain.VProduct,
		},
		CloudChamberConfig: mux.CloudChamberConfig{
			AuthClient: authClient,
		},
	}, apibuild.Routes())

	return &Test{
		DB:   db,
		Auth: auth,
		mux:  mux,
	}
}
