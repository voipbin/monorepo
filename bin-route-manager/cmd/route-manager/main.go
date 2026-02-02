package main

import (
	"database/sql"
	"net/http"
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
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-route-manager/internal/config"
	"monorepo/bin-route-manager/pkg/cachehandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/listenhandler"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

const (
	serviceName = commonoutline.ServiceNameRouteManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "route-manager",
	Short: "Route Manager Service",
	Long:  `Route Manager is a microservice that manages call routing in the VoIP system.`,
	RunE:  run,
}

func init() {
	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Initialize signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()

	// Bootstrap configuration
	if err := config.Bootstrap(rootCmd); err != nil {
		logrus.Fatalf("Failed to bootstrap config: %v", err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log := logrus.WithField("func", "run")

	// Load global configuration
	config.LoadGlobalConfig()

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

	if err := runService(sqlDB, cache); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
		return err
	}
	<-chDone
	return nil
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// runService runs the route-manager
func runService(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	cfg := config.Get()

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRouteEvent, serviceName, "")

	routeHandler := routehandler.NewRouteHandler(db, reqHandler, notifyHandler)
	providerHandler := providerhandler.NewProviderHandler(db, reqHandler, notifyHandler)

	// run request listener
	if err := runRequestListen(sockHandler, providerHandler, routeHandler); err != nil {
		return err
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(sockHandler sockhandler.SockHandler, providerHandler providerhandler.ProviderHandler, routeHandler routehandler.RouteHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, providerHandler, routeHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRouteRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
