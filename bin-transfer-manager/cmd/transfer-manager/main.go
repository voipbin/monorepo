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

	"monorepo/bin-transfer-manager/pkg/cachehandler"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
	"monorepo/bin-transfer-manager/pkg/listenhandler"
	"monorepo/bin-transfer-manager/pkg/subscribehandler"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

const (
	serviceName = "transfer-manager"
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
)

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

// run runs the transfer-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTransferEvent, serviceName)
	transferHandler := transferhandler.NewTransferHandler(reqHandler, notifyHandler, db)

	// run event listener
	if err := runSubscribe(serviceName, sockHandler, transferHandler); err != nil {
		return err
	}

	// run request listener
	if err := runRequestListen(sockHandler, transferHandler); err != nil {
		return err
	}

	return nil
}

// runSubscribe runs the ARI event listen service
func runSubscribe(
	serviceName string,
	sockHandler sockhandler.SockHandler,
	transferHandler transferhandler.TransferHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
	}

	subscribeHandler := subscribehandler.NewSubscribeHandler(serviceName, sockHandler, string(commonoutline.QueueNameTransferSubscribe), subscribeTargets, transferHandler)

	// run
	if err := subscribeHandler.Run(); err != nil {
		logrus.Errorf("Could not run the ari event listen handler correctly. err: %v", err)
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(
	sockHandler sockhandler.SockHandler,
	transferHandler transferhandler.TransferHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameTransferRequest),
		string(commonoutline.QueueNameDelay),
		transferHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
