package config

import (
	"encoding/base64"
	"net/http"
	"os"
	"sync"
	"time"

	joonix "github.com/joonix/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	constSSLPrivFilename = "/tmp/ssl_privkey.pem"
	constSSLCertFilename = "/tmp/ssl_cert.pem"
)

var (
	globalConfig Config
	once         sync.Once
)

// Config holds process-wide configuration values loaded from command-line
// flags and environment variables for the service.
type Config struct {
	RabbitMQAddress         string // RabbitMQAddress is the address (including host and port) of the RabbitMQ server.
	PrometheusEndpoint      string // PrometheusEndpoint is the HTTP path at which Prometheus metrics are exposed.
	PrometheusListenAddress string // PrometheusListenAddress is the network address on which the Prometheus metrics HTTP server listens (for example, ":8080").
	DatabaseDSN             string // DatabaseDSN is the data source name used to connect to the primary database.
	RedisAddress            string // RedisAddress is the address (including host and port) of the Redis server.
	RedisPassword           string // RedisPassword is the password used for authenticating to the Redis server.
	RedisDatabase           int    // RedisDatabase is the numeric Redis logical database index to select, not a name.
	JWTKey                  string // JWTKey is the secret key used for JWT token signing and validation.
	GCPProjectID            string // GCPProjectID is the Google Cloud Platform project identifier.
	GCPBucketName           string // GCPBucketName is the name of the GCP storage bucket for temporary storage.
	SSLCertBase64           string // SSLCertBase64 is the base64-encoded SSL certificate for HTTPS connections.
	SSLPrivKeyBase64        string // SSLPrivKeyBase64 is the base64-encoded SSL private key for HTTPS connections.
	ListenIPAudiosock       string // ListenIPAudiosock is the IP address for audiosocket connection listening.
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}

	// Initialize Prometheus monitoring
	cfg := Get()
	initProm(cfg.PrometheusEndpoint, cfg.PrometheusListenAddress)

	// Write SSL certificate files from base64 encoded config
	if errWrite := writeBase64(constSSLCertFilename, cfg.SSLCertBase64); errWrite != nil {
		return errors.Wrapf(errWrite, "could not write SSL cert file")
	}

	if errWrite := writeBase64(constSSLPrivFilename, cfg.SSLPrivKeyBase64); errWrite != nil {
		return errors.Wrapf(errWrite, "could not write SSL private key file")
	}

	return nil
}

// bindConfig binds CLI flags and environment variables for configuration.
// It maps command-line flags to environment variables using Viper.
func bindConfig(cmd *cobra.Command) error {
	viper.AutomaticEnv()
	f := cmd.PersistentFlags()

	f.String("rabbitmq_address", "", "RabbitMQ server address")
	f.String("prometheus_endpoint", "", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", "", "Prometheus listen address")
	f.String("database_dsn", "", "Database connection DSN")
	f.String("redis_address", "", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 0, "Redis database index")
	f.String("jwt_key", "", "JWT secret key for token signing")
	f.String("gcp_project_id", "", "GCP project ID")
	f.String("gcp_bucket_name", "", "GCP bucket name for temporary storage")
	f.String("ssl_cert_base64", "", "Base64 encoded SSL certificate")
	f.String("ssl_privkey_base64", "", "Base64 encoded SSL private key")
	f.String("listen_ip_audiosock", "", "Listen IP address for audiosocket connection")

	bindings := map[string]string{
		"rabbitmq_address":          "RABBITMQ_ADDRESS",
		"prometheus_endpoint":       "PROMETHEUS_ENDPOINT",
		"prometheus_listen_address": "PROMETHEUS_LISTEN_ADDRESS",
		"database_dsn":              "DATABASE_DSN",
		"redis_address":             "REDIS_ADDRESS",
		"redis_password":            "REDIS_PASSWORD",
		"redis_database":            "REDIS_DATABASE",
		"jwt_key":                   "JWT_KEY",
		"gcp_project_id":            "GCP_PROJECT_ID",
		"gcp_bucket_name":           "GCP_BUCKET_NAME",
		"ssl_cert_base64":           "SSL_CERT_BASE64",
		"ssl_privkey_base64":        "SSL_PRIVKEY_BASE64",
		"listen_ip_audiosock":       "POD_IP",
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

func Get() *Config {
	return &globalConfig
}

// LoadGlobalConfig loads configuration from viper into the global singleton.
// NOTE: This must be called AFTER Bootstrap (which calls bindConfig) has been executed.
// If called before binding, it will load empty/default values.
func LoadGlobalConfig() {
	once.Do(func() {
		globalConfig = Config{
			RabbitMQAddress:         viper.GetString("rabbitmq_address"),
			PrometheusEndpoint:      viper.GetString("prometheus_endpoint"),
			PrometheusListenAddress: viper.GetString("prometheus_listen_address"),
			DatabaseDSN:             viper.GetString("database_dsn"),
			RedisAddress:            viper.GetString("redis_address"),
			RedisPassword:           viper.GetString("redis_password"),
			RedisDatabase:           viper.GetInt("redis_database"),
			JWTKey:                  viper.GetString("jwt_key"),
			GCPProjectID:            viper.GetString("gcp_project_id"),
			GCPBucketName:           viper.GetString("gcp_bucket_name"),
			SSLCertBase64:           viper.GetString("ssl_cert_base64"),
			SSLPrivKeyBase64:        viper.GetString("ssl_privkey_base64"),
			ListenIPAudiosock:       viper.GetString("listen_ip_audiosock"),
		}
		logrus.Debug("Configuration has been loaded and locked.")
	})
}

func initLog() {
	logrus.SetFormatter(joonix.NewFormatter())
	logrus.SetLevel(logrus.DebugLevel)
}

// initProm initializes Prometheus settings
func initProm(endpoint, listen string) {
	log := logrus.WithField("func", "initProm").WithFields(logrus.Fields{
		"endpoint": endpoint,
		"listen":   listen,
	})

	http.Handle(endpoint, promhttp.Handler())
	go func() {
		for {
			if errListen := http.ListenAndServe(listen, nil); errListen != nil {
				log.Errorf("Could not start prometheus listener. err: %v", errListen)
				time.Sleep(time.Second * 1)
				continue
			}
			log.Infof("Finishing the prometheus listener.")
			break
		}
	}()
}

// writeBase64 decodes a base64 string and writes it to a file
func writeBase64(filename string, data string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "writeBase64",
		"filename": filename,
	})

	// Skip if data is empty
	if data == "" {
		return nil
	}

	// Create or open the file
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("Could not create a file. err: %v", err)
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	tmp, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Errorf("Error decoding Base64 string: %v", err)
		return err
	}

	// Write the decoded data to the file
	_, err = file.Write(tmp)
	if err != nil {
		log.Errorf("Could not write to file. err: %v", err)
		return err
	}

	return nil
}
