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
	PublicBaseURL           string // PublicBaseURL is the public base URL of this API (e.g. "https://api.voipbin.net"), used to build absolute URLs handed to external clients (extension provisioning). Env var only (API_PUBLIC_BASE_URL) -- see the NOTE below about inert CLI flags.

	// Rate limiting (per-IP, in-memory token bucket -- see lib/middleware/ratelimit.go).
	// Each pair is (requests/second, burst). A tier is disabled (unlimited
	// pass-through) if its RPS is not a positive number or its burst <= 0 --
	// this is the safe way to turn a tier off at runtime, since feeding 0
	// straight into the underlying limiter would instead deny every request.
	// NOTE: these are read from environment variables only. CLI flags for
	// these fields (and all other fields in this struct) are inert, because
	// LoadGlobalConfig() runs before cobra parses argv -- see Bootstrap/PostBootstrap
	// call order in cmd/api-manager/main.go.
	RateLimitAuthPublicRPS      float64 // RateLimitAuthPublicRPS is the per-IP request rate for the unauthenticated /auth/* routes (login, signup, password reset, etc.).
	RateLimitAuthPublicBurst    int     // RateLimitAuthPublicBurst is the burst size for RateLimitAuthPublicRPS.
	RateLimitAuthProtectedRPS   float64 // RateLimitAuthProtectedRPS is the per-IP request rate for the authenticated /auth/unregister and /auth/delegate routes.
	RateLimitAuthProtectedBurst int     // RateLimitAuthProtectedBurst is the burst size for RateLimitAuthProtectedRPS.
	RateLimitV1RPS              float64 // RateLimitV1RPS is the per-IP request rate for the full authenticated v1.0 API surface.
	RateLimitV1Burst            int     // RateLimitV1Burst is the burst size for RateLimitV1RPS.

	RateLimitProvisioningPublicRPS   float64 // RateLimitProvisioningPublicRPS is the per-IP request rate for the unauthenticated /provisioning/* routes (extension QR provisioning).
	RateLimitProvisioningPublicBurst int     // RateLimitProvisioningPublicBurst is the burst size for RateLimitProvisioningPublicRPS.

	// Customer-scoped rate limiting (Redis-backed, cross-pod global -- see
	// lib/middleware/customer_ratelimit.go and VOIP-1302 design doc §4-9).
	// Applies only to the v1.0 route group, keyed by customer_id rather
	// than IP. As with the IP tiers above, a tier is disabled (unlimited
	// pass-through) if its RPS is not positive or its burst <= 0.
	RateLimitCustomerV1RPS           float64 // RateLimitCustomerV1RPS is the per-customer request rate shared by agent and accesskey identities (tier v1_customer).
	RateLimitCustomerV1Burst         int     // RateLimitCustomerV1Burst is the burst size for RateLimitCustomerV1RPS.
	RateLimitCustomerV1DirectRPS     float64 // RateLimitCustomerV1DirectRPS is the per-customer request rate for direct (resource-scoped) identities (tier v1_customer_direct).
	RateLimitCustomerV1DirectBurst   int     // RateLimitCustomerV1DirectBurst is the burst size for RateLimitCustomerV1DirectRPS.
	RateLimitCustomerV1DelegateRPS   float64 // RateLimitCustomerV1DelegateRPS is the per-customer request rate for delegate identities (tier v1_customer_delegate).
	RateLimitCustomerV1DelegateBurst int     // RateLimitCustomerV1DelegateBurst is the burst size for RateLimitCustomerV1DelegateRPS.
	RateLimitCustomerRedisTimeoutMs  int     // RateLimitCustomerRedisTimeoutMs is the timeout budget (in milliseconds) for the Redis round trip; on timeout the request fails open.
}

func Bootstrap(cmd *cobra.Command) error {
	initLog()
	if errBind := bindConfig(cmd); errBind != nil {
		return errors.Wrapf(errBind, "could not bind config")
	}

	return nil
}

func PostBootstrap() error {
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

	f.String("rabbitmq_address", "amqp://guest:guest@localhost:5672", "RabbitMQ server address")
	f.String("prometheus_endpoint", "/metrics", "Prometheus metrics endpoint")
	f.String("prometheus_listen_address", ":2112", "Prometheus listen address")
	f.String("database_dsn", "testid:testpassword@tcp(127.0.0.1:3306)/test", "Database connection DSN")
	f.String("redis_address", "127.0.0.1:6379", "Redis server address")
	f.String("redis_password", "", "Redis password")
	f.Int("redis_database", 1, "Redis database index")
	f.String("jwt_key", "", "JWT secret key for token signing")
	f.String("gcp_project_id", "", "GCP project ID")
	f.String("gcp_bucket_name", "", "GCP bucket name for temporary storage")
	f.String("ssl_cert_base64", "", "Base64 encoded SSL certificate")
	f.String("ssl_privkey_base64", "", "Base64 encoded SSL private key")
	f.String("listen_ip_audiosock", "", "Listen IP address for audiosocket connection")
	f.String("public_base_url", "https://api.voipbin.net", "Public base URL of this API, used to build absolute URLs handed to external clients. Env var only, see API_PUBLIC_BASE_URL.")
	f.Float64("rate_limit_auth_public_rps", 10, "Rate limit (requests/second per IP) for unauthenticated /auth/* routes. <=0 disables this tier. Env var only, see RATE_LIMIT_AUTH_PUBLIC_RPS.")
	f.Int("rate_limit_auth_public_burst", 20, "Rate limit burst size for the unauthenticated /auth/* routes. <=0 disables this tier. Env var only, see RATE_LIMIT_AUTH_PUBLIC_BURST.")
	f.Float64("rate_limit_auth_protected_rps", 10, "Rate limit (requests/second per IP) for /auth/unregister and /auth/delegate. <=0 disables this tier. Env var only, see RATE_LIMIT_AUTH_PROTECTED_RPS.")
	f.Int("rate_limit_auth_protected_burst", 20, "Rate limit burst size for /auth/unregister and /auth/delegate. <=0 disables this tier. Env var only, see RATE_LIMIT_AUTH_PROTECTED_BURST.")
	f.Float64("rate_limit_v1_rps", 200, "Rate limit (requests/second per IP) for the authenticated v1.0 API surface. <=0 disables this tier. Env var only, see RATE_LIMIT_V1_RPS.")
	f.Int("rate_limit_v1_burst", 400, "Rate limit burst size for the authenticated v1.0 API surface. <=0 disables this tier. Env var only, see RATE_LIMIT_V1_BURST.")
	f.Float64("rate_limit_provisioning_public_rps", 5, "Rate limit (requests/second per IP) for unauthenticated /provisioning/* routes. <=0 disables this tier. Env var only, see RATE_LIMIT_PROVISIONING_PUBLIC_RPS.")
	f.Int("rate_limit_provisioning_public_burst", 10, "Rate limit burst size for the unauthenticated /provisioning/* routes. <=0 disables this tier. Env var only, see RATE_LIMIT_PROVISIONING_PUBLIC_BURST.")
	f.Float64("rate_limit_customer_v1_rps", 16.7, "Redis-backed per-customer request rate (agent+accesskey, tier v1_customer). <=0 disables this tier. Env var only, see RATE_LIMIT_CUSTOMER_V1_RPS.")
	f.Int("rate_limit_customer_v1_burst", 33, "Burst size for RATE_LIMIT_CUSTOMER_V1_RPS. <=0 disables this tier.")
	f.Float64("rate_limit_customer_v1_direct_rps", 50, "Redis-backed per-customer request rate for direct identities (tier v1_customer_direct). <=0 disables this tier. Env var only, see RATE_LIMIT_CUSTOMER_V1_DIRECT_RPS.")
	f.Int("rate_limit_customer_v1_direct_burst", 100, "Burst size for RATE_LIMIT_CUSTOMER_V1_DIRECT_RPS. <=0 disables this tier.")
	f.Float64("rate_limit_customer_v1_delegate_rps", 8.3, "Redis-backed per-customer request rate for delegate identities (tier v1_customer_delegate). <=0 disables this tier. Env var only, see RATE_LIMIT_CUSTOMER_V1_DELEGATE_RPS.")
	f.Int("rate_limit_customer_v1_delegate_burst", 16, "Burst size for RATE_LIMIT_CUSTOMER_V1_DELEGATE_RPS. <=0 disables this tier.")
	f.Int("rate_limit_customer_redis_timeout_ms", 50, "Timeout budget in milliseconds for the customer rate limiter's Redis round trip; on timeout the request fails open. Env var only, see RATE_LIMIT_CUSTOMER_REDIS_TIMEOUT_MS.")

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
		"public_base_url":           "API_PUBLIC_BASE_URL",

		"rate_limit_auth_public_rps":           "RATE_LIMIT_AUTH_PUBLIC_RPS",
		"rate_limit_auth_public_burst":         "RATE_LIMIT_AUTH_PUBLIC_BURST",
		"rate_limit_auth_protected_rps":        "RATE_LIMIT_AUTH_PROTECTED_RPS",
		"rate_limit_auth_protected_burst":      "RATE_LIMIT_AUTH_PROTECTED_BURST",
		"rate_limit_v1_rps":                    "RATE_LIMIT_V1_RPS",
		"rate_limit_v1_burst":                  "RATE_LIMIT_V1_BURST",
		"rate_limit_provisioning_public_rps":   "RATE_LIMIT_PROVISIONING_PUBLIC_RPS",
		"rate_limit_provisioning_public_burst": "RATE_LIMIT_PROVISIONING_PUBLIC_BURST",

		"rate_limit_customer_v1_rps":            "RATE_LIMIT_CUSTOMER_V1_RPS",
		"rate_limit_customer_v1_burst":          "RATE_LIMIT_CUSTOMER_V1_BURST",
		"rate_limit_customer_v1_direct_rps":     "RATE_LIMIT_CUSTOMER_V1_DIRECT_RPS",
		"rate_limit_customer_v1_direct_burst":   "RATE_LIMIT_CUSTOMER_V1_DIRECT_BURST",
		"rate_limit_customer_v1_delegate_rps":   "RATE_LIMIT_CUSTOMER_V1_DELEGATE_RPS",
		"rate_limit_customer_v1_delegate_burst": "RATE_LIMIT_CUSTOMER_V1_DELEGATE_BURST",
		"rate_limit_customer_redis_timeout_ms":  "RATE_LIMIT_CUSTOMER_REDIS_TIMEOUT_MS",
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
			PublicBaseURL:           viper.GetString("public_base_url"),

			RateLimitAuthPublicRPS:      viper.GetFloat64("rate_limit_auth_public_rps"),
			RateLimitAuthPublicBurst:    viper.GetInt("rate_limit_auth_public_burst"),
			RateLimitAuthProtectedRPS:   viper.GetFloat64("rate_limit_auth_protected_rps"),
			RateLimitAuthProtectedBurst: viper.GetInt("rate_limit_auth_protected_burst"),
			RateLimitV1RPS:              viper.GetFloat64("rate_limit_v1_rps"),
			RateLimitV1Burst:            viper.GetInt("rate_limit_v1_burst"),

			RateLimitProvisioningPublicRPS:   viper.GetFloat64("rate_limit_provisioning_public_rps"),
			RateLimitProvisioningPublicBurst: viper.GetInt("rate_limit_provisioning_public_burst"),

			RateLimitCustomerV1RPS:           viper.GetFloat64("rate_limit_customer_v1_rps"),
			RateLimitCustomerV1Burst:         viper.GetInt("rate_limit_customer_v1_burst"),
			RateLimitCustomerV1DirectRPS:     viper.GetFloat64("rate_limit_customer_v1_direct_rps"),
			RateLimitCustomerV1DirectBurst:   viper.GetInt("rate_limit_customer_v1_direct_burst"),
			RateLimitCustomerV1DelegateRPS:   viper.GetFloat64("rate_limit_customer_v1_delegate_rps"),
			RateLimitCustomerV1DelegateBurst: viper.GetInt("rate_limit_customer_v1_delegate_burst"),
			RateLimitCustomerRedisTimeoutMs:  viper.GetInt("rate_limit_customer_redis_timeout_ms"),
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
