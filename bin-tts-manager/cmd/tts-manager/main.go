package main

import (
	"fmt"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/pkg/listenhandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"
)

const serviceName = commonoutline.ServiceNameTTSManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

var (
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""

	gcpCredentialBase64 = ""
	gcpProjectID        = ""
	gcpBucketName       = ""

	awsAccessKey = ""
	awsSecretKey = ""
)

func main() {
	log := logrus.WithFields(logrus.Fields{
		"func": "main",
	})

	if errRun := run(); errRun != nil {
		log.Errorf("Could not run. err: %v", errRun)
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

// Run the services
func run() error {
	// rabbitmq sock connect
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, rabbitMQAddress)
	sockHandler.Connect()

	// create listen handler
	reqHandler := requesthandler.NewRequestHandler(sockHandler, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(sockHandler, reqHandler, commonoutline.QueueNameTTSEvent, serviceName)

	// run listener
	if err := runListen(sockHandler, notifyHandler); err != nil {
		return err
	}

	return nil
}

// runListen run the listener
func runListen(sockHandler sockhandler.SockHandler, notifyHandler notifyhandler.NotifyHandler) error {

	// get pod ip
	localAddress := os.Getenv("POD_IP")

	// create tts handler
	ttsHandler := ttshandler.NewTTSHandler(awsAccessKey, awsSecretKey, gcpCredentialBase64, gcpProjectID, gcpBucketName, "/shared-data", localAddress, notifyHandler)
	if ttsHandler == nil {
		logrus.Errorf("Could not create tts handler.")
		return fmt.Errorf("could not create tts handler")
	}

	listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameTTSRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
