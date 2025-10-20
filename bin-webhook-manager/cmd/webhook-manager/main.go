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

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
	"monorepo/bin-webhook-manager/pkg/listenhandler"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

const serviceName = commonoutline.ServiceNameWebhookManager

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

	logrus.Info("Starting webhook-manager.")

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

	if errRun := run(sqlDB, cache); errRun != nil {
		logrus.Errorf("Could not run correctly. err: %v", errRun)
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

// run runs the webhook-manager
func run(db *sql.DB, cache cachehandler.CacheHandler) error {

	// run listen
	if err := runListen(db, cache); err != nil {
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameWebhookEvent, serviceName)
	accountHandler := accounthandler.NewAccountHandler(db, reqHandler)

	whHandler := webhookhandler.NewWebhookHandler(db, notifyHandler, accountHandler)

	listenHandler := listenhandler.NewListenHandler(sockHandler, whHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameWebhookRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
