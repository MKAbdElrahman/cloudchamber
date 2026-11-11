package sim

import (
	"context"
	"fmt"

	"github.com/ardanlabs/conf/v3"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/extensions/artifactotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/stores/artifactdb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
)

// Open connects the worker to the database and the object store.
func Open(ctx context.Context, log *logger.Logger) (*Activities, error) {
	cfg := struct {
		DB struct {
			User       string `conf:"default:postgres"`
			Password   string `conf:"default:postgres,mask"`
			Host       string `conf:"default:database-service"`
			Name       string `conf:"default:postgres"`
			DisableTLS bool   `conf:"default:true"`
		}
		ObjectStore struct {
			Endpoint     string `conf:"default:http://seaweedfs-service:8333"`
			Bucket       string `conf:"default:cloudchamber-artifacts"`
			Region       string `conf:"default:us-east-1"`
			AccessKey    string `conf:"default:admin,mask"`
			SecretKey    string `conf:"default:seaweedfs-dev,mask"`
			UsePathStyle bool   `conf:"default:true"`
		}
	}{}

	if _, err := conf.Parse("PIPELINE", &cfg); err != nil {
		return nil, fmt.Errorf("parse worker config: %w", err)
	}

	db, err := sqldb.Open(sqldb.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Host:       cfg.DB.Host,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	objects, err := objectstore.New(ctx, objectstore.Config{
		Endpoint:     cfg.ObjectStore.Endpoint,
		Bucket:       cfg.ObjectStore.Bucket,
		Region:       cfg.ObjectStore.Region,
		AccessKey:    cfg.ObjectStore.AccessKey,
		SecretKey:    cfg.ObjectStore.SecretKey,
		UsePathStyle: cfg.ObjectStore.UsePathStyle,
	})
	if err != nil {
		return nil, fmt.Errorf("open object store: %w", err)
	}

	store := artifactdb.NewStore(log, db)
	bus := artifactbus.NewBusiness(log, store, objects, artifactotel.NewExtension())
	return New(bus, objects), nil
}
