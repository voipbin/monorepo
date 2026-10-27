package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

func Test_LoadGlobalConfig_cov2(t *testing.T) {
	tests := []struct {
		name string

		values map[string]any

		expectConfig Config
	}{
		{
			name:         "empty values",
			values:       map[string]any{},
			expectConfig: Config{},
		},
		{
			name: "all values with whitelist",
			values: map[string]any{
				"rabbitmq_address":          "amqp://test-host:5672",
				"prometheus_endpoint":       "/metrics",
				"prometheus_listen_address": ":2112",
				"database_dsn":              "test-user:test-pass@tcp(test-host:3306)/test",
				"redis_address":             "test-host:6379",
				"redis_password":            "test-redis-pass",
				"redis_database":            3,
				"homer_api_address":         "test-homer:9080",
				"homer_auth_token":          "test-token",
				"homer_whitelist":           "203.0.113.1,203.0.113.2",
				"asterisk_ws_port":          8089,
			},
			expectConfig: Config{
				RabbitMQAddress:         "amqp://test-host:5672",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				DatabaseDSN:             "test-user:test-pass@tcp(test-host:3306)/test",
				RedisAddress:            "test-host:6379",
				RedisPassword:           "test-redis-pass",
				RedisDatabase:           3,
				HomerAPIAddress:         "test-homer:9080",
				HomerAuthToken:          "test-token",
				HomerWhitelist:          []string{"203.0.113.1", "203.0.113.2"},
				AsteriskWSPort:          8089,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// isolate the package singleton and the viper global state.
			origConfig := globalConfig
			globalConfig = Config{}
			once = sync.Once{}
			viper.Reset()
			t.Cleanup(func() {
				viper.Reset()
				globalConfig = origConfig
				once = sync.Once{}
			})

			for k, v := range tt.values {
				viper.Set(k, v)
			}

			LoadGlobalConfig()

			if !reflect.DeepEqual(*Get(), tt.expectConfig) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConfig, *Get())
			}

			// the second call must not reload the values.
			viper.Set("rabbitmq_address", "changed")
			LoadGlobalConfig()
			if Get().RabbitMQAddress != tt.expectConfig.RabbitMQAddress {
				t.Errorf("Wrong match. config must be loaded only once. got: %s", Get().RabbitMQAddress)
			}
		})
	}
}
