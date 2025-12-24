package main

import (
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/pkg/cachehandler"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/listenhandler"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
	"monorepo/bin-queue-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameQueueManager

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
	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(databaseDSN)
	if err != nil {
		log.Errorf("Could not connect to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	db := dbhandler.NewHandler(sqlDB, cache)

	if err := run(db); err != nil {
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
func run(db dbhandler.DBHandler) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "run",
		},
	)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameQueueEvent, serviceName)
	queueHandler := queuehandler.NewQueueHandler(reqHandler, db, notifyHandler)
	queuecallHandler := queuecallhandler.NewQueuecallHandler(reqHandler, db, notifyHandler, queueHandler)

	// run listen
	if err := runListen(sockHandler, queueHandler, queuecallHandler); err != nil {
		log.Errorf("Could not run listen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(sockHandler, queueHandler, queuecallHandler); err != nil {
		log.Errorf("Could not run subscribe. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameAgentEvent),
		string(commonoutline.QueueNameConferenceEvent),
	}

	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameQueueSubscribe),
		subscribeTargets,
		queueHandler,
		queuecallHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(
	sockHandler sockhandler.SockHandler,
	queueHandler queuehandler.QueueHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, queueHandler, queuecallHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameQueueRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
