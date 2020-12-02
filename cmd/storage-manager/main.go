package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/listenhandler"
)

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

// args for rabbitmq
var rabbitAddr = flag.String("rabbit_addr", "amqp://guest:guest@localhost:5672", "rabbitmq service address.")
var rabbitQueueListen = flag.String("rabbit_queue_listen", "bin-manager.storage-manager.request", "rabbitmq queue name for request listen")
var rabbitQueueNotify = flag.String("rabbit_queue_notify", "bin-manager.storage-manager.event", "rabbitmq queue name for event notify")
var rabbitExchangeDelay = flag.String("rabbit_exchange_delay", "bin-manager.delay", "rabbitmq exchange name for delayed messaging.")

// args for prometheus
var promEndpoint = flag.String("prom_endpoint", "/metrics", "endpoint for prometheus metric collecting.")
var promListenAddr = flag.String("prom_listen_addr", ":2112", "endpoint for prometheus metric collecting.")

// gcp info
var gcpCredential = flag.String("gcp_credential", "./credential.json", "the GCP credential file path")
var gcpProjectID = flag.String("gcp_project_id", "project", "the gcp project id")
var gcpBucketName = flag.String("gcp_bucket_name", "bucket", "the gcp bucket name to use")

func main() {

	printIPAddresses()

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
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
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

	if err := runHTTPListen(); err != nil {
		return err
	}

	return nil
}

// runListen run the listener
func runListen() error {
	// rabbitmq sock connect
	rabbitSock := rabbitmqhandler.NewRabbit(*rabbitAddr)
	rabbitSock.Connect()

	// create bucket handler
	bucketHandler := buckethandler.NewBucketHandler(*gcpCredential, *gcpProjectID, *gcpBucketName)
	if bucketHandler == nil {
		logrus.Errorf("Could not create bucket handler.")
		return fmt.Errorf("could not create bucket handler")
	}

	// create listen handler
	listenHandler := listenhandler.NewListenHandler(rabbitSock, bucketHandler)

	// run
	if err := listenHandler.Run(*rabbitQueueListen, *rabbitExchangeDelay); err != nil {
		logrus.Errorf("Could not run the listenhandler correctly. err: %v", err)
	}

	return nil
}

func runHTTPListen() error {
	// create bucket handler
	bucketHandler := buckethandler.NewBucketHandler(*gcpCredential, *gcpProjectID, *gcpBucketName)
	if bucketHandler == nil {
		logrus.Errorf("Could not create tts handler.")
		return fmt.Errorf("could not create tts handler")

	}

	return nil
}

func printIPAddresses() {
	ifaces, err := net.Interfaces()
	if err != nil {
		logrus.Errorf("Could not get interfaces info. err: %v", err)
		return
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			logrus.Errorf("Could not get interface's ip address. err: %v", err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			fmt.Printf("interface: %s, ip: %v\n", i.Name, ip)
		}
	}

}
