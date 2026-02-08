package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
	"monorepo/bin-customer-manager/pkg/listenhandler"
)

const serviceName = commonoutline.ServiceNameCustomerManager

var (
	chSigs = make(chan os.Signal, 1)
	chDone = make(chan bool, 1)
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "customer-manager",
		Short: "Voipbin Customer Manager Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDaemon()
		},
	}

	if err := config.Bootstrap(rootCmd); err != nil {
		logrus.Fatalf("Failed to bootstrap config: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddr)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting customer-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return err
	}

	if err := startServices(sqlDB, cache); err != nil {
		return err
	}

	<-chDone
	log.Info("Customer-manager stopped safely.")
	return nil
}

func startServices(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	db := dbhandler.NewHandler(sqlDB, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCustomerEvent, serviceName, "")
	customerHandler := customerhandler.NewCustomerHandler(reqHandler, db, cache, notifyHandler)
	accesskeyHandler := accesskeyhandler.NewAccesskeyHandler(reqHandler, db, notifyHandler)

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, customerHandler, accesskeyHandler)

	// start background cleanup job for expired unverified customers
	go customerHandler.RunCleanupUnverified(context.Background())

	return listenHandler.Run(string(commonoutline.QueueNameCustomerRequest), string(commonoutline.QueueNameDelay))
}

func initCache() (cachehandler.CacheHandler, error) {
	res := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := res.Connect(); err != nil {
		return nil, errors.Wrap(err, "cache connect error")
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

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}
