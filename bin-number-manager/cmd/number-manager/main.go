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

	"monorepo/bin-number-manager/pkg/cachehandler"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/listenhandler"
	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
	"monorepo/bin-number-manager/pkg/subscribehandler"
)

const serviceName = commonoutline.ServiceNameNumberManager

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

	twilioSID          = ""
	twilioToken        = ""
	telnyxConnectionID = ""
	telnyxProfileID    = ""
	telnyxToken        = ""
)

func main() {
	log := logrus.WithField("func", "main")
	log.Debugf("Hello world. Starting number-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("The run returned error. err: %v", errRun)
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
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameNumberEvent, serviceName)

	nHandlerTelnyx := numberhandlertelnyx.NewNumberHandler(reqHandler, db, telnyxConnectionID, telnyxProfileID, telnyxToken)
	nHandlerTwilio := numberhandlertwilio.NewNumberHandler(reqHandler, db, twilioSID, twilioToken)

	numberHandler := numberhandler.NewNumberHandler(reqHandler, db, notifyHandler, nHandlerTelnyx, nHandlerTwilio)

	if err := runListen(sockHandler, numberHandler); err != nil {
		return err
	}

	if err := runSubscribe(sockHandler, numberHandler); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, numberHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameNumberRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, numberHandler numberhandler.NumberHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameNumberSubscribe), subscribeTargets, numberHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
