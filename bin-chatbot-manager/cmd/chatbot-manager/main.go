package main

import (
	"database/sql"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/pkg/cachehandler"
	"monorepo/bin-chatbot-manager/pkg/chatbotcallhandler"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-chatbot-manager/pkg/listenhandler"
	"monorepo/bin-chatbot-manager/pkg/messagehandler"
	"monorepo/bin-chatbot-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameChatbotManager
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

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

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
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
	messageHandler := messagehandler.NewMessageHandler(notifyHandler, db, chatbotcallHandler, chatgptHandler)

	// run listen
	if errListen := runListen(sockHandler, chatbotHandler, chatbotcallHandler, messageHandler); errListen != nil {
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
	messageHandler messagehandler.MessageHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameChatbotRequest),
		string(commonoutline.QueueNameDelay),
		chatbotHandler,
		chatbotcallhandler,
		messageHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
