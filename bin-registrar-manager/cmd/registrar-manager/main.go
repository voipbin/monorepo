package main

import (
	"database/sql"
	"fmt"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/pkg/cachehandler"
	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/listenhandler"
	"monorepo/bin-registrar-manager/pkg/subscribehandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
)

const serviceName = commonoutline.ServiceNameRegistrarManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var (
	databaseDSNAsterisk     = ""
	databaseDSNBin          = ""
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
	redisAddress            = ""
	redisDatabase           = 0
	redisPassword           = ""
)

func main() {
	log := logrus.WithFields(logrus.Fields{
		"func": "main",
	})
	fmt.Printf("hello world\n")

	// connect to the database asterisk
	sqlAst, err := sql.Open("mysql", databaseDSNAsterisk)
	if err != nil {
		log.Errorf("Could not access to database asterisk. err: %v", err)
		return
	}
	defer sqlAst.Close()

	// connect to the database bin-manager
	sqlBin, err := sql.Open("mysql", databaseDSNBin)
	if err != nil {
		log.Errorf("Could not access to database bin-manager. err: %v", err)
		return
	}
	defer sqlBin.Close()

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		log.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	if errRun := run(sqlAst, sqlBin, cache); errRun != nil {
		log.Errorf("Could not run. err: %v", errRun)
	}
	<-chDone

}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// NewWorker creates worker interface
func run(sqlAst *sql.DB, sqlBin *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	dbAst := dbhandler.NewHandler(sqlAst, cache)
	dbBin := dbhandler.NewHandler(sqlBin, cache)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRegistrarEvent, serviceName)
	extensionHandler := extensionhandler.NewExtensionHandler(reqHandler, dbAst, dbBin, notifyHandler)
	trunkHandler := trunkhandler.NewTrunkHandler(reqHandler, dbBin, notifyHandler)
	contactHandler := contacthandler.NewContactHandler(reqHandler, dbAst, dbBin)

	// run listen
	if errListen := runListen(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler); errListen != nil {
		log.Errorf("Could not run the listener. err: %v", errListen)
		return errListen
	}

	// run subscriber
	if errSubscribe := runSubscribe(sockHandler, extensionHandler, trunkHandler); errSubscribe != nil {
		log.Errorf("Could not run the subscriber. err: %v", errSubscribe)
		return errSubscribe
	}

	return nil
}

// runListen runs the listen service
func runListen(sockHandler sockhandler.SockHandler, reqHandler requesthandler.RequestHandler, trunkHandler trunkhandler.TrunkHandler, extensionHandler extensionhandler.ExtensionHandler, contactHandler contacthandler.ContactHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runListen",
	})

	listenHandler := listenhandler.NewListenHandler(sockHandler, reqHandler, trunkHandler, extensionHandler, contactHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRegistrarRequest), string(commonoutline.QueueNameDelay)); err != nil {
		log.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, extensionHandler extensionhandler.ExtensionHandler, trunkHandler trunkhandler.TrunkHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runSubscribe",
	})

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	log.WithField("subscribe_targets", subscribeTargets).Debugf("Subscribe target details. len: %d", len(subscribeTargets))

	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, string(commonoutline.QueueNameRegistrarSubscribe), subscribeTargets, extensionHandler, trunkHandler)

	// run
	if err := subHandler.Run(); err != nil {
		log.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}
