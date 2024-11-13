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

	"monorepo/bin-chatbot-manager/pkg/cachehandler"
	"monorepo/bin-chatbot-manager/pkg/chatbotcallhandler"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-chatbot-manager/pkg/listenhandler"
	"monorepo/bin-chatbot-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameChatbotManager
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
	defaultEngineKeyChatGPT        = ""
)

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
	engineKeyChatgpt        = ""
)

func main() {
	log := logrus.WithField("func", "main")
	log.Info("Starting chatbot-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	_ = run(sqlDB, cache)
	<-chDone

	log.Info("Finishing conference-manager.")
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

	logrus.Info("Init finished.")
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
	pflag.String("engine_key_chatgpt", defaultEngineKeyChatGPT, "Engine key for chatgpt.")
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

	// engine_key_chatgpt
	if errFlag := viper.BindPFlag("engine_key_chatgpt", pflag.Lookup("engine_key_chatgpt")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("engine_key_chatgpt", "ENGINE_KEY_CHATGPT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	engineKeyChatgpt = viper.GetString("redis_password")

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

// run runs the main thread.
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNameChatbotEvent, serviceName)

	chatbotHandler := chatbothandler.NewChatbotHandler(requestHandler, notifyHandler, db)

	chatgptHandler := chatgpthandler.NewChatgptHandler(engineKeyChatgpt)
	chatbotcallHandler := chatbotcallhandler.NewChatbotcallHandler(requestHandler, notifyHandler, db, chatbotHandler, chatgptHandler)

	// run listen
	if errListen := runListen(sockHandler, chatbotHandler, chatbotcallHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errListen
	}

	// run subscribe
	if errSubscribe := runSubscribe(sockHandler, chatbotcallHandler); errSubscribe != nil {
		log.Errorf("Could not start runSubscribe. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameTranscribeEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		string(serviceName),
		sockHandler,
		string(commonoutline.QueueNameChatbotSubscribe),
		subscribeTargets,
		chatbotcallHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		logrus.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	chatbotHandler chatbothandler.ChatbotHandler,
	chatbotcallhandler chatbotcallhandler.ChatbotcallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameChatbotRequest),
		string(commonoutline.QueueNameDelay),
		chatbotHandler,
		chatbotcallhandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
