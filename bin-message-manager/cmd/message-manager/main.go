package main

import (
	"database/sql"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-message-manager/pkg/requestexternal"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/pkg/cachehandler"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/listenhandler"
	"monorepo/bin-message-manager/pkg/messagehandler"
	"monorepo/bin-message-manager/pkg/messagehandlermessagebird"
)

const serviceName = commonoutline.ServiceNameMessageManager

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
	authtokenMessagebird    = ""
)

func main() {
	log := logrus.WithField("func", "main")
	log.Debugf("Starting message-manager.")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer sqlDB.Close()

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

// run runs the call-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {
	if err := runListen(db, cache); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen service
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameMessageEvent, serviceName)

	requestExternal := requestexternal.NewRequestExternal(authtokenMessagebird)
	messagehandlerMessagebird := messagehandlermessagebird.NewMessageHandlerMessagebird(reqHandler, db, requestExternal)

	messageHandler := messagehandler.NewMessageHandler(reqHandler, notifyHandler, db, messagehandlerMessagebird)
	listenHandler := listenhandler.NewListenHandler(sockHandler, messageHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameMessageRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
