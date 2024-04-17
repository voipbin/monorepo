package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

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

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// args for database
var dbDSN = flag.String("dbDSN", "testid:testpassword@tcp(127.0.0.1:3306)/test", "database dsn for conference-manager.")

// args for redis
var redisAddr = flag.String("redis_addr", "127.0.0.1:6379", "redis address.")
var redisPassword = flag.String("redis_password", "", "redis password")
var redisDB = flag.Int("redis_db", 1, "redis database.")

// args for chatbot engines
var engineKeyChatgpt = flag.String("engine_key_chatgpt", "", "engine key for chatbot engine chatgpt")

const (
	serviceName = commonoutline.ServiceNameChatbotManager
)

func main() {
	log := logrus.WithField("func", "main")
	log.Info("Starting chatbot-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", *dbDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(*redisAddr, *redisPassword, *redisDB)
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
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	logrus.Info("Init finished.")
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

// run runs the main thread.
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	requestHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, requestHandler, commonoutline.QueueNameChatbotEvent, serviceName)

	chatbotHandler := chatbothandler.NewChatbotHandler(requestHandler, notifyHandler, db)

	chatgptHandler := chatgpthandler.NewChatgptHandler(*engineKeyChatgpt)
	chatbotcallHandler := chatbotcallhandler.NewChatbotcallHandler(requestHandler, notifyHandler, db, chatbotHandler, chatgptHandler)

	// run listen
	if errListen := runListen(rabbitSock, chatbotHandler, chatbotcallHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errListen
	}

	// run subscribe
	if errSubscribe := runSubscribe(rabbitSock, chatbotcallHandler); errSubscribe != nil {
		log.Errorf("Could not start runSubscribe. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	rabbitSock rabbitmqhandler.Rabbit,
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameTranscribeEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		string(serviceName),
		rabbitSock,
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
	rabbitSock rabbitmqhandler.Rabbit,
	chatbotHandler chatbothandler.ChatbotHandler,
	chatbotcallhandler chatbotcallhandler.ChatbotcallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(
		rabbitSock,
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
