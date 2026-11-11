package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/ardanlabs/conf/v3"
	"go.temporal.io/sdk/activity"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"

	pipelinesim "github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/activities/sim"
	pipelinewf "github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/workflow"
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

	traceIDFn := func(ctx context.Context) string {
		return otel.GetTraceID(ctx)
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "PIPELINE-RENDER", traceIDFn, events)

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "msg", "startup failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0), "build", build)

	cfg := struct {
		conf.Version
		Temporal struct {
			HostPort  string `conf:"default:localhost:7233"`
			Namespace string `conf:"default:default"`
		}
		Tempo struct {
			Host        string  `conf:"default:tempo:4317"`
			ServiceName string  `conf:"default:pipeline-render"`
			Probability float64 `conf:"default:1.0"`
		}
	}{
		Version: conf.Version{
			Build: build,
			Desc:  "CloudChamber pipeline render worker",
		},
	}

	const prefix = "PIPELINE"
	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if err == conf.ErrHelpWanted {
			fmt.Println(help)
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

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

	acts, err := pipelinesim.Open(ctx, log)
	if err != nil {
		return err
	}
	w := worker.New(c, pipelinewf.TaskQueueRender, worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{tracingInterceptor},
	})
	w.RegisterActivityWithOptions(acts.ComposeExport, activity.RegisterOptions{Name: pipelinewf.ActivityComposeExport})

	log.Info(ctx, "startup", "status", "worker started", "taskQueue", pipelinewf.TaskQueueRender, "hostPort", cfg.Temporal.HostPort, "namespace", cfg.Temporal.Namespace)

	if err := w.Run(worker.InterruptCh()); err != nil {
		return fmt.Errorf("worker run: %w", err)
	}

	log.Info(ctx, "shutdown", "status", "render worker stopped")
	return nil
}
