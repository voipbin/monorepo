package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// Reset once for testing
	once.Do(func() {})

	// Create a new cmd for binding
	cmd := &cobra.Command{}

	// Bootstrap to bind config
	err := Bootstrap(cmd)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// Load config
	LoadGlobalConfig()

	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil after LoadGlobalConfig()")
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		wantErr bool
	}{
		{
			name:    "valid_command",
			cmd:     &cobra.Command{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Bootstrap(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DatabaseDSN:             "user:pass@tcp(localhost:3306)/db",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		RabbitMQAddress:         "amqp://localhost:5672",
		RedisAddress:            "localhost:6379",
		RedisDatabase:           1,
		RedisPassword:           "secret",
		GCPProjectID:            "my-project",
		GCPBucketNameMedia:      "media-bucket",
		GCPBucketNameTmp:        "tmp-bucket",
	}

	if cfg.DatabaseDSN != "user:pass@tcp(localhost:3306)/db" {
		t.Errorf("DatabaseDSN = %v, expected user:pass@tcp(localhost:3306)/db", cfg.DatabaseDSN)
	}
	if cfg.PrometheusEndpoint != "/metrics" {
		t.Errorf("PrometheusEndpoint = %v, expected /metrics", cfg.PrometheusEndpoint)
	}
	if cfg.PrometheusListenAddress != ":2112" {
		t.Errorf("PrometheusListenAddress = %v, expected :2112", cfg.PrometheusListenAddress)
	}
	if cfg.RabbitMQAddress != "amqp://localhost:5672" {
		t.Errorf("RabbitMQAddress = %v, expected amqp://localhost:5672", cfg.RabbitMQAddress)
	}
	if cfg.RedisAddress != "localhost:6379" {
		t.Errorf("RedisAddress = %v, expected localhost:6379", cfg.RedisAddress)
	}
	if cfg.RedisDatabase != 1 {
		t.Errorf("RedisDatabase = %v, expected 1", cfg.RedisDatabase)
	}
	if cfg.RedisPassword != "secret" {
		t.Errorf("RedisPassword = %v, expected secret", cfg.RedisPassword)
	}
	if cfg.GCPProjectID != "my-project" {
		t.Errorf("GCPProjectID = %v, expected my-project", cfg.GCPProjectID)
	}
	if cfg.GCPBucketNameMedia != "media-bucket" {
		t.Errorf("GCPBucketNameMedia = %v, expected media-bucket", cfg.GCPBucketNameMedia)
	}
	if cfg.GCPBucketNameTmp != "tmp-bucket" {
		t.Errorf("GCPBucketNameTmp = %v, expected tmp-bucket", cfg.GCPBucketNameTmp)
	}
}
