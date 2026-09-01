package config

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg  Config
	once sync.Once
)

// defaults of the configuration values.
const (
	defaultPrometheusEndpoint      = "/metrics"
	defaultPrometheusListenAddress = ":2112"
	defaultRabbitMQAddress         = "amqp://guest:guest@localhost:5672"

	// defaultDockerSocketProxyAddress points at the read-only docker-socket-proxy SIDECAR
	// declared in this service's own komodo/docker-compose.yml, NOT at /var/run/docker.sock.
	//
	// The raw socket grants root-equivalent host access, so it is never mounted into this
	// service. The proxy sits on a dedicated `internal: true` network joined only by these two
	// containers -- matching the shape infra-prometheus (VOIP-1402) and infra-loki (VOIP-1423)
	// already use in monorepo-etc, and specifically NOT the shared `production` network, where
	// every other container could reach the proxy and read any container's env vars through
	// `docker inspect`.
	defaultDockerSocketProxyAddress = "tcp://sentinel-docker-socket-proxy:2375"

	defaultRedisAddress  = "localhost:6379"
	defaultRedisPassword = ""
	defaultRedisDatabase = 1
)

// Config holds all configuration for the sentinel-manager service
type Config struct {
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string

	// DockerSocketProxyAddress is the Docker host URL of the read-only socket proxy.
	DockerSocketProxyAddress string

	// Redis holds the `asterisk.<id>.address-internal` keys voip-asterisk-proxy publishes. This
	// service only ever READS them.
	RedisAddress  string
	RedisPassword string
	RedisDatabase int
}

// Get returns the current configuration
func Get() Config {
	return cfg
}

// Bootstrap sets up configuration flags and bindings for CLI tools
func Bootstrap(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", defaultRabbitMQAddress, "RabbitMQ server address")
	f.String("prometheus_endpoint", defaultPrometheusEndpoint, "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", defaultPrometheusListenAddress, "Prometheus listen address")
	f.String("docker_socket_proxy_address", defaultDockerSocketProxyAddress, "Read-only docker-socket-proxy host address")
	f.String("redis_address", defaultRedisAddress, "Redis server address")
	f.String("redis_password", defaultRedisPassword, "Redis server password")
	f.Int("redis_database", defaultRedisDatabase, "Redis database index")

	bindings := map[string]string{
		"rabbitmq_address":            "RABBITMQ_ADDRESS",
		"prometheus_endpoint":         "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address":   "PROMETHEUS_LISTEN_ADDRESS",
		"docker_socket_proxy_address": "DOCKER_SOCKET_PROXY_ADDRESS",
		"redis_address":               "REDIS_ADDRESS",
		"redis_password":              "REDIS_PASSWORD",
		"redis_database":              "REDIS_DATABASE",
	}

	for flagKey, envKey := range bindings {
		if errBind := viper.BindPFlag(flagKey, f.Lookup(flagKey)); errBind != nil {
			return errors.Wrapf(errBind, "could not bind flag. key: %s", flagKey)
		}

		if errBind := viper.BindEnv(flagKey, envKey); errBind != nil {
			return errors.Wrapf(errBind, "could not bind the env. key: %s", envKey)
		}
	}

	return nil
}

// LoadGlobalConfig loads configuration from viper into the global singleton
func LoadGlobalConfig() {
	once.Do(func() {
		cfg = loadFromViper()
	})
}

// InitConfig initializes the configuration with Cobra command
func InitConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()

	flagKeys := []string{
		"prometheus_endpoint",
		"prometheus_listen_address",
		"rabbitmq_address",
		"docker_socket_proxy_address",
		"redis_address",
		"redis_password",
		"redis_database",
	}

	for _, flagKey := range flagKeys {
		flag := cmd.Flags().Lookup(flagKey)
		if flag == nil {
			return fmt.Errorf("error binding %s flag: flag not defined", flagKey)
		}

		if err := viper.BindPFlag(flagKey, flag); err != nil {
			return fmt.Errorf("error binding %s flag: %w", flagKey, err)
		}
	}

	cfg = loadFromViper()

	return nil
}

// loadFromViper materializes the Config struct out of viper's resolved values.
func loadFromViper() Config {
	return Config{
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),

		DockerSocketProxyAddress: viper.GetString("docker_socket_proxy_address"),

		RedisAddress:  viper.GetString("redis_address"),
		RedisPassword: viper.GetString("redis_password"),
		RedisDatabase: viper.GetInt("redis_database"),
	}
}
