package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/analysishandler"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/homerhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler"
	"monorepo/bin-timeline-manager/pkg/peereventhandler"
	"monorepo/bin-timeline-manager/pkg/siphandler"
	"monorepo/bin-timeline-manager/pkg/subscribehandler"
)

var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "timeline-manager",
		Short: "Voipbin Timeline Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.Bootstrap(rootCmd); errBind != nil {
		logrus.Fatalf("Failed to bootstrap config: %v", errBind)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		logrus.Errorf("Command execution failed: %v", errExecute)
		os.Exit(1)
	}
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting timeline-manager...")

	// Run database migrations before starting services
	if errMigrate := runMigrations(); errMigrate != nil {
		return errors.Wrapf(errMigrate, "could not run migrations")
	}

	// Create a context that cancels when the process receives a shutdown signal.
	// This allows the subscribe handler's flush worker to drain buffered events.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscribeDone, errStart := runServices(ctx)
	if errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	cancel()

	// Wait for the flush worker to finish draining buffered events.
	select {
	case <-subscribeDone:
		log.Info("Subscribe handler drain completed.")
	case <-time.After(15 * time.Second):
		log.Warn("Subscribe handler drain timed out after 15s.")
	}

	log.Info("Timeline-manager stopped safely.")
	return nil
}

func runMigrations() error {
	log := logrus.WithField("func", "runMigrations")

	cfg := config.Get()
	if cfg.ClickHouseAddress == "" {
		log.Warn("ClickHouse address not configured, skipping migrations")
		return nil
	}

	dsn := fmt.Sprintf("clickhouse://%s/%s?x-multi-statement=true", cfg.ClickHouseAddress, cfg.ClickHouseDatabase)
	sourceURL := fmt.Sprintf("file://%s", cfg.MigrationsPath)

	log.WithFields(logrus.Fields{
		"source": sourceURL,
		"dsn":    cfg.ClickHouseAddress,
	}).Info("Running database migrations...")

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return errors.Wrap(err, "failed to initialize migrate")
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return errors.Wrap(err, "migration failed")
	}

	version, dirty, _ := m.Version()
	log.WithFields(logrus.Fields{
		"version": version,
		"dirty":   dirty,
	}).Info("Database migrations completed")

	return nil
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

func initProm(endpoint, listen string) {
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics server disabled")
		return
	}

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func runServices(ctx context.Context) (<-chan struct{}, error) {
	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	evtHandler := eventhandler.NewEventHandler(db)
	peerEvtHandler := peereventhandler.NewPeerEventHandler(db)

	homerH := homerhandler.NewHomerHandler(config.Get().HomerAPIAddress, config.Get().HomerAuthToken)

	// Initialize GCS reader for RTP pcap fetching (optional).
	// Note: The GCS client is intentionally not closed. runServices() returns immediately
	// (listenHandler.Run() is non-blocking), so defer client.Close() would close prematurely.
	// The client lives for the process lifetime, matching other long-lived resources (sockHandler, db).
	var gcsReader siphandler.GCSReader
	gcsBucket := config.Get().GCSBucketName
	if gcsBucket != "" {
		client, err := storage.NewClient(context.Background())
		if err != nil {
			logrus.Warnf("Could not create GCS client, RTP pcap merge disabled: %v", err)
		} else {
			gcsReader = siphandler.NewGCSReader(client, gcsBucket)
			logrus.WithField("bucket", gcsBucket).Info("GCS reader initialized for RTP pcap merge.")
		}
	}

	sipH := siphandler.NewSIPHandler(homerH, gcsReader, gcsBucket)

	// Build the analysis handler (MySQL store + requesthandler client). It is
	// optional: if DATABASE_DSN is unset the analysis endpoints are not served
	// (timeline-manager keeps its read-only ClickHouse role).
	var analysisH analysishandler.AnalysisHandler
	cfg := config.Get()
	if cfg.DatabaseDSN != "" {
		sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
		if err != nil {
			return nil, errors.Wrapf(err, "could not connect to analysis MySQL store")
		}

		analysisDB := analysisdbhandler.NewAnalysisDBHandler(sqlDB)
		reqHandler := requesthandler.NewRequestHandler(sockHandler, commonoutline.ServiceNameTimelineManager)
		models := analysishandler.StageModels{
			Stage1: cfg.AnalysisModelStage1,
			Stage2: cfg.AnalysisModelStage2,
			Stage3: cfg.AnalysisModelStage3,
		}
		analysisH = analysishandler.NewAnalysisHandler(reqHandler, analysisDB, evtHandler, models)
		logrus.Info("Analysis handler initialized (MySQL store + requesthandler).")
	} else {
		logrus.Warn("DATABASE_DSN not configured; analysis endpoints are disabled.")
	}

	if errListen := runListen(sockHandler, evtHandler, peerEvtHandler, sipH, analysisH); errListen != nil {
		return nil, errors.Wrapf(errListen, "failed to run service listen")
	}

	// Run subscribe handler to consume events from all services and write to ClickHouse
	subscribeDone, errSubscribe := runSubscribe(ctx, sockHandler, db)
	if errSubscribe != nil {
		return nil, errors.Wrapf(errSubscribe, "failed to run subscribe handler")
	}

	return subscribeDone, nil
}

func runListen(sockListen sockhandler.SockHandler, evtHandler eventhandler.EventHandler, peerEvtHandler peereventhandler.PeerEventHandler, sipH siphandler.SIPHandler, analysisH analysishandler.AnalysisHandler) error {
	log := logrus.WithField("func", "runListen")

	listenHdlr := listenhandler.NewListenHandler(sockListen, evtHandler, peerEvtHandler, sipH, analysisH)

	if errRun := listenHdlr.Run(string(commonoutline.QueueNameTimelineRequest)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}

func runSubscribe(ctx context.Context, sockHandler sockhandler.SockHandler, db dbhandler.DBHandler) (<-chan struct{}, error) {
	log := logrus.WithField("func", "runSubscribe")

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, db)

	doneCh, errRun := subHandler.Run(ctx)
	if errRun != nil {
		log.Errorf("Error occurred in subscribe handler. err: %v", errRun)
		return nil, errRun
	}

	return doneCh, nil
}
