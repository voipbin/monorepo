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

	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"monorepo/bin-storage-manager/pkg/filehandler"
	"monorepo/bin-storage-manager/pkg/listenhandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"monorepo/bin-storage-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameStorageManager
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

	gcpProjectID       = ""
	gcpBucketNameMedia = ""
	gcpBucketNameTmp   = ""
)

func main() {
	log := logrus.WithFields(logrus.Fields{
		"func": "main",
	})

	// create dbhandler
	dbHandler, err := createDBHandler()
	if err != nil {
		logrus.Errorf("Could not connect to the database or failed to initiate the cachehandler. err: ")
		return
	}

	if errRun := run(dbHandler); errRun != nil {
		log.Errorf("Could not run correctly. err: %v", errRun)
		return
	}
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

// Run the services
func run(dbHandler dbhandler.DBHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "run",
	})

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameStorageEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(notifyHandler, dbHandler)
	fileHandler := filehandler.NewFileHandler(notifyHandler, dbHandler, accountHandler, gcpProjectID, gcpBucketNameMedia, gcpBucketNameTmp)
	storageHandler := storagehandler.NewStorageHandler(reqHandler, fileHandler, gcpBucketNameMedia)

	// run listener
	if errListen := runListen(sockHandler, storageHandler, accountHandler); errListen != nil {
		log.Errorf("Could not run the listener correctly. err: %v", errListen)
		return errListen
	}

	// run sbuscriber
	if errSubs := runSubscribe(sockHandler, string(commonoutline.QueueNameStorageSubscribe), accountHandler, fileHandler); errSubs != nil {
		log.Errorf("Could not run the subscriber correctly. err: %v", errSubs)
		return errSubs
	}

	return nil
}

// runListen run the listener
func runListen(sockHandler sockhandler.SockHandler, storageHandler storagehandler.StorageHandler, accountHandler accounthandler.AccountHandler) error {

	// create listen handler
	listenHandler := listenhandler.NewListenHandler(sockHandler, storageHandler, accountHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameStorageRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		return errRun
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(sockHandler sockhandler.SockHandler, subscribeQueue string, accountHandler accounthandler.AccountHandler, fileHandler filehandler.FileHandler) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(sockHandler, subscribeQueue, subscribeTargets, accountHandler, fileHandler)

	// run
	if err := subHandler.Run(); err != nil {
		return err
	}

	return nil
}
