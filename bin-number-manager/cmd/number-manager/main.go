package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-number-manager/internal/config"
	"monorepo/bin-number-manager/migrations"
	"monorepo/bin-number-manager/pkg/cachehandler"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/listenhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
	"monorepo/bin-number-manager/pkg/secretcrypto"
	"monorepo/bin-number-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameNumberManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "number-manager",
		Short: "Voipbin Number Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if errBind := config.Bootstrap(rootCmd); errBind != nil {
		logrus.Fatalf("Failed to bind config: %v", errBind)
	}

	if errExecute := rootCmd.Execute(); errExecute != nil {
		logrus.Errorf("Command execution failed: %v", errExecute)
		os.Exit(1)
	}
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrap(errConnect, "cache connect error")
	}
	return res, nil
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

// promReadHeaderTimeout bounds how long the metrics server waits for request headers.
const promReadHeaderTimeout = 10 * time.Second

// newPromServer returns the metrics http server. Handler is nil, so it serves http.DefaultServeMux.
func newPromServer(listen string) *http.Server {
	return &http.Server{
		Addr:              listen,
		ReadHeaderTimeout: promReadHeaderTimeout,
	}
}

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := newPromServer(listen).ListenAndServe(); err != nil {
			// Prometheus server error is logged but not treated as fatal to avoid unsafe exit from a goroutine.
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	logStartup(log, config.Get())

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	// apply the schema migrations before anything else uses the database
	if errMigrate := migrationhandler.Run(config.Get().DatabaseDSN, migrations.FS, migrations.VersionTable); errMigrate != nil {
		return errors.Wrapf(errMigrate, "could not apply the database migrations")
	}

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	if errStart := runServices(sqlDB, cache); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	log.Info("The number-manager stopped safely.")
	return nil
}

// runServices runs the services
func runServices(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameNumberEvent, serviceName, notifyhandler.WithGlobalTopicPublish())

	// A malformed key set must not stop the daemon: other services depend on the numbers. The
	// provider secrets of the DB stay unusable (fail closed) and the environment credentials apply.
	crypto, errCrypto := secretcrypto.New(config.Get().NumberSecretEncryptionKeys)
	if errCrypto != nil {
		logrus.Errorf("The number secret encryption keys are malformed. The provider secrets of the DB cannot be used. err: %v", errCrypto)
		crypto, _ = secretcrypto.New("")
	}

	env := providerhandler.NewEnvCredentials(
		config.Get().TelnyxConnectionID,
		config.Get().TelnyxProfileID,
		config.Get().TelnyxToken,
		config.Get().TwilioSID,
		config.Get().TwilioToken,
	)
	providerHandler := providerhandler.NewProviderHandler(reqHandler, db, crypto, env)

	numberHandler := numberhandler.NewNumberHandler(reqHandler, db, notifyHandler, providerHandler)

	if err := runServiceListen(sockHandler, numberHandler, providerHandler); err != nil {
		return err
	}

	if err := runServiceSubscribe(sockHandler, numberHandler); err != nil {
		return err
	}

	return nil
}

// runServiceListen runs the listen service
func runServiceListen(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler, providerHandler providerhandler.ProviderHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, numberHandler, providerHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		return fmt.Errorf("could not run the listenhandler: %w", err)
	}

	return nil
}

// runServiceSubscribe runs the subscribed event handler
func runServiceSubscribe(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler) error {

	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameNumberSubscribe),
		numberHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}

// logStartup logs the settings at startup. The credentials of the config are left out on purpose.
func logStartup(log *logrus.Entry, cfg *config.Config) {
	log.WithFields(logrus.Fields{
		"prometheus_endpoint":       cfg.PrometheusEndpoint,
		"prometheus_listen_address": cfg.PrometheusListenAddress,
		"redis_address":             cfg.RedisAddress,
		"redis_database":            cfg.RedisDatabase,
		"telnyx_connection_id":      cfg.TelnyxConnectionID,
		"telnyx_profile_id":         cfg.TelnyxProfileID,
	}).Info("Starting number-manager...")
}
