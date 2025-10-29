package main

import (
	"database/sql"
	"fmt"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-pipecat-manager/pkg/cachehandler"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/listenhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"os"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	serviceName = commonoutline.ServiceNamePipecatManager
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
	log.Info("Starting pipecat-manager.")

	if errRun := run(); errRun != nil {
		log.Errorf("Could not run the main thread. err: %v", errRun)
	}
	<-chDone

	log.Info("Finished pipecat-manager.")
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// run runs the main thread.
func run() error {
	log := logrus.WithField("func", "run")

	dbHandler, err := createDBHandler()
	if err != nil {
		return errors.Wrapf(err, "could not create dbhandler")
	}
	log.Debugf("Connected to database and cache server.")

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	listenIP := os.Getenv("POD_IP")
	if listenIP == "" {
		return fmt.Errorf("could not get the listen ip address")
	}
	listenAddress := fmt.Sprintf("%s:%d", listenIP, 8080)

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNamePipecatEvent, serviceName)

	pipecatcallHandler := pipecatcallhandler.NewPipecatcallHandler(requestHandler, notifyHandler, dbHandler, listenAddress, listenIP)

	// run listen
	if errListen := runListen(sockHandler, listenIP, pipecatcallHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errors.Wrapf(errListen, "could not start runListen")
	}

	// run streaming
	if errStreaming := runStreaming(pipecatcallHandler); errStreaming != nil {
		log.Errorf("Could not start runStreaming. err: %v", errStreaming)
		return errors.Wrapf(errStreaming, "could not start runStreaming")
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	hostID string,
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, pipecatcallHandler)

	// run
	listenQueue := fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, hostID)
	if err := listenHandler.Run(string("bin-manager.pipecat-manager.request"), listenQueue, string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run the listenhandler correctly")
	}

	return nil
}

func runStreaming(pipecatcallHandler pipecatcallhandler.PipecatcallHandler) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})

	go func() {
		if errRun := pipecatcallHandler.Run(); errRun != nil {
			log.Errorf("Could not run the streaming handler correctly. err: %v", errRun)
		}
	}()

	return nil
}

// connectDatabase connects to the database and cachehandler
func createDBHandler() (dbhandler.DBHandler, error) {
	// connect to database
	db, err := sql.Open("mysql", databaseDSN)
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
