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
	"monorepo/bin-message-manager/pkg/requestexternal"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/pkg/cachehandler"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/listenhandler"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

const serviceName = commonoutline.ServiceNameMessageManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "message-manager",
		Short: "Message Manager Service",
		Long:  `Message Manager handles SMS and messaging for the VoIPbin platform.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config.LoadGlobalConfig()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runService()
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

func runService() error {
	log := logrus.WithField("func", "runService")
	log.Debugf("Starting message-manager.")

	initSignal()

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return err
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return err
	}

	if errRun := run(sqlDB, cache, cfg); errRun != nil {
		log.Errorf("The run returned error. err: %v", errRun)
		return errRun
	}
	<-chDone
	log.Info("Message-manager stopped safely.")
	return nil
}

// initSignal inits signal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	// Skip Prometheus initialization if endpoint or listen address is not configured
	if endpoint == "" || listen == "" {
		logrus.Debug("Prometheus metrics server disabled (endpoint or listen address not configured)")
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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the message-manager
func run(db *sql.DB, cache cachehandler.CacheHandler, cfg *config.Config) error {
	if err := runListen(db, cache, cfg); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler, cfg *config.Config) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameMessageEvent, serviceName, "")

	requestExternal := requestexternal.NewRequestExternal(cfg.AuthtokenMessagebird, cfg.AuthtokenTelnyx)

	messageHandler := messagehandler.NewMessageHandler(reqHandler, notifyHandler, db, requestExternal)
	listenHandler := listenhandler.NewListenHandler(sockHandler, messageHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameMessageRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
