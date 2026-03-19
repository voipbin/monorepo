package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/storage"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	joonix "github.com/joonix/log"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-rag-manager/internal/config"
	"monorepo/bin-rag-manager/pkg/bucketreader"
	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/embedder"
	"monorepo/bin-rag-manager/pkg/listenhandler"
	"monorepo/bin-rag-manager/pkg/raghandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "rag-manager",
	Short: "RAG Manager Service",
	Long:  `RAG Manager is a microservice that provides multi-tenant RAG knowledge base for VoIPBin.`,
	RunE:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server")
	rootCmd.Flags().String("gcp_project_id", "", "GCP project ID for Vertex AI")
	rootCmd.Flags().String("gcp_location", "", "GCP region for Vertex AI")
	rootCmd.Flags().String("google_embedding_model", "text-embedding-004", "Google embedding model")
	rootCmd.Flags().Int("rag_top_k", 5, "Default number of chunks to retrieve")
	rootCmd.Flags().String("gcp_bucket_name_media", "", "GCS bucket name for media files")
	rootCmd.Flags().String("postgresql_dsn", "", "PostgreSQL connection string")

	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Initialize signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log := logrus.WithField("func", "run")

	// Initialize configuration
	if err := config.InitConfig(cmd); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	cfg := *config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// Run database migrations before starting services
	if err := runMigrations(cfg); err != nil {
		return fmt.Errorf("could not run migrations: %w", err)
	}

	if err := runService(cfg); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
		return err
	}

	return nil
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// runMigrations applies pending database migrations at startup
func runMigrations(cfg config.Config) error {
	log := logrus.WithField("func", "runMigrations")

	if cfg.PostgreSQLDSN == "" {
		log.Warn("PostgreSQL DSN not configured, skipping migrations")
		return nil
	}

	const migrationsPath = "./migrations"
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	log.WithFields(logrus.Fields{
		"source": sourceURL,
	}).Info("Running database migrations...")

	m, err := migrate.New(sourceURL, cfg.PostgreSQLDSN)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, _ := m.Version()
	log.WithFields(logrus.Fields{
		"version": version,
		"dirty":   dirty,
	}).Info("Database migrations completed")

	return nil
}

// runService initializes and starts the RAG service
func runService(cfg config.Config) error {
	log := logrus.WithField("func", "runService")

	ctx := context.Background()

	// RabbitMQ connection
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// PostgreSQL connection
	db, err := sql.Open("postgres", cfg.PostgreSQLDSN)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("could not ping PostgreSQL: %w", err)
	}
	log.Info("Connected to PostgreSQL")

	// Request handler
	reqHandler := requesthandler.NewRequestHandler(sockHandler, commonoutline.ServiceNameRagManager)

	// GCS client
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create GCS client: %w", err)
	}

	// Bucket reader
	br := bucketreader.NewBucketReader(gcsClient)

	dbH := dbhandler.NewHandler(db)

	// Initialize Google Gemini embedder
	emb, err := embedder.NewGoogleEmbedder(ctx, cfg.GoogleCloudProject, cfg.GoogleCloudLocation, cfg.GoogleEmbeddingModel)
	if err != nil {
		return fmt.Errorf("could not create embedder: %w", err)
	}

	// Initialize rag handler
	ragH := raghandler.NewRagHandler(emb, dbH, reqHandler, br, cfg.GCPBucketNameMedia)

	// Startup sweep: re-process pending documents
	ragH.DocumentIngestPendingAll(ctx)

	// Start periodic ticker
	go ragH.RunIngestionTicker(ctx, 5*time.Minute)

	// Run listen handler
	if err := runListen(sockHandler, ragH); err != nil {
		return err
	}

	// Block until shutdown signal — keeps DB connection alive for request handlers
	<-chDone

	return nil
}

// runListen starts the RPC listen handler
func runListen(sockHandler sockhandler.SockHandler, ragH raghandler.RagHandler) error {
	lh := listenhandler.NewListenHandler(sockHandler, ragH)

	if err := lh.Run(string(commonoutline.QueueNameRagRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// initProm initializes prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
