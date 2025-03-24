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
	defaultARIAddress      = "localhost:8088"
	defaultARIAccount      = "asterisk:asterisk"
	defaultARISbuscribeAll = "true"
	defaultARIApplication  = "voipbin"

	defaultAMIHost        = "127.0.0.1"
	defaultAMIPort        = "5038"
	defaultAMIUsername    = "asterisk"
	defaultAMIPassword    = "asterisk"
	defaultAMIEventFilter = ""

	defaultInterfaceName = "eth0"

	defaultRabbitMQAddress            = "amqp://guest:guest@localhost:5672"
	defaultRabbitMQQueueListenRequest = "asterisk.call.request"

	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"

	defaultRedisAddress  = "localhost:6379"
	defaultRedisPassword = ""
	defaultRedisDB       = 1
)

// proces init
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

	pflag.String("ari_address", defaultARIAddress, "Address of the ARI server (e.g., localhost:8088)")
	pflag.String("ari_account", defaultARIAccount, "Account for the ARI server (e.g., asterisk:asterisk)")
	pflag.String("ari_subscribe_all", defaultARISbuscribeAll, "Subscribe to all ARI events (e.g., true)")
	pflag.String("ari_application", defaultARIApplication, "Application name for the ARI (e.g., voipbin)")

	pflag.String("ami_host", defaultAMIHost, "Host of the AMI server (e.g., 127.0.0.1)")
	pflag.String("ami_port", defaultAMIPort, "Port of the AMI server (e.g., 5038)")
	pflag.String("ami_username", defaultAMIUsername, "Username for the AMI server (e.g., asterisk)")
	pflag.String("ami_password", defaultAMIPassword, "Password for the AMI server (e.g., asterisk)")
	pflag.String("ami_event_filter", defaultAMIEventFilter, "Event filter for the AMI server")

	pflag.String("interface_name", defaultInterfaceName, "Network interface name (e.g., eth0)")

	pflag.String("rabbitmq_address", defaultRabbitMQAddress, "Address of the RabbitMQ server (e.g., amqp://guest:guest@localhost:5672)")
	pflag.String("rabbitmq_queue_listen", defaultRabbitMQQueueListenRequest, "Queue name for listening to RabbitMQ requests (e.g., asterisk.call.request)")

	pflag.String("prometheus_endpoint", defaultPrometheusEndpoint, "URL for the Prometheus metrics endpoint")
	pflag.String("prometheus_listen_address", defaultPrometheusListenAddress, "Address for Prometheus to listen on (e.g., localhost:8080)")

	pflag.String("redis_address", defaultRedisAddress, "Address of the Redis server (e.g., localhost:6379)")
	pflag.String("redis_password", defaultRedisPassword, "Password of the Redis server")
	pflag.Int("redis_database", defaultRedisDB, "Redis database index to use (default is 1)")

	pflag.Parse()

	// ari_address
	if errFlag := viper.BindPFlag("ari_address", pflag.Lookup("ari_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ari_address", "ARI_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	ariAddress = viper.GetString("ari_address")

	// ari_account
	if errFlag := viper.BindPFlag("ari_account", pflag.Lookup("ari_account")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ari_account", "ARI_ACCOUNT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	ariAccount = viper.GetString("ari_account")

	// ari_subscribe_all
	if errFlag := viper.BindPFlag("ari_subscribe_all", pflag.Lookup("ari_subscribe_all")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ari_subscribe_all", "ARI_SUBSCRIBE_ALL"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	ariSubscribeAll = viper.GetString("ari_subscribe_all")

	// ari_application
	if errFlag := viper.BindPFlag("ari_application", pflag.Lookup("ari_application")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ari_application", "ARI_APPLICATION"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	ariApplication = viper.GetString("ari_application")

	// ami_host
	if errFlag := viper.BindPFlag("ami_host", pflag.Lookup("ami_host")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ami_host", "AMI_HOST"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	amiHost = viper.GetString("ami_host")

	// ami_port
	if errFlag := viper.BindPFlag("ami_port", pflag.Lookup("ami_port")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ami_port", "AMI_PORT"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	amiPort = viper.GetString("ami_port")

	// ami_username
	if errFlag := viper.BindPFlag("ami_username", pflag.Lookup("ami_username")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ami_username", "AMI_USERNAME"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	amiUsername = viper.GetString("ami_username")

	// ami_password
	if errFlag := viper.BindPFlag("ami_password", pflag.Lookup("ami_password")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ami_password", "AMI_PASSWORD"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	amiPassword = viper.GetString("ami_password")

	// ami_event_filter
	if errFlag := viper.BindPFlag("ami_event_filter", pflag.Lookup("ami_event_filter")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("ami_event_filter", "AMI_EVENT_FILTER"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	amiEventFilter = viper.GetString("ami_event_filter")

	// interface_name
	if errFlag := viper.BindPFlag("interface_name", pflag.Lookup("interface_name")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("interface_name", "INTERFACE_NAME"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	interfaceName = viper.GetString("interface_name")

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

	// rabbitmq_queue_listen
	if errFlag := viper.BindPFlag("rabbitmq_queue_listen", pflag.Lookup("rabbitmq_queue_listen")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("rabbitmq_queue_listen", "RABBITMQ_QUEUE_LISTEN"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	rabbitMQQueueListen = viper.GetString("rabbitmq_queue_listen")

	// redis_address
	if errFlag := viper.BindPFlag("redis_address", pflag.Lookup("redis_address")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_address", "REDIS_ADDRESS"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisAddress = viper.GetString("redis_address")

	// redis_password
	if errFlag := viper.BindPFlag("redis_password", pflag.Lookup("redis_password")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_password", "REDIS_PASSWORD"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisPassword = viper.GetString("redis_password")

	// redis_database
	if errFlag := viper.BindPFlag("redis_database", pflag.Lookup("redis_database")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("redis_database", "REDIS_DATABASE"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	redisDatabase = viper.GetInt("redis_database")

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

	// recording_asterisk_directory
	if errFlag := viper.BindPFlag("recording_asterisk_directory", pflag.Lookup("recording_asterisk_directory")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("recording_asterisk_directory", "RECORDING_ASTERISK_DIRECTORY"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	recordingAsteriskDirectory = viper.GetString("recording_asterisk_directory")

	// recording_bucket_directory
	if errFlag := viper.BindPFlag("recording_bucket_directory", pflag.Lookup("recording_bucket_directory")); errFlag != nil {
		log.Errorf("Error binding flag: %v", errFlag)
		panic(errFlag)
	}
	if errEnv := viper.BindEnv("recording_bucket_directory", "RECORDING_BUCKET_DIRECTORY"); errEnv != nil {
		log.Errorf("Error binding env: %v", errEnv)
		panic(errEnv)
	}
	recordingBucketDirectory = viper.GetString("recording_bucket_directory")

}

// initLog inits log settings.
func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initSignal inits sinal settings.
func initSignal() {
	signal.Notify(chSigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
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
