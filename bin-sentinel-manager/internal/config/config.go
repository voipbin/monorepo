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

	// defaultSentinelBackend is deliberately EMPTY -- there is no default backend.
	//
	// Auto-detecting "am I in a Kubernetes pod or a Docker container" is exactly the kind of
	// implicit, hard-to-audit behavior this service has spent many review rounds removing
	// elsewhere, and defaulting to either one would silently give a misconfigured deployment the
	// wrong watcher rather than a clear error. An operator has to say which one they mean
	// (design §8.3).
	defaultSentinelBackend = ""
)

// list of the supported monitoring backends, selected by SENTINEL_BACKEND.
const (
	// BackendKubernetes watches pods through the Kubernetes API. Used by self-hosted Kubernetes
	// deployments via k8s/deployment.yml.
	BackendKubernetes = "kubernetes"
	// BackendDocker watches containers through the Docker Events API. Used by the bm-nyc-01
	// deployment via komodo/docker-compose.yml.
	BackendDocker = "docker"
)

// Config holds all configuration for the sentinel-manager service
type Config struct {
	PrometheusEndpoint      string
	PrometheusListenAddress string
	RabbitMQAddress         string

	// SentinelBackend selects the monitoring backend: BackendKubernetes or BackendDocker.
	// Required, with no default -- see Validate.
	SentinelBackend string

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
	f.String("sentinel_backend", defaultSentinelBackend, "Monitoring backend to run: kubernetes | docker (required)")

	bindings := map[string]string{
		"rabbitmq_address":            "RABBITMQ_ADDRESS",
		"prometheus_endpoint":         "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address":   "PROMETHEUS_LISTEN_ADDRESS",
		"docker_socket_proxy_address": "DOCKER_SOCKET_PROXY_ADDRESS",
		"redis_address":               "REDIS_ADDRESS",
		"redis_password":              "REDIS_PASSWORD",
		"redis_database":              "REDIS_DATABASE",
		"sentinel_backend":            "SENTINEL_BACKEND",
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

// LoadGlobalConfig loads configuration from viper into the global singleton.
//
// It returns an error so that cmd/sentinel-control's PersistentPreRunE can actually PROPAGATE a
// validation failure instead of discarding it. The two binaries bootstrap config through
// different entry points -- sentinel-manager via InitConfig, sentinel-control via
// Bootstrap + LoadGlobalConfig -- so validation does not wire itself into both just because they
// share this package. The signature change is what makes the CLI fail the same way the service
// does on a missing or invalid SENTINEL_BACKEND, rather than silently running against
// unvalidated config.
func LoadGlobalConfig() error {
	var err error

	once.Do(func() {
		loaded := loadFromViper()
		if errValidate := loaded.Validate(); errValidate != nil {
			err = errValidate
			return
		}
		cfg = loaded
	})

	return err
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
		"sentinel_backend",
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

	loaded := loadFromViper()
	if err := loaded.Validate(); err != nil {
		return err
	}
	cfg = loaded

	return nil
}

// Validate rejects a configuration the selected backend cannot actually run on.
//
// Validation is BACKEND-CONDITIONAL by design (§8.3): the Docker-only fields are required when
// running the Docker backend and irrelevant when running the Kubernetes one. Validating all of
// them unconditionally would fail a perfectly good Kubernetes deployment on Docker config it was
// never given and has no reason to provide. Symmetrically there is nothing extra to require for
// Kubernetes -- rest.InClusterConfig() reads the in-cluster service-account mount, which it
// either finds or does not.
func (c Config) Validate() error {
	switch c.SentinelBackend {
	case BackendKubernetes:
		return nil

	case BackendDocker:
		if c.DockerSocketProxyAddress == "" {
			return errors.Errorf("docker_socket_proxy_address is required when sentinel_backend is %q", BackendDocker)
		}
		if c.RedisAddress == "" {
			return errors.Errorf("redis_address is required when sentinel_backend is %q", BackendDocker)
		}
		return nil

	case "":
		return errors.Errorf("sentinel_backend is required and has no default. set SENTINEL_BACKEND to %q or %q", BackendKubernetes, BackendDocker)

	default:
		return errors.Errorf("invalid sentinel_backend %q. must be %q or %q", c.SentinelBackend, BackendKubernetes, BackendDocker)
	}
}

// loadFromViper materializes the Config struct out of viper's resolved values.
func loadFromViper() Config {
	return Config{
		PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
		PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
		RabbitMQAddress:         viper.GetString("rabbitmq_address"),

		SentinelBackend: viper.GetString("sentinel_backend"),

		DockerSocketProxyAddress: viper.GetString("docker_socket_proxy_address"),

		RedisAddress:  viper.GetString("redis_address"),
		RedisPassword: viper.GetString("redis_password"),
		RedisDatabase: viper.GetInt("redis_database"),
	}
}
