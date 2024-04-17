package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/pkg/filehandler"
	"monorepo/bin-storage-manager/pkg/listenhandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
)

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
var gcpBucketMedia = flag.String("gcp_bucket_media", "bucket", "the gcp bucket name for media storage")
var gcpBucketTmp = flag.String("gcp_bucket_tmp", "bucket", "the gcp bucket name for tmp storage")

const (
	serviceName = commonoutline.ServiceNameStorageManager
)

func main() {

	run()
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
		*gcpCredential, *promListenAddr, *rabbitAddr, *gcpBucketMedia)
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

	// run listener
	if err := runListen(); err != nil {
		return err
	}

	return nil
}

// runListen run the listener
func runListen() error {
	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	reqHandler := requesthandler.NewRequestHandler(rabbitSock, serviceName)

	// create bucket handler
	bucketHandler := filehandler.NewFileHandler(*gcpCredential, *gcpProjectID, *gcpBucketMedia, *gcpBucketTmp)
	if bucketHandler == nil {
		logrus.Errorf("Could not create bucket handler.")
		return fmt.Errorf("could not create bucket handler")
	}

	// create storage handler
	storageHandler := storagehandler.NewStorageHandler(reqHandler, bucketHandler, *gcpBucketMedia)

	// create listen handler
	listenHandler := listenhandler.NewListenHandler(rabbitSock, storageHandler)

	// run
	if err := listenHandler.Run(string(commonoutline.QueueNameStorageRequest), string(commonoutline.QueueNameDelay)); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}
