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
	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/pkg/emailhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-email-manager/pkg/cachehandler"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"monorepo/bin-email-manager/pkg/listenhandler"
)

const serviceName = commonoutline.ServiceNameFlowManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "email-manager",
	Short: "Email Manager Service",
	Long:  `Email Manager handles email sending and management for the VoIPbin platform.`,
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
	rootCmd.Flags().String("sendgrid_api_key", "", "API key for Sendgrid")
	rootCmd.Flags().String("mailgun_api_key", "", "API key for Mailgun")

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
	fmt.Printf("Hello world!\n")

	cfg := config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// create dbhandler
	dbHandler, err := createDBHandler(cfg)
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: ")
		return
	}

	// run the service
	run(dbHandler, cfg)
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameFlowEvent, serviceName)

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
