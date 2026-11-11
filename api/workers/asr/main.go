package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/ardanlabs/conf/v3"
	"go.temporal.io/sdk/activity"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/activities/sim"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	pipelinewf "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	temporalclient "github.com/MKAbdElrahman/cloudchamber/app/sdk/temporal"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/extensions/artifactotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/stores/artifactdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus/extensions/pipelinejobotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus/stores/pipelinejobpg"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
)

var build = "develop"

func main() {
	var log *logger.Logger

	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "PIPELINE-ASR", otel.GetTraceID, events)

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0), "build", build)

	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		Temporal struct {
			HostPort  string `conf:"default:localhost:7233"`
			Namespace string `conf:"default:default"`
		}
		DB struct {
			User         string `conf:"default:postgres"`
			Password     string `conf:"default:postgres,mask"`
			Host         string `conf:"default:database-service"`
			Name         string `conf:"default:postgres"`
			MaxIdleConns int    `conf:"default:0"`
			MaxOpenConns int    `conf:"default:0"`
			DisableTLS   bool   `conf:"default:true"`
		}
		ObjectStore struct {
			Endpoint     string `conf:"default:http://seaweedfs-service:8333"`
			Bucket       string `conf:"default:cloudchamber-artifacts"`
			Region       string `conf:"default:us-east-1"`
			AccessKey    string `conf:"default:admin,mask"`
			SecretKey    string `conf:"default:seaweedfs-dev,mask"`
			UsePathStyle bool   `conf:"default:true"`
		}
		Tempo struct {
			Host        string  `conf:"default:tempo:4317"`
			ServiceName string  `conf:"default:pipeline-asr"`
			Probability float64 `conf:"default:1.0"`
		}
	}{
		Build: build,
		Desc:  "CloudChamber pipeline ASR worker",
	}

	const prefix = "PIPELINE"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(ctx, "starting service", "version", cfg.Build)
	defer log.Info(ctx, "shutdown complete")

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %w", err)
	}
	log.Info(ctx, "startup", "config", out)

	log.BuildInfo(ctx)

	// -------------------------------------------------------------------------
	// Database Support

	log.Info(ctx, "startup", "status", "initializing database support", "hostport", cfg.DB.Host)

	db, err := sqldb.Open(sqldb.Config{
		User:         cfg.DB.User,
		Password:     cfg.DB.Password,
		Host:         cfg.DB.Host,
		Name:         cfg.DB.Name,
		MaxIdleConns: cfg.DB.MaxIdleConns,
		MaxOpenConns: cfg.DB.MaxOpenConns,
		DisableTLS:   cfg.DB.DisableTLS,
	})
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}

	defer db.Close()

	// -------------------------------------------------------------------------
	// Start Tracing Support

	log.Info(ctx, "startup", "status", "initializing tracing support")

	traceProvider, teardown, err := otel.InitTracing(otel.Config{
		ServiceName: cfg.Tempo.ServiceName,
		Host:        cfg.Tempo.Host,
		Probability: cfg.Tempo.Probability,
	})
	if err != nil {
		return fmt.Errorf("init tracing: %w", err)
	}
	defer teardown(context.Background())

	tracer := traceProvider.Tracer(cfg.Tempo.ServiceName)

	// -------------------------------------------------------------------------
	// Create Business Packages

	objects, err := objectstore.New(ctx, objectstore.Config{
		Endpoint:     cfg.ObjectStore.Endpoint,
		Bucket:       cfg.ObjectStore.Bucket,
		Region:       cfg.ObjectStore.Region,
		AccessKey:    cfg.ObjectStore.AccessKey,
		SecretKey:    cfg.ObjectStore.SecretKey,
		UsePathStyle: cfg.ObjectStore.UsePathStyle,
	})
	if err != nil {
		return fmt.Errorf("initializing object store: %w", err)
	}

	artifactOtelExt := artifactotel.NewExtension()
	artifactStorage := artifactdb.NewStore(log, db)
	artifactBus := artifactbus.NewBusiness(log, artifactStorage, objects, artifactOtelExt)

	pipelineJobOtelExt := pipelinejobotel.NewExtension()
	pipelineJobStorage := pipelinejobpg.NewStore(log, db)
	pipelineJobBus := pipelinejobbus.NewBusiness(log, pipelineJobStorage, pipelineJobOtelExt)

	acts := sim.New(artifactBus, pipelineJobBus, objects, tracer)

	tracingInterceptor, err := temporalotel.NewTracingInterceptor(temporalotel.TracerOptions{
		Tracer: tracer,
	})
	if err != nil {
		return fmt.Errorf("create temporal tracing interceptor: %w", err)
	}

	c, err := temporalclient.NewClient(temporalclient.Config{
		HostPort:     cfg.Temporal.HostPort,
		Namespace:    cfg.Temporal.Namespace,
		Interceptors: []interceptor.ClientInterceptor{tracingInterceptor},
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// The client copies interceptors that implement WorkerInterceptor onto the
	// worker. Passing the tracing interceptor here as well records every span twice.
	w := worker.New(c, pipelinewf.TaskQueueASR, worker.Options{})
	w.RegisterActivityWithOptions(acts.Transcribe, activity.RegisterOptions{Name: pipelinewf.ActivityTranscribe})

	log.Info(ctx, "startup", "status", "worker started", "taskQueue", pipelinewf.TaskQueueASR, "hostPort", cfg.Temporal.HostPort, "namespace", cfg.Temporal.Namespace)

	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("worker run: %w", err)
	}

	log.Info(ctx, "shutdown", "status", "ASR worker stopped")
	return nil
}
