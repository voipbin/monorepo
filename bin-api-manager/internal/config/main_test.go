package config

import (
	"encoding/base64"
	"os"
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

func TestLoadGlobalConfig(t *testing.T) {
	// Set up environment variables for config
	os.Setenv("RABBITMQ_ADDRESS", "amqp://test:5672")
	os.Setenv("PROMETHEUS_ENDPOINT", "/test-metrics")
	os.Setenv("REDIS_DATABASE", "2")
	defer func() {
		os.Unsetenv("RABBITMQ_ADDRESS")
		os.Unsetenv("PROMETHEUS_ENDPOINT")
		os.Unsetenv("REDIS_DATABASE")
	}()

	// Create a test command and bootstrap
	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	// Load global config
	LoadGlobalConfig()

	// Verify config was loaded
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Note: LoadGlobalConfig uses sync.Once, so it can only be tested once per test run
	// Multiple calls should not reload
	LoadGlobalConfig()
	LoadGlobalConfig()
}

func TestWriteBase64(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		expectErr bool
	}{
		{
			name:      "valid base64",
			data:      base64.StdEncoding.EncodeToString([]byte("test content")),
			expectErr: false,
		},
		{
			name:      "empty data",
			data:      "",
			expectErr: false,
		},
		{
			name:      "invalid base64",
			data:      "not!!!valid!!!base64",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := "/tmp/test-write-base64-" + tt.name + ".txt"
			defer os.Remove(tmpFile)

			err := writeBase64(tmpFile, tt.data)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify file content if no error expected and data is not empty
			if !tt.expectErr && tt.data != "" {
				content, readErr := os.ReadFile(tmpFile)
				if readErr != nil {
					t.Errorf("Could not read written file: %v", readErr)
				}
				expected, _ := base64.StdEncoding.DecodeString(tt.data)
				if string(content) != string(expected) {
					t.Errorf("File content mismatch. expected: %s, got: %s", expected, content)
				}
			}

			// Verify empty file is not created when data is empty
			if tt.data == "" {
				if _, err := os.Stat(tmpFile); err == nil {
					t.Error("Expected no file to be created for empty data, but file exists")
				}
			}
		})
	}
}

func TestPostBootstrap(t *testing.T) {
	// Create test command and bootstrap first
	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	// Set valid base64 test data
	testCert := base64.StdEncoding.EncodeToString([]byte("test cert"))
	testKey := base64.StdEncoding.EncodeToString([]byte("test key"))

	globalConfig.SSLCertBase64 = testCert
	globalConfig.SSLPrivKeyBase64 = testKey
	globalConfig.PrometheusEndpoint = "/test-metrics"
	globalConfig.PrometheusListenAddress = ":0" // Use port 0 to avoid conflicts

	defer func() {
		os.Remove(constSSLCertFilename)
		os.Remove(constSSLPrivFilename)
	}()

	err := PostBootstrap()
	if err != nil {
		t.Errorf("PostBootstrap() returned error: %v", err)
	}

	// Verify SSL files were created
	if _, err := os.Stat(constSSLCertFilename); os.IsNotExist(err) {
		t.Error("SSL cert file was not created")
	}
	if _, err := os.Stat(constSSLPrivFilename); os.IsNotExist(err) {
		t.Error("SSL private key file was not created")
	}
}

func TestInitLog(t *testing.T) {
	// This function modifies global logrus state, just ensure it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("initLog() panicked: %v", r)
		}
	}()
	initLog()
}
