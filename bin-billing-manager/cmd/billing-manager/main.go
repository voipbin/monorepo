package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
	"monorepo/bin-billing-manager/pkg/cachehandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	"monorepo/bin-billing-manager/pkg/failedeventhandler"
	"monorepo/bin-billing-manager/pkg/listenhandler"
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
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameBillingEvent, serviceName, "")

	accountHandler := accounthandler.NewAccountHandler(reqHandler, db, notifyHandler)
	billingHandler := billinghandler.NewBillingHandler(reqHandler, db, notifyHandler, accountHandler)

	// run listen
	if err := runListen(sockHandler, accountHandler, billingHandler); err != nil {
		return err
	}

	// run subscribe (with failed event handler)
	if err := runSubscribe(db, sockHandler, accountHandler, billingHandler); err != nil {
		return err
	}

	// run monthly token top-up cron
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := runMonthlyTopUp(context.Background(), db); err != nil {
					log.Errorf("Monthly top-up failed. err: %v", err)
				}
			case <-chDone:
				return
			}
		}
	}()

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(db dbhandler.DBHandler, sockHandler sockhandler.SockHandler, accoutHandler accounthandler.AccountHandler, billingHandler billinghandler.BillingHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameMessageEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameNumberEvent),
	}

	// create subscribe handler first, then wire the failed event handler with a circular reference
	// to the subscribe handler's processEvent
	var subHandler subscribehandler.SubscribeHandler
	var failedHandler failedeventhandler.FailedEventHandler

	// placeholder processor — will be set after subscribe handler is created
	subHandler = subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameBillingSubscribe),
		subscribeTargets,
		accoutHandler,
		billingHandler,
		nil, // temporary nil — set below
	)

	// create failed event handler with the subscribe handler's process function
	failedHandler = failedeventhandler.NewFailedEventHandler(db, subscribehandler.GetEventProcessor(subHandler))

	// set the failed event handler on the subscribe handler
	subscribehandler.SetFailedEventHandler(subHandler, failedHandler)

	// run
	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", err)
		return err
	}

	// start failed event retry loop
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := failedHandler.RetryPending(context.Background()); err != nil {
					log.Errorf("Failed event retry error. err: %v", err)
				}
			case <-chDone:
				return
			}
		}
	}()

	return nil
}

// runListen runs the listen handler
func runListen(sockHandler sockhandler.SockHandler, accoutHandler accounthandler.AccountHandler, billingHandler billinghandler.BillingHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, accoutHandler, billingHandler)

	if err := listenHandler.Run(string(commonoutline.QueueNameBillingRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}

func runMonthlyTopUp(ctx context.Context, db dbhandler.DBHandler) error {
	log := logrus.WithField("func", "runMonthlyTopUp")

	now := time.Now()
	filters := map[account.Field]any{
		account.FieldDeleted: false,
	}

	// Paginate through all accounts
	var pageToken string
	for {
		accounts, err := db.AccountList(ctx, 500, pageToken, filters)
		if err != nil {
			return fmt.Errorf("could not list accounts. err: %v", err)
		}
		if len(accounts) == 0 {
			break
		}

		for _, a := range accounts {
			if a.TmNextTopUp == nil || a.TmNextTopUp.After(now) {
				continue
			}

			tokenAmount, ok := account.PlanTokenMap[a.PlanType]
			if !ok || tokenAmount <= 0 {
				continue
			}

			if err := db.AccountTopUpTokens(ctx, a.ID, a.CustomerID, int64(tokenAmount), string(a.PlanType)); err != nil {
				log.Errorf("Could not top up tokens for account. account_id: %s, err: %v", a.ID, err)
				continue
			}
			log.Infof("Topped up tokens for account. account_id: %s, tokens: %d", a.ID, tokenAmount)
		}

		if len(accounts) < 500 {
			break
		}
		pageToken = accounts[len(accounts)-1].TMCreate.Format(time.RFC3339Nano)
	}

	return nil
}
