package main

import (
	"database/sql"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/pkg/cachehandler"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/listenhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
	"monorepo/bin-number-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameNumberManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

const (
	defaultDatabaseDSN             = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultRedisAddress            = "127.0.0.1:6379"
	defaultRedisDatabase           = 1
	defaultRedisPassword           = ""

	defaultTwilioSID          = ""
	defaultTwilioToken        = ""
	defaultTelnyxConnectionID = ""
	defaultTelnyxProfileID    = ""
	defaultTelnyxToken        = ""
)

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""

	twilioSID          = ""
	twilioToken        = ""
	telnyxConnectionID = ""
	telnyxProfileID    = ""
	telnyxToken        = ""
)

func main() {
	log := logrus.WithField("func", "main")
	log.Debugf("Hello world. Starting number-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("The run returned error. err: %v", errRun)
	}
	<-chDone
}

// proces init
func init() {
	initVariable()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(prometheusEndpoint, prometheusListenAddress)

	logrus.Info("init finished.")
}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	pflag.String("database_dsn", defaultDatabaseDSN, "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
	pflag.String("redis_address", defaultRedisAddress, "Address of the Redis server (e.g., localhost:6379)")
	pflag.String("redis_password", defaultRedisPassword, "Password for authenticating with the Redis server (if required)")
	pflag.Int("redis_database", defaultRedisDatabase, "Redis database index to use (default is 1)")

	pflag.String("twilio_sid", defaultTwilioSID, "Password for authenticating with the Redis server (if required)")
	pflag.String("twilio_token", defaultTwilioToken, "Password for authenticating with the Redis server (if required)")
	pflag.String("telnyx_connection_id", defaultTelnyxConnectionID, "Password for authenticating with the Redis server (if required)")
	pflag.String("telnyx_profile_id", defaultTelnyxProfileID, "Password for authenticating with the Redis server (if required)")
	pflag.String("telnyx_token", defaultTelnyxToken, "Password for authenticating with the Redis server (if required)")

	pflag.Parse()

	// rabbitmq_address
	if errFlag := viper.BindPFlag("rabbitmq_address", pflag.Lookup("rabbitmq_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQAddress = viper.GetString("rabbitmq_address")

	// prometheus_endpoint
	if errFlag := viper.BindPFlag("prometheus_endpoint", pflag.Lookup("prometheus_endpoint")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusEndpoint = viper.GetString("prometheus_endpoint")

	// prometheus_listen_address
	if errFlag := viper.BindPFlag("prometheus_listen_address", pflag.Lookup("prometheus_listen_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusListenAddress = viper.GetString("prometheus_listen_address")

	// database_dsn
	if errFlag := viper.BindPFlag("database_dsn", pflag.Lookup("database_dsn")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("database_dsn", "DATABASE_DSN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	databaseDSN = viper.GetString("database_dsn")

	// redis_address
	if errFlag := viper.BindPFlag("redis_address", pflag.Lookup("redis_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_address", "REDIS_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisAddress = viper.GetString("redis_address")

	// redis_password
	if errFlag := viper.BindPFlag("redis_password", pflag.Lookup("redis_password")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_password", "REDIS_PASSWORD"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisPassword = viper.GetString("redis_password")

	// redis_database
	if errFlag := viper.BindPFlag("redis_database", pflag.Lookup("redis_database")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_database", "REDIS_DATABASE"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisDatabase = viper.GetInt("redis_database")

	// twilio_sid
	if errFlag := viper.BindPFlag("twilio_sid", pflag.Lookup("twilio_sid")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("twilio_sid", "TWILIO_SID"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	twilioSID = viper.GetString("twilio_sid")

	// twilio_token
	if errFlag := viper.BindPFlag("twilio_token", pflag.Lookup("twilio_token")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("twilio_token", "TWILIO_TOKEN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	twilioToken = viper.GetString("twilio_token")

	// telnyx_connection_id
	if errFlag := viper.BindPFlag("telnyx_connection_id", pflag.Lookup("telnyx_connection_id")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("telnyx_connection_id", "TELNYX_CONNECTION_ID"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	telnyxConnectionID = viper.GetString("telnyx_connection_id")

	// telnyx_profile_id
	if errFlag := viper.BindPFlag("telnyx_profile_id", pflag.Lookup("telnyx_profile_id")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("telnyx_profile_id", "TELNYX_PROFILE_ID"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	telnyxProfileID = viper.GetString("telnyx_profile_id")

	// telnyx_token
	if errFlag := viper.BindPFlag("telnyx_token", pflag.Lookup("telnyx_token")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("telnyx_token", "TELNYX_TOKEN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	telnyxToken = viper.GetString("telnyx_token")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal inits sinal settings.
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

// run runs the listen
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameNumberEvent, serviceName)

	nHandlerTelnyx := numberhandlertelnyx.NewNumberHandler(reqHandler, db, telnyxConnectionID, telnyxProfileID, telnyxToken)
	nHandlerTwilio := numberhandlertwilio.NewNumberHandler(reqHandler, db, twilioSID, twilioToken)

	numberHandler := numberhandler.NewNumberHandler(reqHandler, db, notifyHandler, nHandlerTelnyx, nHandlerTwilio)

	if err := runListen(sockHandler, numberHandler); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, numberHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, numberHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameNumberSubscribe), subscribeTargets, numberHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
