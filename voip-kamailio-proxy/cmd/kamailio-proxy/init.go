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
	defaultRabbitMQAddress     = "amqp://guest:guest@localhost:5672"
	defaultRabbitMQQueueListen = "voip.kamailio.request"

	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"

	defaultSIPTimeout = "5s"
)

// init runs before main(), initializing logging, config, signals, and prometheus.
func init() {
	initLog()
	initVariable()
	initSignal()
	initProm(prometheusEndpoint, prometheusListenAddress)
	logrus.Info("Init finished.")
}

func initVariable() {
	viper.AutomaticEnv()

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server")
	pflag.String("rabbitmq_queue_listen", defaultRabbitMQQueueListen, "Queue name for listening to health check requests")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on")
	pflag.String("sip_timeout", defaultSIPTimeout, "Timeout for SIP OPTIONS health check (e.g. 5s, 3s)")

	pflag.Parse()

	if errFlag := viper.BindPFlag("rabbitmq_address", pflag.Lookup("rabbitmq_address")); errFlag != nil { //nolint
		logrus.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_address", "RABBITMQ_ADDRESS"); errEnv != nil { //nolint
		logrus.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQAddress = viper.GetString("rabbitmq_address")

	if errFlag := viper.BindPFlag("rabbitmq_queue_listen", pflag.Lookup("rabbitmq_queue_listen")); errFlag != nil { //nolint
		logrus.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_queue_listen", "RABBITMQ_QUEUE_LISTEN"); errEnv != nil { //nolint
		logrus.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQQueueListen = viper.GetString("rabbitmq_queue_listen")

	if errFlag := viper.BindPFlag("prometheus_endpoint", pflag.Lookup("prometheus_endpoint")); errFlag != nil { //nolint
		logrus.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_endpoint", "PROMETHEUS_ENDPOINT"); errEnv != nil { //nolint
		logrus.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusEndpoint = viper.GetString("prometheus_endpoint")

	if errFlag := viper.BindPFlag("prometheus_listen_address", pflag.Lookup("prometheus_listen_address")); errFlag != nil { //nolint
		logrus.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS"); errEnv != nil { //nolint
		logrus.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	prometheusListenAddress = viper.GetString("prometheus_listen_address")

	if errFlag := viper.BindPFlag("sip_timeout", pflag.Lookup("sip_timeout")); errFlag != nil { //nolint
		logrus.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("sip_timeout", "SIP_TIMEOUT"); errEnv != nil { //nolint
		logrus.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	sipTimeoutStr := viper.GetString("sip_timeout")
	var parseErr error
	sipTimeout, parseErr = time.ParseDuration(sipTimeoutStr)
	if parseErr != nil {
		logrus.Warnf("Could not parse sip_timeout %q, using default 5s: %v", sipTimeoutStr, parseErr)
		sipTimeout = 5 * time.Second
	}
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM)
}

func initProm(endpoint, listenAddress string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(listenAddress, nil); err != nil { //nolint
			logrus.Errorf("Could not start prometheus server. err: %v", err)
		}
	}()
}
