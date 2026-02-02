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
	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/pkg/emailhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-email-manager/pkg/cachehandler"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"monorepo/bin-email-manager/pkg/listenhandler"
)

const serviceName = commonoutline.ServiceNameEmailManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "email-manager",
		Short: "Email Manager Service",
		Long:  `Email Manager handles email sending and management for the VoIPbin platform.`,
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
	initSignal()

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// create dbhandler
	dbHandler, err := createDBHandler(cfg)
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: %v", err)
		return err
	}

	// run the service
	run(dbHandler, cfg)
	<-chDone
	logrus.Info("Email-manager stopped safely.")
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

// connectDatabase connects to the database and cachehandler
func createDBHandler(cfg *config.Config) (dbhandler.DBHandler, error) {
	// connect to database
	db, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	return dbHandler, nil
}

func run(dbHandler dbhandler.DBHandler, cfg *config.Config) {
	log := logrus.WithField("func", "run")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameEmailEvent, serviceName, "")

	emailHandler := emailhandler.NewEmailHandler(dbHandler, reqHandler, notifyHandler, cfg.SendgridAPIKey, cfg.MailgunAPIKey)

	// run listen
	if errListen := runListen(sockHandler, emailHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}

}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,

	emailHandler emailhandler.EmailHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, emailHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameEmailRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
