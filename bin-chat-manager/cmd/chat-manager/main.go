package main

import (
	"fmt"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chat-manager/pkg/cachehandler"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
	"monorepo/bin-chat-manager/pkg/listenhandler"
	"monorepo/bin-chat-manager/pkg/messagechathandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

const serviceName = commonoutline.ServiceNameChatManager

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
)

func main() {
	fmt.Printf("Hello world!\n")

	// create dbhandler
	dbHandler, err := createDBHandler()
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: ")
		return
	}

	// run the service
	run(dbHandler)
	<-chDone
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	// connect to database
	db, err := commondatabasehandler.Connect(databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return nil, err
	}

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return nil, err
	}

	// create dbhandler
	dbHandler := dbhandler.NewHandler(db, cache)

	return dbHandler, nil
}

func run(dbHandler dbhandler.DBHandler) {
	log := logrus.WithField("func", "run")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameChatEvent, serviceName)

	chatroomHandler := chatroomhandler.NewChatroomHandler(dbHandler, reqHandler, notifyHandler)
	chatHandler := chathandler.NewChatHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler)

	messagechatroomHandler := messagechatroomhandler.NewMessagechatroomHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler)
	messagechatHandler := messagechathandler.NewMessagechatHandler(dbHandler, reqHandler, notifyHandler, chatroomHandler, messagechatroomHandler)

	// run listen
	if errListen := runListen(sockHandler, chatHandler, chatroomHandler, messagechatHandler, messagechatroomHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,

	chatHandler chathandler.ChatHandler,
	chatroomHandler chatroomhandler.ChatroomHandler,
	messagechatHandler messagechathandler.MessagechatHandler,
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, chatHandler, chatroomHandler, messagechatHandler, messagechatroomHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameChatRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}
