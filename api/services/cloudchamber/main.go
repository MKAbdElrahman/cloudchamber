package main

import (
	"context"
	"embed"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/api/services/cloudchamber/build"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient/grpc"
	http2 "github.com/MKAbdElrahman/cloudchamber/app/sdk/authclient/http"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/debug"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/mux"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	apptemporal "github.com/MKAbdElrahman/cloudchamber/app/sdk/temporal"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/extensions/artifactotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus/stores/artifactdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus/extensions/auditotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/auditbus/stores/auditdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus/stores/pipelinejobpg"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/productbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/productbus/extensions/productotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/productbus/stores/productpg"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus/extensions/projectotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/projectbus/stores/projectdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus/stores/uploaddb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/extensions/useraudit"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/extensions/userotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/stores/usercache"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/userbus/stores/userdb"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/vproductbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/vproductbus/extensions/vproductotel"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/vproductbus/stores/vproductdb"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/delegate"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/MKAbdElrahman/cloudchamber/foundation/otel"
	"github.com/ardanlabs/conf/v3"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
)

/*
	Need to figure out timeouts for http service.
*/

//go:embed static
var static embed.FS

var tag = "develop"

func main() {
	var log *logger.Logger

	events := logger.Events{
		Error: func(ctx context.Context, r logger.Record) {
			log.Info(ctx, "******* SEND ALERT *******")
		},
	}

	log = logger.NewWithEvents(os.Stdout, logger.LevelInfo, "CLOUDCHAMBER", otel.GetTraceID, events)

	// -------------------------------------------------------------------------

	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		conf.Version
		Web struct {
			ReadTimeout        time.Duration `conf:"default:5s"`
			WriteTimeout       time.Duration `conf:"default:10s"`
			IdleTimeout        time.Duration `conf:"default:120s"`
			ShutdownTimeout    time.Duration `conf:"default:20s"`
			APIHost            string        `conf:"default:0.0.0.0:3000"`
			DebugHost          string        `conf:"default:0.0.0.0:3010"`
			CORSAllowedOrigins []string      `conf:"default:*"`
		}
		Auth struct {
			Host      string `conf:"default:http://auth-service:6000"`
			Mechanism string `conf:"default:http"`
			GRPC      struct {
				Host string `conf:"default:auth-service:6001"`
			}
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
		Tempo struct {
			Host        string  `conf:"default:tempo:4317"`
			ServiceName string  `conf:"default:cloudchamber"`
			Probability float64 `conf:"default:0.05"`
			// Shouldn't use a high Probability value in non-developer systems.
			// 0.05 should be enough for most systems. Some might want to have
			// this even lower.
		}
		Temporal struct {
			HostPort  string `conf:"default:localhost:7233"`
			Namespace string `conf:"default:default"`
		}
		ObjectStore struct {
			Endpoint       string `conf:"default:http://seaweedfs-service:8333"`
			PublicEndpoint string `conf:"default:http://127.0.0.1:8333"`
			Bucket         string `conf:"default:cloudchamber-artifacts"`
			Region         string `conf:"default:us-east-1"`
			AccessKey      string `conf:"default:admin,mask"`
			SecretKey      string `conf:"default:seaweedfs-dev,mask"`
			UsePathStyle   bool   `conf:"default:true"`
		}
	}{
		Build: tag,
		Desc:  "CloudChamber",
	}

	const prefix = "CLOUDCHAMBER"
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

	expvar.NewString("build").Set(cfg.Build)

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
	// Create Business Packages

	delegate := delegate.New(log)

	auditOtelExt := auditotel.NewExtension()
	auditStorage := auditdb.NewStore(log, db)
	auditBus := auditbus.NewBusiness(log, auditStorage, auditOtelExt)

	userOtelExt := userotel.NewExtension()
	userAuditExt := useraudit.NewExtension(auditBus)
	userStorage := usercache.NewStore(log, userdb.NewStore(log, db), time.Minute)
	userBus := userbus.NewBusiness(log, delegate, userStorage, userOtelExt, userAuditExt)

	productOtelExt := productotel.NewExtension()
	productStorage := productpg.NewStore(log, db)
	productBus := productbus.NewBusiness(log, userBus, delegate, productStorage, productOtelExt)

	projectOtelExt := projectotel.NewExtension()
	projectStorage := projectdb.NewStore(log, db)
	projectBus := projectbus.NewBusiness(projectStorage, projectOtelExt)

	vproductOtelExt := vproductotel.NewExtension()
	vproductStorage := vproductdb.NewStore(log, db)
	vproductBus := vproductbus.NewBusiness(vproductStorage, vproductOtelExt)

	pipelineJobStorage := pipelinejobpg.NewStore(log, db)
	pipelineJobBus := pipelinejobbus.NewBusiness(log, pipelineJobStorage)

	uploadStorage := uploaddb.NewStore(log, db)
	uploadBus := uploadbus.NewBusiness(log, uploadStorage)

	// -------------------------------------------------------------------------
	// Initialize authentication support

	log.Info(ctx, "startup", "status", "initializing authentication support")

	var authClient authclient.Authenticator
	switch cfg.Auth.Mechanism {
	case "grpc":
		authClient, err = grpc.New(log, cfg.Auth.GRPC.Host)
	default:
		authClient, err = http2.New(log, cfg.Auth.Host)
	}

	if err != nil {
		log.Error(ctx, "failed to initialize authentication client", "error", err)
		return fmt.Errorf("failed to initialize authentication client: %w", err)
	}

	defer authClient.Close()

	// -------------------------------------------------------------------------
	// Start Tracing Support

	log.Info(ctx, "startup", "status", "initializing tracing support")

	traceProvider, teardown, err := otel.InitTracing(otel.Config{
		ServiceName: cfg.Tempo.ServiceName,
		Host:        cfg.Tempo.Host,
		ExcludedRoutes: map[string]struct{}{
			"/v1/liveness":  {},
			"/v1/readiness": {},
		},
		Probability: cfg.Tempo.Probability,
	})
	if err != nil {
		return fmt.Errorf("starting tracing: %w", err)
	}

	defer teardown(context.Background())

	tracer := traceProvider.Tracer(cfg.Tempo.ServiceName)

	// -------------------------------------------------------------------------
	// Initialize Temporal support

	log.Info(ctx, "startup", "status", "initializing temporal support", "hostport", cfg.Temporal.HostPort, "namespace", cfg.Temporal.Namespace)

	temporalTracingInterceptor, err := temporalotel.NewTracingInterceptor(temporalotel.TracerOptions{
		Tracer: tracer,
	})
	if err != nil {
		return fmt.Errorf("creating temporal tracing interceptor: %w", err)
	}

	temporalClient, err := apptemporal.NewClient(apptemporal.Config{
		HostPort:     cfg.Temporal.HostPort,
		Namespace:    cfg.Temporal.Namespace,
		Interceptors: []interceptor.ClientInterceptor{temporalTracingInterceptor},
	})
	if err != nil {
		return fmt.Errorf("connecting to temporal: %w", err)
	}

	defer temporalClient.Close()

	// -------------------------------------------------------------------------
	// Start Debug Service

	go func() {
		log.Info(ctx, "startup", "status", "debug v1 router started", "host", cfg.Web.DebugHost)

		if err := http.ListenAndServe(cfg.Web.DebugHost, debug.Mux()); err != nil {
			log.Error(ctx, "shutdown", "status", "debug v1 router closed", "host", cfg.Web.DebugHost, "msg", err)
		}
	}()

	// -------------------------------------------------------------------------
	// Start API Service

	log.Info(ctx, "startup", "status", "initializing V1 API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	objectStore, err := objectstore.New(ctx, objectstore.Config{
		Endpoint:       cfg.ObjectStore.Endpoint,
		PublicEndpoint: cfg.ObjectStore.PublicEndpoint,
		Bucket:         cfg.ObjectStore.Bucket,
		Region:         cfg.ObjectStore.Region,
		AccessKey:      cfg.ObjectStore.AccessKey,
		SecretKey:      cfg.ObjectStore.SecretKey,
		UsePathStyle:   cfg.ObjectStore.UsePathStyle,
	})
	if err != nil {
		return fmt.Errorf("initializing object store: %w", err)
	}

	artifactOtelExt := artifactotel.NewExtension()
	artifactStorage := artifactdb.NewStore(log, db)
	artifactBus := artifactbus.NewBusiness(log, artifactStorage, objectStore, artifactOtelExt)

	cfgMux := mux.Config{
		Build:  tag,
		Log:    log,
		DB:     db,
		Tracer: tracer,
		BusConfig: mux.BusConfig{
			AuditBus:       auditBus,
			UserBus:        userBus,
			ProductBus:     productBus,
			ProjectBus:     projectBus,
			VProductBus:    vproductBus,
			PipelineJobBus: pipelineJobBus,
			UploadBus:      uploadBus,
			ArtifactBus:    artifactBus,
		},
		CloudChamberConfig: mux.CloudChamberConfig{
			AuthClient:        authClient,
			TemporalClient:    temporalClient,
			TemporalNamespace: cfg.Temporal.Namespace,
			ObjectStore:       objectStore,
		},
	}

	webAPI := mux.WebAPI(cfgMux,
		build.Routes(),
		mux.WithCORS(cfg.Web.CORSAllowedOrigins),
		mux.WithFileServer(false, static, "static", "/"),
	)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      webAPI,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)

		serverErrors <- api.ListenAndServe()
	}()

	// -------------------------------------------------------------------------
	// Shutdown

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
