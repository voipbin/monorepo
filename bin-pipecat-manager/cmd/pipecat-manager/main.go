package main

import (
	"fmt"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-pipecat-manager/pkg/listenhandler"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"os"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// serviceName = commonoutline.ServiceNamePipecatManager
	serviceName = "pipecat-manager"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var (
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""
)

func main() {
	log := logrus.WithField("func", "main")
	log.Info("Starting pipecat-manager.")

	if errRun := run(); errRun != nil {
		log.Errorf("Could not run the main thread. err: %v", errRun)
	}
	<-chDone

	log.Info("Finished ai-manager.")
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

	// // dbhandler
	// db := dbhandler.NewHandler(sqlDB, cache)

	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	hostID := uuid.Must(uuid.NewV4())
	log.Debugf("Generated host id. host_id: %s", hostID)

	listenIP := os.Getenv("POD_IP")
	if listenIP == "" {
		return fmt.Errorf("could not get the listen ip address")
	}
	listenAddress := fmt.Sprintf("%s:%d", listenIP, 8080)

	requestHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, requestHandler, commonoutline.QueueNameAIEvent, serviceName)

	pipecatcallHandler := pipecatcallhandler.NewPipecatcallHandler(requestHandler, notifyHandler, listenAddress, hostID)

	// run listen
	if errListen := runListen(sockHandler, hostID, pipecatcallHandler); errListen != nil {
		log.Errorf("Could not start runListen. err: %v", errListen)
		return errListen
	}

	return nil
}

// runListen runs the listen handler
func runListen(
	sockHandler sockhandler.SockHandler,
	hostID uuid.UUID,
	pipecatcallHandler pipecatcallhandler.PipecatcallHandler,
) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, pipecatcallHandler)

	// run
	listenQueue := fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID)
	if err := listenHandler.Run(string("bin-manager.pipecat-manager.request"), listenQueue, string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run the listenhandler correctly")
	}

	return nil
}
