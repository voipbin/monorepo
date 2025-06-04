package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	joonix "github.com/joonix/log"
)

const (
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"
)

func init() {
	initVariable()
	initLog()
	initProm(prometheusEndpoint, prometheusListenAddress)
	logrus.Info("The init finished.")
}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")
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
}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
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
