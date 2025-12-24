package main

import (
	"database/sql"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/cachehandler"
	"monorepo/bin-agent-manager/pkg/dbhandler"
	"monorepo/bin-agent-manager/pkg/listenhandler"
	"monorepo/bin-agent-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameAgentManager

var (
	databaseDSN             = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

func main() {
	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if err := run(sqlDB, cache); err != nil {
		log.Errorf("Run func has finished. err: %v", err)
	}
	<-chDone
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the listen
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	db := dbhandler.NewHandler(sqlDB, cache)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameAgentEvent, serviceName)
	agentHandler := agenthandler.NewAgentHandler(reqHandler, db, notifyHandler)

	if err := runListen(sockHandler, agentHandler); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, agentHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, agentHandler agenthandler.AgentHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, agentHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameAgentRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	agentHandler agenthandler.AgentHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameWebhookEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameAgentSubscribe), subscribeTargets, agentHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
