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

	"monorepo/bin-route-manager/pkg/cachehandler"
	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/listenhandler"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

const (
	serviceName = commonoutline.ServiceNameRouteManager
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
	sqlDB, err := sql.Open("mysql", databaseDSN)
	if err != nil {
		log.Errorf("Could not access to database. err: %v", err)
		return
	} else if err := sqlDB.Ping(); err != nil {
		log.Errorf("Could not set the connection correctly. err: %v", err)
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

// run runs the call-manager
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create handlers
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameRouteEvent, serviceName)

	routeHandler := routehandler.NewRouteHandler(db, reqHandler, notifyHandler)
	providerHandler := providerhandler.NewProviderHandler(db, reqHandler, notifyHandler)

	// run request listener
	if err := runRequestListen(sockHandler, providerHandler, routeHandler); err != nil {
		return err
	}

	return nil
}

// runRequestListen runs the request listen service
func runRequestListen(sockHandler sockhandler.SockHandler, providerHandler providerhandler.ProviderHandler, routeHandler routehandler.RouteHandler) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, providerHandler, routeHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameRouteRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
