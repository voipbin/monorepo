package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
		expectCfg   Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
			expectCfg:   Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
			},
			expectCfg: Config{
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the global config
			cfg = tt.setupConfig

			res := Get()

			if res.PrometheusEndpoint != tt.expectCfg.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.expectCfg.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.expectCfg.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.expectCfg.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.expectCfg.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.expectCfg.RabbitMQAddress, res.RabbitMQAddress)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name string

		prometheusEndpoint      string
		prometheusListenAddress string
		rabbitmqAddress         string

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			prometheusEndpoint:      "/metrics",
			prometheusListenAddress: ":2112",
			rabbitmqAddress:         "amqp://guest:guest@localhost:5672",

			expectErr: false,
		},
		{
			name: "initializes_with_custom_values",

			prometheusEndpoint:      "/custom-metrics",
			prometheusListenAddress: ":9090",
			rabbitmqAddress:         "amqp://user:pass@rabbitmq.example.com:5672",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("prometheus_endpoint", tt.prometheusEndpoint, "")
			cmd.Flags().String("prometheus_listen_address", tt.prometheusListenAddress, "")
			cmd.Flags().String("rabbitmq_address", tt.rabbitmqAddress, "")

			err := InitConfig(cmd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			res := Get()
			if res.PrometheusEndpoint != tt.prometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.prometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.prometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.prometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.rabbitmqAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.rabbitmqAddress, res.RabbitMQAddress)
			}
		})
	}
}
