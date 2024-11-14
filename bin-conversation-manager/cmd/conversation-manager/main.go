package main

import (
	"database/sql"
	"fmt"
	"os"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/listenhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/subscribehandler"
)

const serviceName = "conversation-manager"

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
	db, err := sql.Open("mysql", databaseDSN)
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameConversationEvent, serviceName)

	lineHandler := linehandler.NewLineHandler()
	accountHandler := accounthandler.NewAccountHandler(dbHandler, reqHandler, notifyHandler, lineHandler)
	smsHandler := smshandler.NewSMSHandler(reqHandler, accountHandler)

	messageHandler := messagehandler.NewMessageHandler(dbHandler, notifyHandler, accountHandler, lineHandler, smsHandler)
	conversationHandler := conversationhandler.NewConversationHandler(dbHandler, notifyHandler, accountHandler, messageHandler, lineHandler, smsHandler)

	// run listen
	if errListen := runListen(sockHandler, accountHandler, conversationHandler, messageHandler); errListen != nil {
		log.Errorf("Could not run the listen correctly. err: %v", errListen)
		return
	}

	// run subscribe
	if errSub := runSubscribe(sockHandler, accountHandler, conversationHandler); errSub != nil {
		log.Errorf("Could not run the subscribe correctly. err: %v", errSub)
		return
	}
}

// runListen runs the listen service
func runListen(
	sockListen sockhandler.SockHandler,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
	messageHandler messagehandler.MessageHandler,
) error {
	log := logrus.WithField("func", "runListen")

	listenHandler := listenhandler.NewListenHandler(sockListen, accountHandler, conversationHandler, messageHandler)

	// run the service
	if errRun := listenHandler.Run(string(commonoutline.QueueNameConversationRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Error occurred in listen handler. err: %v", errRun)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	accountHandler accounthandler.AccountHandler,
	conversationHandler conversationhandler.ConversationHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameMessageEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameConversationSubscribe),
		subscribeTargets,
		accountHandler,
		conversationHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
