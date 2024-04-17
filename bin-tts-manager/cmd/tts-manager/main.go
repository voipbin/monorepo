package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/ttshandler"
)

const serviceName = commonoutline.ServiceNameTTSManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// gcp info
var gcpCredential = flag.String("gcp_credential", "./credential.json", "the GCP credential file path")
var gcpProjectID = flag.String("gcp_project_id", "project", "the gcp project id")
var gcpBucketName = flag.String("gcp_bucket_name", "bucket", "the gcp bucket name to use")

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

// proces init
func init() {
	flag.Parse()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(*promEndpoint, *promListenAddr)

	logrus.Infof("init finished. credential: %s, prom_list: %s, rabbit_addr: %s, bucket_name: %s",
		*gcpCredential, *promListenAddr, *rabbitAddr, *gcpBucketName)
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal inits sinal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go signalHandler()
}

// initProm inits prometheus settings
func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			err := http.ListenAndServe(listen, nil)
			if err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second * 1)
				continue
			}
			break
		}
	}()
}

// Run the services
func run() error {
	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create listen handler
	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)
	notifyHandler := notifyhandler.NewNotifyHandler(rabbitSock, reqHandler, commonoutline.QueueNameTTSEvent, serviceName)

	// run listener
	if err := runListen(rabbitSock, notifyHandler); err != nil {
		return err
	}

	return nil
}

// runListen run the listener
func runListen(rabbitSock rabbitmqhandler.Rabbit, notifyHandler notifyhandler.NotifyHandler) error {

	// get pod ip
	localAddress := os.Getenv("POD_IP")

	// create tts handler
	ttsHandler := ttshandler.NewTTSHandler(*gcpCredential, *gcpProjectID, *gcpBucketName, "/shared-data", localAddress)
	if ttsHandler == nil {
		logrus.Errorf("Could not create tts handler.")
		return fmt.Errorf("could not create tts handler")
	}

	listenHandler := listenhandler.NewListenHandler(rabbitSock, ttsHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameTTSRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
