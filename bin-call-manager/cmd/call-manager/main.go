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

	_ "github.com/go-sql-driver/mysql"
	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/listenhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	"monorepo/bin-call-manager/pkg/subscribehandler"
)

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
)

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
)

func main() {
	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	} else if err := sqlDB.Ping(); err != nil {
		log.Errorf("Could not set the connection correctly. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// run
	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("Could not run the process correctly. err: %v", errRun)
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
	pflag.String("database_dsn", defaultDatabaseDSN, "Data Source Name for database connection (e.g., user:password@tcp(localhost:3306)/dbname)")
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

// run runs the call-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, common.Servicename)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCallEvent, common.Servicename)
	channelHandler := channelhandler.NewChannelHandler(reqHandler, notifyHandler, db)
	bridgeHandler := bridgehandler.NewBridgeHandler(reqHandler, notifyHandler, db)
	externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	recordingHandler := recordinghandler.NewRecordingHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
	groupcallHandler := groupcallhandler.NewGroupcallHandler(reqHandler, notifyHandler, db)
	callHandler := callhandler.NewCallHandler(reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler, groupcallHandler)
	ariEventHandler := arieventhandler.NewEventHandler(sockHandler, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler)

	// run subscribe listener
	if err := runSubscribe(sockHandler, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler); err != nil {
		return err
	}

	// run request listener
	if err := runRequestListen(sockHandler, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler); err != nil {
		return err
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameAsteriskEventAll),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameFlowEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	ariEventListenHandler := subscribehandler.NewSubscribeHandler(sockHandler, commonoutline.QueueNameCallSubscribe, subscribeTargets, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler)

	// run
	if err := ariEventListenHandler.Run(); err != nil {
		log.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	sockHandler sockhandler.SockHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runRequestListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameCallRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
