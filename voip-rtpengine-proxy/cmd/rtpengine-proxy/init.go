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
	defaultInterfaceName = "eth0"

	defaultRtpengineNGAddress = "127.0.0.1:22222"
	defaultRtpengineNGTimeout = "5s"

	defaultRabbitMQAddress     = "amqp://guest:guest@localhost:5672"
	defaultRabbitMQQueueListen = "rtpengine.proxy.request"

	defaultRedisAddress  = "localhost:6379"
	defaultRedisPassword = ""
	defaultRedisDB       = 1

	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
)

func init() {
	initLog()
	initVariable()
	initSignal()
	initProm(prometheusEndpoint, prometheusListenAddress)
	logrus.Info("Init finished.")
}

func initVariable() {
	log := logrus.WithField("func", "initVariable")
	viper.AutomaticEnv()

	pflag.String("interface_name", defaultInterfaceName, "Network interface name for proxy ID detection")
	pflag.String("rtpengine_ng_address", defaultRtpengineNGAddress, "RTPEngine NG UDP address")
	pflag.String("rtpengine_ng_timeout", defaultRtpengineNGTimeout, "Timeout for RTPEngine NG responses")
	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "RabbitMQ address")
	pflag.String("rabbitmq_queue_listen", defaultRabbitMQQueueListen, "Permanent listen queue")
	pflag.String("redis_address", defaultRedisAddress, "Redis address")
	pflag.String("redis_password", defaultRedisPassword, "Redis password")
	pflag.Int("redis_database", defaultRedisDB, "Redis database index")
	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "Prometheus metrics path")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Prometheus listen address")

	pflag.Parse()

	bindVar := func(name, env string, target *string) {
		if err := viper.BindPFlag(name, pflag.Lookup(name)); err != nil {
			log.Errorf("Error binding flag %s: %v", name, err)
			panic(err)
		}
		if err := viper.BindEnv(name, env); err != nil {
			log.Errorf("Error binding env %s: %v", env, err)
			panic(err)
		}
		*target = viper.GetString(name)
	}

	bindVar("interface_name", "INTERFACE_NAME", &interfaceName)
	bindVar("rtpengine_ng_address", "RTPENGINE_NG_ADDRESS", &rtpengineNGAddress)
	bindVar("rtpengine_ng_timeout", "RTPENGINE_NG_TIMEOUT", &rtpengineNGTimeout)
	bindVar("rabbitmq_address", "RABBITMQ_ADDRESS", &rabbitMQAddress)
	bindVar("rabbitmq_queue_listen", "RABBITMQ_QUEUE_LISTEN", &rabbitMQQueueListen)
	bindVar("redis_address", "REDIS_ADDRESS", &redisAddress)
	bindVar("redis_password", "REDIS_PASSWORD", &redisPassword)
	bindVar("prometheus_endpoint", "PROMETHEUS_ENDPOINT", &prometheusEndpoint)
	bindVar("prometheus_listen_address", "PROMETHEUS_LISTEN_ADDRESS", &prometheusListenAddress)

	if err := viper.BindPFlag("redis_database", pflag.Lookup("redis_database")); err != nil {
		panic(err)
	}
	if err := viper.BindEnv("redis_database", "REDIS_DATABASE"); err != nil {
		panic(err)
	}
	redisDatabase = viper.GetInt("redis_database")
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
}

func initProm(endpoint, listen string) {
	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			if err := http.ListenAndServe(listen, nil); err != nil {
				logrus.Errorf("Could not start prometheus listener")
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}()
}
