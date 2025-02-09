package main

import (
	"net/http"
	"os/signal"
	"syscall"
	"time"

	joonix "github.com/joonix/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
	defaultGCPCredentialBase64     = ""
	defaultGCPProjectID            = ""
	defaultGCPBucketName           = ""
	defaultAWSAccessKey            = ""
	defaultAWSSecretKey            = ""
)

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
func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
	pflag.String("gcp_credential_base64", defaultGCPCredentialBase64, "Base64 encoded GCP credential.")
	pflag.String("gcp_project_id", defaultGCPProjectID, "GCP project id.")
	pflag.String("gcp_bucket_name", defaultGCPBucketName, "GCP bucket name for tmp storage.")
	pflag.String("aws_access_key", defaultAWSAccessKey, "AWS access key.")
	pflag.String("aws_secret_key", defaultAWSSecretKey, "AWS secret key.")
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

	// aws_access_key
	if errFlag := viper.BindPFlag("aws_access_key", pflag.Lookup("aws_access_key")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("aws_access_key", "AWS_ACCESS_KEY"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	awsAccessKey = viper.GetString("aws_access_key")

	// aws_secret_key
	if errFlag := viper.BindPFlag("aws_secret_key", pflag.Lookup("aws_secret_key")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("aws_secret_key", "AWS_SECRET_KEY"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	awsSecretKey = viper.GetString("aws_secret_key")

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
