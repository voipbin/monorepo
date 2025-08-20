package main

import (
	"database/sql"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/listenhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	"monorepo/bin-call-manager/pkg/subscribehandler"
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

	homerAPIAddress = ""
	homerAuthToken  = ""
	homerWhitelist  = []string{}
)

func main() {
	log := logrus.WithField("func", "main")

	// connect to database
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	} else if err := sqlDB.Ping(); err != nil {
		log.Errorf("Could not set the connection correctly. err: %v", err)
		return
	}
	defer sqlDB.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	// run
	if errRun := run(sqlDB, cache); errRun != nil {
		log.Errorf("Could not run the process correctly. err: %v", errRun)
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
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, common.Servicename)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameCallEvent, common.Servicename)
	channelHandler := channelhandler.NewChannelHandler(reqHandler, notifyHandler, db)
	bridgeHandler := bridgehandler.NewBridgeHandler(reqHandler, notifyHandler, db)
	externalMediaHandler := externalmediahandler.NewExternalMediaHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	recordingHandler := recordinghandler.NewRecordingHandler(reqHandler, notifyHandler, db, channelHandler, bridgeHandler)
	confbridgeHandler := confbridgehandler.NewConfbridgeHandler(reqHandler, notifyHandler, db, cache, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)
	groupcallHandler := groupcallhandler.NewGroupcallHandler(reqHandler, notifyHandler, db)
	recoveryHandler := callhandler.NewRecoveryHandler(reqHandler, homerAPIAddress, homerAuthToken, homerWhitelist)
	callHandler := callhandler.NewCallHandler(reqHandler, notifyHandler, db, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler, groupcallHandler, recoveryHandler)
	ariEventHandler := arieventhandler.NewEventHandler(sockHandler, db, cache, reqHandler, notifyHandler, callHandler, confbridgeHandler, channelHandler, bridgeHandler, recordingHandler, externalMediaHandler)

	// run subscribe listener
	if err := runSubscribe(sockHandler, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler); err != nil {
		return err
	}

	// run request listener
	if err := runRequestListen(sockHandler, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler); err != nil {
		return err
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	ariEventHandler arieventhandler.ARIEventHandler,
	callHandler callhandler.CallHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameAsteriskEventAll),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameSentinelEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debug("Running subscribe handler")

	ariEventListenHandler := subscribehandler.NewSubscribeHandler(sockHandler, commonoutline.QueueNameCallSubscribe, subscribeTargets, ariEventHandler, callHandler, groupcallHandler, confbridgeHandler)

	// run
	if err := ariEventListenHandler.Run(); err != nil {
		log.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	sockHandler sockhandler.SockHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
	groupcallHandler groupcallhandler.GroupcallHandler,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runRequestListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, callHandler, confbridgeHandler, channelHandler, recordingHandler, externalMediaHandler, groupcallHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameCallRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
