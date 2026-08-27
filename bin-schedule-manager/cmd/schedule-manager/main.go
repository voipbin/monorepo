package main

import (
	"context"
	"database/sql"
	"fmt"
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

	"monorepo/bin-schedule-manager/internal/config"
	"monorepo/bin-schedule-manager/pkg/backuphandler"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
	"monorepo/bin-schedule-manager/pkg/dispatchhandler"
	"monorepo/bin-schedule-manager/pkg/listenhandler"
	"monorepo/bin-schedule-manager/pkg/schedulehandler"
	"monorepo/bin-schedule-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameScheduleManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	rootCmd := &cobra.Command{
		Use:   "schedule-manager",
		Short: "Voipbin Schedule Manager Daemon",
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

func runDaemon() error {
	initSignal()
	initProm(config.Get().PrometheusEndpoint, config.Get().PrometheusListenAddress)

	log := logrus.WithField("func", "runDaemon")
	log.WithField("config", config.Get()).Info("Starting schedule-manager...")

	sqlDB, err := commondatabasehandler.Connect(config.Get().DatabaseDSN)
	if err != nil {
		return errors.Wrapf(err, "could not connect to the database")
	}
	commondatabasehandler.RegisterDBStatsCollector(sqlDB, "main")
	defer commondatabasehandler.Close(sqlDB)

	cache, err := initCache()
	if err != nil {
		return errors.Wrapf(err, "could not initialize the cache")
	}

	// the dispatch loop stops when this context is cancelled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if errStart := runServices(ctx, sqlDB, cache); errStart != nil {
		return errors.Wrapf(errStart, "could not start services")
	}

	<-chDone
	cancel()
	log.Info("The schedule-manager stopped safely.")
	return nil
}

// runServices runs the services
func runServices(ctx context.Context, sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	cfg := config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameScheduleEvent, serviceName)

	scheduleHandler := schedulehandler.NewScheduleHandler(reqHandler, db, notifyHandler)
	backupHandler := backuphandler.NewBackupHandler(cfg.DatabaseDSN, cfg.ScheduleBackupDir, cfg.ScheduleBackupRetentionCount)
	dispatchHandler := dispatchhandler.NewDispatchHandler(db, cache, reqHandler, notifyHandler, cfg.ScheduleTickIntervalSec, cfg.ScheduleDispatchConcurrency)

	if err := runListen(sockHandler, scheduleHandler, dispatchHandler, backupHandler, cfg.ScheduleExecutionRetentionDays); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, scheduleHandler); err != nil {
		return err
	}

	// dispatch loop (design §5.2): every replica runs the same tick loop
	go dispatchHandler.Run(ctx)

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	scheduleHandler schedulehandler.ScheduleHandler,
	dispatchHandler dispatchhandler.DispatchHandler,
	backupHandler backuphandler.BackupHandler,
	executionRetentionDays int,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, scheduleHandler, dispatchHandler, backupHandler, executionRetentionDays)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameScheduleRequest), string(commonoutline.QueueNameDelay)); err != nil {
		return fmt.Errorf("could not run the listenhandler: %w", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, scheduleHandler schedulehandler.ScheduleHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameScheduleSubscribe),
		subscribeTargets,
		scheduleHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
