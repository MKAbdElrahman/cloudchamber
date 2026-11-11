package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/ardanlabs/conf/v3"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	pipelinewf "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	temporalclient "github.com/MKAbdElrahman/cloudchamber/app/sdk/temporal"
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

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "PIPELINE-WORKFLOW", otel.GetTraceID, events)

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
		Tempo struct {
			Host        string  `conf:"default:tempo:4317"`
			ServiceName string  `conf:"default:pipeline-workflow"`
			Probability float64 `conf:"default:1.0"`
		}
	}{
		Build: build,
		Desc:  "CloudChamber pipeline workflow worker",
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
	w := worker.New(c, pipelinewf.TaskQueueWorkflow, worker.Options{})
	w.RegisterWorkflow(pipelinewf.ProcessSourceWorkflow)

	log.Info(ctx, "startup", "status", "worker started", "taskQueue", pipelinewf.TaskQueueWorkflow, "hostPort", cfg.Temporal.HostPort, "namespace", cfg.Temporal.Namespace)

	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("worker run: %w", err)
	}

	log.Info(ctx, "shutdown", "status", "workflow worker stopped")
	return nil
}
