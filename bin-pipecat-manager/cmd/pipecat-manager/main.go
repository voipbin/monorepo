package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-pipecat-manager/pkg/cachehandler"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/httphandler"
	"monorepo/bin-pipecat-manager/pkg/listenhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	_ "github.com/go-sql-driver/mysql"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-pipecat-manager/internal/config"
)

const (
	serviceName = commonoutline.ServiceNamePipecatManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "pipecat-manager",
		Short: "Voipbin Pipecat Manager Daemon",
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

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting pipecat-manager...")

	if errRun := run(); errRun != nil {
		return errors.Wrapf(errRun, "could not run the main thread")
	}

	<-chDone
	log.Info("Pipecat-manager stopped safely.")
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

// run runs the main thread.
func run() error {
	log := logrus.WithField("func", "run")

	dbHandler, err := createDBHandler()
	if err != nil {
		return errors.Wrapf(err, "could not create dbhandler")
	}
	log.Debugf("Connected to database and cache server.")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	listenIP := os.Getenv("POD_IP")
	if listenIP == "" {
		return fmt.Errorf("could not get the listen ip address")
	}
	listenAddress := fmt.Sprintf("%s:%d", listenIP, 8080)

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNamePipecatEvent, serviceName)

	pipecatcallHandler := pipecatcallhandler.NewPipecatcallHandler(requestHandler, notifyHandler, dbHandler, listenAddress, listenIP)
	httpHandler := httphandler.NewHttpHandler(requestHandler, pipecatcallHandler)

	// run listen
	if errListen := runListen(sockHandler, listenIP, pipecatcallHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errors.Wrapf(errListen, "could not start runListen")
	}

	// run streaming
	if errStreaming := runStreaming(pipecatcallHandler); errStreaming != nil {
		log.Errorf("Could not start runStreaming. err: %v", errStreaming)
		return errors.Wrapf(errStreaming, "could not start runStreaming")
	}

	// run http
	if errHttp := runHttp(httpHandler); errHttp != nil {
		log.Errorf("Could not start runHttp. err: %v", errHttp)
		return errors.Wrapf(errHttp, "could not start runHttp")
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	hostID string,
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, pipecatcallHandler)

	// run
	listenQueue := fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, hostID)
	if err := listenHandler.Run(string("bin-manager.pipecat-manager.request"), listenQueue, string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run the listenhandler correctly")
	}

	return nil
}

func runStreaming(pipecatcallHandler pipecatcallhandler.PipecatcallHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})

	go func() {
		if errRun := pipecatcallHandler.Run(); errRun != nil {
			log.Errorf("Could not run the streaming handler correctly. err: %v", errRun)
		}
	}()

	return nil
}

func runHttp(httpHandler httphandler.HttpHandler) error {
	go func() {
		if errRun := httpHandler.Run(); errRun != nil {
			logrus.Errorf("Could not run the http handler correctly. err: %v", errRun)
		}
	}()

	return nil
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
