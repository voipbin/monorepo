package main

import (
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-transfer-manager/internal/config"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
	"monorepo/bin-transfer-manager/pkg/listenhandler"
	"monorepo/bin-transfer-manager/pkg/subscribehandler"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

const serviceName = "transfer-manager"

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "transfer-manager",
		Short: "Voipbin Transfer Manager Daemon",
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

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-chSigs
		logrus.Infof("Received signal: %v", sig)
		chDone <- true
	}()
}

func initProm(endpoint, listen string) {
	// Skip Prometheus initialization if endpoint or listen address is not configured
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics disabled (endpoint or listen address not configured)")
		return
	}

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			// Prometheus server error is logged but not treated as fatal to avoid unsafe exit from a goroutine.
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if errConnect := res.Connect(); errConnect != nil {
		return nil, errors.Wrap(errConnect, "cache connect error")
	}
	return res, nil
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting transfer-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	if errRun := run(sqlDB, cache); errRun != nil {
		return errors.Wrapf(errRun, "could not run transfer-manager")
	}

	<-chDone
	log.Info("Transfer-manager stopped safely.")
	return nil
}

// run runs the transfer-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTransferEvent, serviceName, "")
	transferHandler := transferhandler.NewTransferHandler(reqHandler, notifyHandler, db)

	// run event listener
	if err := runSubscribe(serviceName, sockHandler, transferHandler); err != nil {
		return errors.Wrapf(err, "could not run the subscribe handler")
	}

	// run request listener
	if err := runRequestListen(sockHandler, transferHandler); err != nil {
		return errors.Wrapf(err, "could not run the listen handler")
	}

	log.Debug("All handlers started successfully")
	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	serviceName string,
	sockHandler sockhandler.SockHandler,
	transferHandler transferhandler.TransferHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	subscribeHandler := subscribehandler.NewSubscribeHandler(serviceName, sockHandler, string(commonoutline.QueueNameTransferSubscribe), subscribeTargets, transferHandler)

	// run
	if err := subscribeHandler.Run(); err != nil {
		return errors.Wrapf(err, "could not run the subscribe handler correctly")
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	sockHandler sockhandler.SockHandler,
	transferHandler transferhandler.TransferHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runRequestListen",
	})
	log.Debugf("Running listen handler")

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameTransferRequest),
		string(commonoutline.QueueNameDelay),
		transferHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		return errors.Wrapf(err, "could not run the listen handler correctly")
	}

	return nil
}
