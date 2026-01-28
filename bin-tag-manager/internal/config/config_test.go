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
				DatabaseDSN:             "user:pass@tcp(127.0.0.1:3306)/db",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				RedisAddress:            "127.0.0.1:6379",
				RedisDatabase:           1,
				RedisPassword:           "secret",
			},
			expectCfg: Config{
				DatabaseDSN:             "user:pass@tcp(127.0.0.1:3306)/db",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				RedisAddress:            "127.0.0.1:6379",
				RedisDatabase:           1,
				RedisPassword:           "secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = tt.setupConfig

			res := Get()

			if res.DatabaseDSN != tt.expectCfg.DatabaseDSN {
				t.Errorf("Wrong DatabaseDSN. expect: %s, got: %s", tt.expectCfg.DatabaseDSN, res.DatabaseDSN)
			}
			if res.PrometheusEndpoint != tt.expectCfg.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.expectCfg.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.expectCfg.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.expectCfg.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.RabbitMQAddress != tt.expectCfg.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.expectCfg.RabbitMQAddress, res.RabbitMQAddress)
			}
			if res.RedisAddress != tt.expectCfg.RedisAddress {
				t.Errorf("Wrong RedisAddress. expect: %s, got: %s", tt.expectCfg.RedisAddress, res.RedisAddress)
			}
			if res.RedisDatabase != tt.expectCfg.RedisDatabase {
				t.Errorf("Wrong RedisDatabase. expect: %d, got: %d", tt.expectCfg.RedisDatabase, res.RedisDatabase)
			}
			if res.RedisPassword != tt.expectCfg.RedisPassword {
				t.Errorf("Wrong RedisPassword. expect: %s, got: %s", tt.expectCfg.RedisPassword, res.RedisPassword)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name string

		databaseDSN             string
		prometheusEndpoint      string
		prometheusListenAddress string
		rabbitmqAddress         string
		redisAddress            string
		redisPassword           string
		redisDatabase           int

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			databaseDSN:             "testid:testpassword@tcp(127.0.0.1:3306)/test",
			prometheusEndpoint:      "/metrics",
			prometheusListenAddress: ":2112",
			rabbitmqAddress:         "amqp://guest:guest@localhost:5672",
			redisAddress:            "127.0.0.1:6379",
			redisPassword:           "",
			redisDatabase:           1,

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("database_dsn", tt.databaseDSN, "")
			cmd.Flags().String("prometheus_endpoint", tt.prometheusEndpoint, "")
			cmd.Flags().String("prometheus_listen_address", tt.prometheusListenAddress, "")
			cmd.Flags().String("rabbitmq_address", tt.rabbitmqAddress, "")
			cmd.Flags().String("redis_address", tt.redisAddress, "")
			cmd.Flags().String("redis_password", tt.redisPassword, "")
			cmd.Flags().Int("redis_database", tt.redisDatabase, "")

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
			if res.DatabaseDSN != tt.databaseDSN {
				t.Errorf("Wrong DatabaseDSN. expect: %s, got: %s", tt.databaseDSN, res.DatabaseDSN)
			}
		})
	}
}
