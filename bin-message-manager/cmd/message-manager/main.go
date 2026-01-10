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
	"monorepo/bin-message-manager/pkg/requestexternal"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
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

var rootCmd = &cobra.Command{
	Use:   "message-manager",
	Short: "Message Manager Service",
	Long:  `Message Manager handles SMS and messaging for the VoIPbin platform.`,
	Run:   runService,
}

func init() {
	// Define flags
	rootCmd.Flags().String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	rootCmd.Flags().String("redis_address", "127.0.0.1:6379", "Address of the Redis server (e.g., localhost:6379)")
	rootCmd.Flags().String("redis_password", "", "Password for authenticating with the Redis server (if required)")
	rootCmd.Flags().Int("redis_database", 1, "Redis database index to use (default is 1)")
	rootCmd.Flags().String("authtoken_messagebird", "", "The authtoken for the messagebird.")
	rootCmd.Flags().String("authtoken_telnyx", "", "The authtoken for the telnyx.")

	// Initialize configuration
	config.InitConfig(rootCmd)

	// Initialize logging
	initLog()

	// Initialize signal handler
	initSignal()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Errorf("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

func runService(cmd *cobra.Command, args []string) {
	log := logrus.WithField("func", "runService")
	log.Debugf("Starting message-manager.")

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlDB, cache, cfg); errRun != nil {
		log.Errorf("The run returned error. err: %v", errRun)
	}
	<-chDone
}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal inits signal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameMessageEvent, serviceName)

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
