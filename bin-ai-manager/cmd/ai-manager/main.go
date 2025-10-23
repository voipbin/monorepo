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

	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_dialogflow_handler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-ai-manager/pkg/listenhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/subscribehandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
)

const (
	serviceName = commonoutline.ServiceNameAIManager
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
	log.Info("Starting ai-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	_ = run(sqlDB, cache)
	<-chDone

	log.Info("Finished ai-manager.")
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNameAIEvent, serviceName)

	aiHandler := aihandler.NewAIHandler(requestHandler, notifyHandler, db)

	engineOpenaiHandler := engine_openai_handler.NewEngineOpenaiHandler(engineKeyChatgpt)
	engineDialogflowHandler := engine_dialogflow_handler.NewEngineDialogflowHandler()

	messageHandler := messagehandler.NewMessageHandler(requestHandler, notifyHandler, db, engineOpenaiHandler, engineDialogflowHandler)
	aicallHandler := aicallhandler.NewAIcallHandler(requestHandler, notifyHandler, db, aiHandler, messageHandler)
	summaryHandler := summaryhandler.NewSummaryHandler(requestHandler, notifyHandler, db, engineOpenaiHandler)

	// run listen
	if errListen := runListen(sockHandler, aiHandler, aicallHandler, messageHandler, summaryHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errListen
	}

	// run subscribe
	if errSubscribe := runSubscribe(sockHandler, aicallHandler, summaryHandler, messageHandler); errSubscribe != nil {
		log.Errorf("Could not start runSubscribe. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	aicallHandler aicallhandler.AIcallHandler,
	summaryHandler summaryhandler.SummaryHandler,
	messageHandler messagehandler.MessageHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameTranscribeEvent),
		string(commonoutline.QueueNameTTSEvent),
		string(commonoutline.QueueNamePipecatEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		string(serviceName),
		sockHandler,
		string(commonoutline.QueueNameAISubscribe),
		subscribeTargets,
		aicallHandler,
		summaryHandler,
		messageHandler,
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
	aiHandler aihandler.AIHandler,
	aicallhandler aicallhandler.AIcallHandler,
	messageHandler messagehandler.MessageHandler,
	summaryHandler summaryhandler.SummaryHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameAIRequest),
		string(commonoutline.QueueNameDelay),
		aiHandler,
		aicallhandler,
		messageHandler,
		summaryHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
