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

	"monorepo/bin-webhook-manager/internal/config"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
	"monorepo/bin-webhook-manager/pkg/listenhandler"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

const serviceName = commonoutline.ServiceNameWebhookManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "webhook-manager",
		Short: "Voipbin Webhook Manager Daemon",
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
	log.WithField("config", config.Get()).Info("Starting webhook-manager...")

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
		return errors.Wrapf(errRun, "could not run webhook-manager")
	}

	<-chDone
	log.Info("Webhook-manager stopped safely.")
	return nil
}

// run runs the webhook-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// run listen
	if err := runListen(db, cache); err != nil {
		return errors.Wrapf(err, "could not run listen handler")
	}

	log.Debug("All handlers started successfully")
	return nil
}

// runListen runs the listen handler
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})
	log.Debugf("Running listen handler")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebhookEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler)

	whHandler := webhookhandler.NewWebhookHandler(db, notifyHandler, accountHandler)

	listenHandler := listenhandler.NewListenHandler(sockHandler, whHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameWebhookRequest), string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run the listen handler correctly")
	}

	return nil
}
