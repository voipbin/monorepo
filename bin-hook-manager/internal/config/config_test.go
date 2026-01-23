package config

import (
	"testing"
)

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DatabaseDSN:             "user:pass@tcp(localhost:3306)/db",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":8080",
		RabbitMQAddress:         "amqp://localhost:5672",
		SSLPrivkeyBase64:        "dGVzdA==",
		SSLCertBase64:           "dGVzdA==",
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"DatabaseDSN", cfg.DatabaseDSN, "user:pass@tcp(localhost:3306)/db"},
		{"PrometheusEndpoint", cfg.PrometheusEndpoint, "/metrics"},
		{"PrometheusListenAddress", cfg.PrometheusListenAddress, ":8080"},
		{"RabbitMQAddress", cfg.RabbitMQAddress, "amqp://localhost:5672"},
		{"SSLPrivkeyBase64", cfg.SSLPrivkeyBase64, "dGVzdA=="},
		{"SSLCertBase64", cfg.SSLCertBase64, "dGVzdA=="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}
