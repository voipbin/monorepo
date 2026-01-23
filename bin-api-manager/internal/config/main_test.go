package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil, expected *Config")
	}
}

func TestBootstrap(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
	}

	err := Bootstrap(cmd)
	if err != nil {
		t.Errorf("Bootstrap() returned error: %v", err)
	}

	flags := cmd.PersistentFlags()

	tests := []struct {
		name     string
		flagName string
	}{
		{"rabbitmq_address", "rabbitmq_address"},
		{"prometheus_endpoint", "prometheus_endpoint"},
		{"prometheus_listen_address", "prometheus_listen_address"},
		{"database_dsn", "database_dsn"},
		{"redis_address", "redis_address"},
		{"redis_password", "redis_password"},
		{"redis_database", "redis_database"},
		{"jwt_key", "jwt_key"},
		{"gcp_project_id", "gcp_project_id"},
		{"gcp_bucket_name", "gcp_bucket_name"},
		{"ssl_cert_base64", "ssl_cert_base64"},
		{"ssl_privkey_base64", "ssl_privkey_base64"},
		{"listen_ip_audiosock", "listen_ip_audiosock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := flags.Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %s was not registered", tt.flagName)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		RabbitMQAddress:         "amqp://localhost:5672",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":8080",
		DatabaseDSN:             "user:pass@tcp(localhost:3306)/db",
		RedisAddress:            "localhost:6379",
		RedisPassword:           "secret",
		RedisDatabase:           1,
		JWTKey:                  "test-jwt-key",
		GCPProjectID:            "test-project",
		GCPBucketName:           "test-bucket",
		SSLCertBase64:           "dGVzdA==",
		SSLPrivKeyBase64:        "dGVzdA==",
		ListenIPAudiosock:       "0.0.0.0",
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"RabbitMQAddress", cfg.RabbitMQAddress, "amqp://localhost:5672"},
		{"PrometheusEndpoint", cfg.PrometheusEndpoint, "/metrics"},
		{"PrometheusListenAddress", cfg.PrometheusListenAddress, ":8080"},
		{"DatabaseDSN", cfg.DatabaseDSN, "user:pass@tcp(localhost:3306)/db"},
		{"RedisAddress", cfg.RedisAddress, "localhost:6379"},
		{"RedisPassword", cfg.RedisPassword, "secret"},
		{"RedisDatabase", cfg.RedisDatabase, 1},
		{"JWTKey", cfg.JWTKey, "test-jwt-key"},
		{"GCPProjectID", cfg.GCPProjectID, "test-project"},
		{"GCPBucketName", cfg.GCPBucketName, "test-bucket"},
		{"SSLCertBase64", cfg.SSLCertBase64, "dGVzdA=="},
		{"SSLPrivKeyBase64", cfg.SSLPrivKeyBase64, "dGVzdA=="},
		{"ListenIPAudiosock", cfg.ListenIPAudiosock, "0.0.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestSSLFilenameConstants(t *testing.T) {
	if constSSLPrivFilename != "/tmp/ssl_privkey.pem" {
		t.Errorf("constSSLPrivFilename = %v, expected /tmp/ssl_privkey.pem", constSSLPrivFilename)
	}
	if constSSLCertFilename != "/tmp/ssl_cert.pem" {
		t.Errorf("constSSLCertFilename = %v, expected /tmp/ssl_cert.pem", constSSLCertFilename)
	}
}
