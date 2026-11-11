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
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"monorepo/bin-push-manager/internal/config"
	"monorepo/bin-push-manager/migrations"
	"monorepo/bin-push-manager/pkg/dbhandler"
	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/eventhandler"
	"monorepo/bin-push-manager/pkg/listenhandler"
	"monorepo/bin-push-manager/pkg/notificationhandler"
	"monorepo/bin-push-manager/pkg/pushhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
	"monorepo/bin-push-manager/pkg/subscribehandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var rootCmd = &cobra.Command{
	Use:   "push-manager",
	Short: "Push Manager Service",
	Long:  `Push Manager is a microservice that stores the push devices of the agents and delivers push notifications to them.`,
	RunE:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	rootCmd.Flags().String("prometheus_endpoint", "/metrics", "URL for the Prometheus metrics endpoint")
	rootCmd.Flags().String("prometheus_listen_address", ":2112", "Address for Prometheus to listen on (e.g., localhost:8080)")
	rootCmd.Flags().String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "Address of the RabbitMQ server")
	rootCmd.Flags().String("fcm_service_account_file", "", "Path of the FCM service account JSON file. Empty means FCM is not configured")

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

	cfg := *config.Get()

	// Initialize Prometheus
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return err
	}
	defer commondatabasehandler.Close(sqlDB)

	// apply the schema migrations before anything else uses the database
	if err := migrationhandler.Run(cfg.DatabaseDSN, migrations.FS, migrations.VersionTable); err != nil {
		log.Errorf("Could not apply the database migrations. err: %v", err)
		return err
	}

	if err := runService(sqlDB); err != nil {
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

// runService runs the listen and the subscribe
func runService(sqlDB *sql.DB) error {
	cfg := *config.Get()

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// create handlers. The sender never fails to build: without a usable credential it reports
	// "not configured" for every send.
	db := dbhandler.NewHandler(sqlDB)
	sender := pushhandler.NewSender(cfg.FCMServiceAccountFile)
	deviceHandler := devicehandler.NewDeviceHandler(db)
	notificationHandler := notificationhandler.NewNotificationHandler(db, sender)
	settingsHandler := settingshandler.NewSettingsHandler(db)

	// the request handler registers its metrics, so it is built once per process
	reqHandler := requesthandler.NewRequestHandler(sockHandler, commonoutline.ServiceNamePushManager)
	eventHandler := eventhandler.NewEventHandler(reqHandler, settingsHandler, notificationHandler)

	listenHandler := listenhandler.NewListenHandler(sockHandler, deviceHandler, notificationHandler, settingsHandler)
	if err := listenHandler.Run(string(commonoutline.QueueNamePushRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNamePushSubscribe), deviceHandler, settingsHandler, eventHandler)
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
			err := http.ListenAndServe(listen, nil) //nolint:gosec // G114: Prometheus metrics endpoint serving only the metrics handler; adding server timeouts would change behavior and is out of scope for lint cleanup
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}
