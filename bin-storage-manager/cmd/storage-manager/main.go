package main

import (
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

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
	"monorepo/bin-storage-manager/pkg/listenhandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"monorepo/bin-storage-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameStorageManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "storage-manager",
	Short: "Storage Manager Service",
	Long:  `Storage Manager is a microservice that manages file storage and GCP bucket integration.`,
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
	rootCmd.Flags().String("gcp_project_id", "", "GCP project id.")
	rootCmd.Flags().String("gcp_bucket_name_media", "", "GCP bucket name for media storage.")
	rootCmd.Flags().String("gcp_bucket_name_tmp", "", "GCP bucket name for tmp storage.")

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
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// Initialize configuration
	if err := config.Bootstrap(cmd); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	config.LoadGlobalConfig()
	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// create dbhandler
	dbHandler, err := createDBHandler()
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: ")
		return err
	}

	if errRun := runService(dbHandler); errRun != nil {
		log.Errorf("Could not run correctly. err: %v", errRun)
		return errRun
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

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	cfg := config.Get()

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

// Run the services
func runService(dbHandler dbhandler.DBHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runService",
	})

	cfg := config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameStorageEvent, serviceName, "")
	accountHandler := accounthandler.NewAccountHandler(notifyHandler, dbHandler)
	fileHandler := filehandler.NewFileHandler(notifyHandler, dbHandler, accountHandler, cfg.GCPProjectID, cfg.GCPBucketNameMedia, cfg.GCPBucketNameTmp)
	storageHandler := storagehandler.NewStorageHandler(reqHandler, fileHandler, cfg.GCPBucketNameMedia)

	// run listener
	if errListen := runListen(sockHandler, storageHandler, accountHandler); errListen != nil {
		log.Errorf("Could not run the listener correctly. err: %v", errListen)
		return errListen
	}

	// run sbuscriber
	if errSubs := runSubscribe(sockHandler, string(commonoutline.QueueNameStorageSubscribe), accountHandler, fileHandler); errSubs != nil {
		log.Errorf("Could not run the subscriber correctly. err: %v", errSubs)
		return errSubs
	}

	return nil
}

// runListen run the listener
func runListen(sockHandler sockhandler.SockHandler, storageHandler storagehandler.StorageHandler, accountHandler accounthandler.AccountHandler) error {

	// create listen handler
	listenHandler := listenhandler.NewListenHandler(sockHandler, storageHandler, accountHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameStorageRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		return errRun
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, subscribeQueue string, accountHandler accounthandler.AccountHandler, fileHandler filehandler.FileHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, subscribeQueue, subscribeTargets, accountHandler, fileHandler)

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
