package main

import (
	"fmt"
	"os"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/pkg/listenhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"
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

	awsAccessKey     = ""
	awsSecretKey     = ""
	elevenlabsAPIKey = ""
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

	localAddress := os.Getenv("POD_IP")
	podID := os.Getenv("HOSTNAME")
	listenAddress := fmt.Sprintf("%s:8080", localAddress)

	ttsHandler := ttshandler.NewTTSHandler(awsAccessKey, awsSecretKey, gcpCredentialBase64, gcpProjectID, gcpBucketName, "/shared-data", localAddress, reqHandler, notifyHandler)
	streamingHandler := streaminghandler.NewStreamingHandler(reqHandler, notifyHandler, listenAddress, podID, elevenlabsAPIKey)

	// run listener
	go runListen(sockHandler, ttsHandler, streamingHandler, podID)
	go runStreaming(streamingHandler)

	return nil
}

// runListen run the listener
func runListen(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler, podID string) {

	if errRun := runListenNormal(sockHandler, ttsHandler, streamingHandler); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run listen handler in normal mode"))
	}

	if errRun := runListenPod(sockHandler, ttsHandler, streamingHandler, podID); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run listen handler in pod mode"))
	}
}

func runListenNormal(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler) error {

	listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler, streamingHandler)

	// run
	if errRun := listenHandler.Run(string(commonoutline.QueueNameTTSRequest), string(commonoutline.QueueNameDelay)); errRun != nil {
		return errors.Wrapf(errRun, "could not run listen handler in normal mode")
	}

	return nil
}

// runListen run the listener
func runListenPod(sockHandler sockhandler.SockHandler, ttsHandler ttshandler.TTSHandler, streamingHandler streaminghandler.StreamingHandler, podID string) error {
	listenHandler := listenhandler.NewListenHandler(sockHandler, ttsHandler, streamingHandler)

	queueName := fmt.Sprintf("%s.%s", commonoutline.QueueNameTTSRequest, podID)
	if err := listenHandler.Run(queueName, string(commonoutline.QueueNameDelay)); err != nil {
		return errors.Wrapf(err, "could not run listen handler in pod mode")
	}

	return nil
}

func runStreaming(streamingHandler streaminghandler.StreamingHandler) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStreaming",
	})

	log.Debugf("Starting streaming handler.")
	if errRun := streamingHandler.Run(); errRun != nil {
		panic(errors.Wrapf(errRun, "could not run streaming handler"))
	}
}
