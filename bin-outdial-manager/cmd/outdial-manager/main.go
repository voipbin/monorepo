package main

import (
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

	"monorepo/bin-outdial-manager/internal/config"
	"monorepo/bin-outdial-manager/pkg/cachehandler"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
	"monorepo/bin-outdial-manager/pkg/listenhandler"
	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

const serviceName = commonoutline.ServiceNameOutdialManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "outdial-manager",
		Short: "Voipbin Outdial Manager Daemon",
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

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting outdial-manager...")

	// create dbhandler
	dbHandler, err := createDBHandler()
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database or failed to initiate the cachehandler")
	}

	// run the service
	if errRun := run(dbHandler); errRun != nil {
		return errors.Wrapf(errRun, "could not run the service")
	}

	<-chDone
	log.Info("Outdial-manager stopped safely.")
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
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	// connect to database
	db, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	return dbHandler, nil
}

func run(dbHandler dbhandler.DBHandler) error {
	log := logrus.WithField("func", "run")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameOutdialEvent, serviceName)

	outdialHandler := outdialhandler.NewOutdialHandler(dbHandler, reqHandler, notifyHandler)
	outdialTargethandler := outdialtargethandler.NewOutdialTargetHandler(dbHandler, reqHandler, notifyHandler)

	// run listen
	if errListen := runListen(sockHandler, outdialHandler, outdialTargethandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return errors.Wrapf(errListen, "could not run the listen")
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,
	outdialHandler outdialhandler.OutdialHandler,
	outdialtargetHandler outdialtargethandler.OutdialTargetHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, outdialHandler, outdialtargetHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameOutdialRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
