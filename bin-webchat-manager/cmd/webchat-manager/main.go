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

	"monorepo/bin-webchat-manager/internal/config"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/listenhandler"
	"monorepo/bin-webchat-manager/pkg/messagehandler"
	"monorepo/bin-webchat-manager/pkg/sessionhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

const serviceName = commonoutline.ServiceNameWebchatManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "webchat-manager",
		Short: "Voipbin Webchat Manager Daemon",
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
	log.WithField("config", config.Get()).Info("Starting webchat-manager...")

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to database")
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := cache.Connect(); err != nil {
		return errors.Wrapf(err, "could not connect to cache server")
	}

	db := dbhandler.NewHandler(sqlDB, cache)

	if errRun := run(db); errRun != nil {
		return errors.Wrapf(errRun, "run func has finished")
	}

	<-chDone
	log.Info("Webchat-manager stopped safely.")
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

// run runs the listen
func run(db dbhandler.DBHandler) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "run",
		},
	)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebchatEvent, serviceName)
	widgetHandler := widgethandler.NewWidgetHandler(reqHandler, db)
	sessionHandler := sessionhandler.NewSessionHandler(reqHandler, db)
	messageHandler := messagehandler.NewMessageHandler(reqHandler, notifyHandler, db)

	// run listen
	if err := runListen(sockHandler, widgetHandler, sessionHandler, messageHandler); err != nil {
		log.Errorf("Could not run listen. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	widgetHandler widgethandler.WidgetHandler,
	sessionHandler sessionhandler.SessionHandler,
	messageHandler messagehandler.MessageHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, widgetHandler, sessionHandler, messageHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameWebchatRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
