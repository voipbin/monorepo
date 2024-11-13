package main

import (
	"database/sql"
	"fmt"
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

	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/listenhandler"
	"monorepo/bin-registrar-manager/pkg/subscribehandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

const serviceName = commonoutline.ServiceNameRegistrarManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

const (
	defaultDatabaseDSNAsterisk     = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultDatabaseDSNBin          = "testid:testpassword@tcp(127.0.0.1:3306)/test"
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultRedisAddress            = "127.0.0.1:6379"
	defaultRedisDatabase           = 1
	defaultRedisPassword           = ""
)

var (
	databaseDSNAsterisk     = ""
	databaseDSNBin          = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
)

func main() {
	log := logrus.WithFields(logrus.Fields{
		"func": "main",
	})
	fmt.Printf("hello world\n")

	// connect to the database asterisk
	sqlAst, err := sql.Open("mysql", databaseDSNAsterisk)
	if err != nil {
		log.Errorf("Could not access to database asterisk. err: %v", err)
		return
	}
	defer sqlAst.Close()

	// connect to the database bin-manager
	sqlBin, err := sql.Open("mysql", databaseDSNBin)
	if err != nil {
		log.Errorf("Could not access to database bin-manager. err: %v", err)
		return
	}
	defer sqlBin.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlAst, sqlBin, cache); errRun != nil {
		log.Errorf("Could not run. err: %v", errRun)
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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	pflag.String("database_dsn_asterisk", defaultDatabaseDSNAsterisk, "Data Source Name for database connection-asterisk (e.g., user:password@tcp(localhost:3306)/dbname)")
	pflag.String("database_dsn_bin", defaultDatabaseDSNBin, "Data Source Name for database connection-bin (e.g., user:password@tcp(localhost:3306)/dbname)")
	pflag.String("redis_address", defaultRedisAddress, "Address of the Redis server (e.g., localhost:6379)")
	pflag.String("redis_password", defaultRedisPassword, "Password for authenticating with the Redis server (if required)")
	pflag.Int("redis_database", defaultRedisDatabase, "Redis database index to use (default is 1)")
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

	// database_dsn_asterisk
	if errFlag := viper.BindPFlag("database_dsn_asterisk", pflag.Lookup("database_dsn_asterisk")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("database_dsn_asterisk", "DATABASE_DSN_ASTERISK"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	databaseDSNAsterisk = viper.GetString("database_dsn_asterisk")

	// database_dsn_bin
	if errFlag := viper.BindPFlag("database_dsn_bin", pflag.Lookup("database_dsn_bin")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("database_dsn_bin", "DATABASE_DSN_BIN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	databaseDSNBin = viper.GetString("database_dsn_bin")

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

// NewWorker creates worker interface
func run(sqlAst *sql.DB, sqlBin *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	dbAst := dbhandler.NewHandler(sqlAst, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler)
	trunkHandler := trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler)
	contactHandler := contacthandler.NewContactHandler(reqHandler, dbAst, dbBin)

	// run listen
	if errListen := runListen(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler); errListen != nil {
		log.Errorf("Could not run the listener. err: %v", errListen)
		return errListen
	}

	// run subscriber
	if errSubscribe := runSubscribe(sockHandler, extensionHandler, trunkHandler); errSubscribe != nil {
		log.Errorf("Could not run the subscriber. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, trunkHandler trunkhandler.TrunkHandler, extensionHandler extensionhandler.ExtensionHandler, contactHandler contacthandler.ContactHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRegistrarRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, extensionHandler extensionhandler.ExtensionHandler, trunkHandler trunkhandler.TrunkHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debugf("Subscribe target details. len: %d", len(subscribeTargets))

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameRegistrarSubscribe), subscribeTargets, extensionHandler, trunkHandler)

	// run
	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}
