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

	"monorepo/bin-conference-manager/pkg/cachehandler"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
	"monorepo/bin-conference-manager/pkg/listenhandler"
	"monorepo/bin-conference-manager/pkg/subscribehandler"
)

const (
	serviceName = commonoutline.ServiceNameConferenceManager
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
	log.Info("Starting conference-manager.")

	// connect to database
	sqlDB, err := commondatabasehandler.Connect(databaseDSN)
	if err != nil {
		logrus.Errorf("Could not access to database. err: %v", err)
		return
	}
	defer commondatabasehandler.Close(sqlDB)

	// connect to cache
	cache := cachehandler.NewHandler(redisAddress, redisPassword, redisDatabase)
	if err := cache.Connect(); err != nil {
		logrus.Errorf("Could not connect to cache server. err: %v", err)
		return
	}

	_ = run(sqlDB, cache)
	<-chDone

	log.Info("Finishing conference-manager.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the main thread.
func run(sqlDB *sql.DB, cache cachehandler.CacheHandler) error {
	log := logrus.WithField("func", "run")

	// dbhandler
	db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNameConferenceEvent, serviceName)

	conferenceHandler := conferencehandler.NewConferenceHandler(requestHandler, notifyHandler, db)
	conferencecallHandler := conferencecallhandler.NewConferencecallHandler(requestHandler, notifyHandler, db, conferenceHandler)

	// run listen
	if err := runListen(sockHandler, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runListen. err: %v", err)
		return err
	}

	// run subscribe
	if err := runSubscribe(sockHandler, conferenceHandler, conferencecallHandler); err != nil {
		log.Errorf("Could not start runSubscribe. err: %v", err)
		return err
	}

	return nil
}

// runSubscribe runs the subscribed event handler
func runSubscribe(
	sockHandler sockhandler.SockHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
	}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		string(commonoutline.QueueNameConferenceSubscribe),
		subscribeTargets,
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := subHandler.Run(); err != nil {
		logrus.Errorf("Could not run the subscribehandler correctly. err: %v", err)
		return err
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	conferenceHandler conferencehandler.ConferenceHandler,
	conferencecallHandler conferencecallhandler.ConferencecallHandler,
) error {

	listenHandler := listenhandler.NewListenHandler(
		sockHandler,
		string(commonoutline.QueueNameConferenceRequest),
		string(commonoutline.QueueNameDelay),
		conferenceHandler,
		conferencecallHandler,
	)

	// run
	if err := listenHandler.Run(); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
		return err
	}

	return nil
}
