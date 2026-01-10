package main

import (
	"database/sql"
	"fmt"
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

	"monorepo/bin-tag-manager/internal/config"
	"monorepo/bin-tag-manager/pkg/cachehandler"
	"monorepo/bin-tag-manager/pkg/dbhandler"
	"monorepo/bin-tag-manager/pkg/listenhandler"
	"monorepo/bin-tag-manager/pkg/subscribehandler"
	"monorepo/bin-tag-manager/pkg/taghandler"
)

const serviceName = commonoutline.ServiceNameTagManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "tag-manager",
	Short: "Tag Manager Service",
	Long:  `Tag Manager is a microservice that manages resource labeling and tagging.`,
	RunE:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	rootCmd.Flags().String("redis_address", "127.0.0.1:6379", "Address of the Redis server (e.g., localhost:6379)")
	rootCmd.Flags().Int("redis_database", 1, "Redis database index to use (default is 1)")
	rootCmd.Flags().String("redis_password", "", "Password for authenticating with the Redis server (if required)")

	// Initialize logging
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)

	// Initialize signal handler
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	log := logrus.WithField("func", "run")

	// Initialize configuration
	if err := config.InitConfig(cmd); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

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

// runService runs the listen
func runService(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	cfg := config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTagEvent, serviceName)
	tagHandler := taghandler.NewTagHandler(reqHandler, db, notifyHandler)

	if err := runListen(sockHandler, tagHandler); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, tagHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, tagHandler taghandler.TagHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, tagHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameTagRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, tagHandler taghandler.TagHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameTagSubscribe), subscribeTargets, tagHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
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
