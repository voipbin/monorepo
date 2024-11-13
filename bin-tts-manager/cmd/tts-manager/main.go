package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/pkg/listenhandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"
)

const serviceName = commonoutline.ServiceNameTTSManager

// channels
var chSigs = make(chan os.Signal, 1)
var chDone = make(chan bool, 1)

const (
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultGCPCredentialBase64     = ""
	defaultGCPProjectID            = ""
	defaultGCPBucketName           = ""
)

var (
	prometheusEndpoint      = ""
	prometheusListenAddress = ""
	rabbitMQAddress         = ""

	gcpCredentialBase64 = ""
	gcpProjectID        = ""
	gcpBucketName       = ""
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

// proces init
func init() {
	initVariable()

	// init logs
	initLog()

	// init signal handler
	initSignal()

	// init prometheus setting
	initProm(prometheusEndpoint, prometheusListenAddress)
}

// signalHandler catches signals and set the done
func signalHandler() {
	sig := <-chSigs
	logrus.Debugf("Received signal. sig: %v", sig)
	chDone <- true
}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	pflag.String("gcp_credential_base64", defaultGCPCredentialBase64, "Base64 encoded GCP credential.")
	pflag.String("gcp_project_id", defaultGCPProjectID, "GCP project id.")
	pflag.String("gcp_bucket_name", defaultGCPBucketName, "GCP bucket name for tmp storage.")
	pflag.Parse()

	// rabbitmq_address
	if errFlag := viper.BindPFlag("rabbitmq_address", pflag.Lookup("rabbitmq_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQAddress = viper.GetString("rabbitmq_address")

	// prometheus_endpoint
	if errFlag := viper.BindPFlag("prometheus_endpoint", pflag.Lookup("prometheus_endpoint")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusEndpoint = viper.GetString("prometheus_endpoint")

	// prometheus_listen_address
	if errFlag := viper.BindPFlag("prometheus_listen_address", pflag.Lookup("prometheus_listen_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusListenAddress = viper.GetString("prometheus_listen_address")

	// gcp_credential_base64
	if errFlag := viper.BindPFlag("gcp_credential_base64", pflag.Lookup("gcp_credential_base64")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("gcp_credential_base64", "GCP_CREDENTIAL_BASE64"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	gcpCredentialBase64 = viper.GetString("gcp_credential_base64")

	// gcp_project_id
	if errFlag := viper.BindPFlag("gcp_project_id", pflag.Lookup("gcp_project_id")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("gcp_project_id", "GCP_PROJECT_ID"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	gcpProjectID = viper.GetString("gcp_project_id")

	// gcp_bucket_name
	if errFlag := viper.BindPFlag("gcp_bucket_name", pflag.Lookup("gcp_bucket_name")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("gcp_bucket_name", "GCP_BUCKET_NAME"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	gcpBucketName = viper.GetString("gcp_bucket_name")

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
	ttsHandler := ttshandler.NewTTSHandler(gcpCredentialBase64, gcpProjectID, gcpBucketName, "/shared-data", localAddress, notifyHandler)
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
