package main

import (
	"database/sql"
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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-billing-manager/internal/config"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/cachehandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
	"monorepo/bin-billing-manager/pkg/listenhandler"
	"monorepo/bin-billing-manager/pkg/paddlehandler"
	"monorepo/bin-billing-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameBillingManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "billing-manager",
		Short: "billing-manager is a billing management service",
		Long:  `billing-manager manages billing accounts, balance tracking, and subscription management.`,
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

func runDaemon() error {
	log := logrus.WithField("func", "runDaemon")

	log.Info("Starting billing-manager.")

	// init signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()

	// init prometheus
	config.InitPrometheus()

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return err
	}
	commondatabasehandler.RegisterDBStatsCollector(sqlDB, "main")
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(config.Get().RedisAddress, config.Get().RedisPassword, config.Get().RedisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return err
	}

	// run
	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("Could not run the process correctly. err: %v", errRun)
		return errRun
	}
	<-chDone
	log.Info("billing-manager stopped safely.")
	return nil
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	close(chDone)
}

// run runs the billing-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameBillingEvent, serviceName)

	paddleHandler := paddlehandler.NewPaddleHandler(
		config.Get().PaddleAPIKey,
		config.Get().PaddlePriceIDBasic,
		config.Get().PaddlePriceIDProfessional,
	)

	accountHandler := accounthandler.NewAccountHandler(reqHandler, db, notifyHandler, paddleHandler)
	billingHandler := billinghandler.NewBillingHandler(reqHandler, db, notifyHandler, accountHandler)

	// build the subscribe handler and its failed event handler together — this must
	// happen before runListen so the failed event handler can be wired into the
	// listenhandler for the /v1/failed_events/retry route.
	subHandler, failedHandler := buildFailedEventHandler(db, sockHandler, accountHandler, billingHandler)

	// run listen
	if err := runListen(sockHandler, accountHandler, billingHandler, paddleHandler, failedHandler); err != nil {
		return err
	}

	// run subscribe
	if err := runSubscribe(subHandler); err != nil {
		return err
	}

	return nil
}

// buildFailedEventHandler constructs the subscribe handler and its failed event
// handler together. The two have a circular dependency: the failed event handler
// needs the subscribe handler's event processor to retry events, and the subscribe
// handler needs the failed event handler to persist events it fails to process. The
// subscribe handler is created first with a nil failed-event processor placeholder,
// then the failed event handler is built from its event processor, then patched back
// onto the subscribe handler.
func buildFailedEventHandler(db dbhandler.DBHandler, sockHandler sockhandler.SockHandler, accoutHandler accounthandler.AccountHandler, billingHandler billinghandler.BillingHandler) (subscribehandler.SubscribeHandler, failedeventhandler.FailedEventHandler) {
	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameMessageEvent),
		string(commonoutline.QueueNameEmailEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameNumberEvent),
		string(commonoutline.QueueNameTTSEvent),
	}

	// placeholder processor — will be set after subscribe handler is created
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameBillingSubscribe),
		subscribeTargets,
		accoutHandler,
		billingHandler,
		nil, // temporary nil — set below
	)

	// create failed event handler with the subscribe handler's process function
	failedHandler := failedeventhandler.NewFailedEventHandler(db, subscribehandler.GetEventProcessor(subHandler))

	// set the failed event handler on the subscribe handler
	subscribehandler.SetFailedEventHandler(subHandler, failedHandler)

	return subHandler, failedHandler
}

// runSubscribe runs the subscribed event handler
func runSubscribe(subHandler subscribehandler.SubscribeHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(sockHandler sockhandler.SockHandler, accoutHandler accounthandler.AccountHandler, billingHandler billinghandler.BillingHandler, paddleHandler paddlehandler.PaddleHandler, failedHandler failedeventhandler.FailedEventHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, accoutHandler, billingHandler, paddleHandler, failedHandler)

	if err := listenHandler.Run(string(commonoutline.QueueNameBillingRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
