package main

import (
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	commonoutline "monorepo/bin-common-handler/models/outline"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-registrar-manager/internal/config"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/listenhandler"
	"monorepo/bin-registrar-manager/pkg/subscribehandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

const serviceName = commonoutline.ServiceNameRegistrarManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "registrar-manager",
		Short: "Voipbin Registrar Manager Daemon",
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
	log.WithField("config", config.Get()).Info("Starting registrar-manager...")

	sqlDBBin, err := commondatabasehandler.Connect(config.Get().DatabaseDSNBin)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the voipbin database")
	}
	defer commondatabasehandler.Close(sqlDBBin)

	sqlDBAsterisk, err := commondatabasehandler.Connect(config.Get().DatabaseDSNAsterisk)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the asterisk database")
	}
	defer commondatabasehandler.Close(sqlDBAsterisk)

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	if errStart := run(sqlDBAsterisk, sqlDBBin, cache); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	log.Info("Registrar-manager stopped safely.")
	return nil
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

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		logrus.Infof("Prometheus metrics server starting on %s%s", listen, endpoint)
		if err := http.ListenAndServe(listen, nil); err != nil {
			// Prometheus server error is logged but not treated as fatal to avoid unsafe exit from a goroutine.
			logrus.Errorf("Prometheus server error: %v", err)
		}
	}()
}

// NewWorker creates worker interface
func run(sqlAst *sql.DB, sqlBin *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	dbAst := dbhandler.NewHandler(sqlAst, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler)
	trunkHandler := trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler)
	contactHandler := contacthandler.NewContactHandler(reqHandler, dbAst, dbBin)

	// run listen
	if errListen := runListen(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler); errListen != nil {
		log.Errorf("Could not run the listener. err: %v", errListen)
		return errListen
	}

	// run subscriber
	if errSubscribe := runSubscribe(sockHandler, extensionHandler, trunkHandler); errSubscribe != nil {
		log.Errorf("Could not run the subscriber. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, trunkHandler trunkhandler.TrunkHandler, extensionHandler extensionhandler.ExtensionHandler, contactHandler contacthandler.ContactHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRegistrarRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, extensionHandler extensionhandler.ExtensionHandler, trunkHandler trunkhandler.TrunkHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debugf("Subscribe target details. len: %d", len(subscribeTargets))

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameRegistrarSubscribe), subscribeTargets, extensionHandler, trunkHandler)

	// run
	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}
