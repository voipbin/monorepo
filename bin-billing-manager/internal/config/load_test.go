package config

import (
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// Test_LoadGlobalConfig loads the process config from viper once and keeps the first values.
func Test_LoadGlobalConfig(t *testing.T) {
	savedConfig := globalConfig
	t.Cleanup(func() {
		globalConfig = savedConfig
		once = sync.Once{}
		viper.Reset()
	})

	once = sync.Once{}
	globalConfig = Config{}
	viper.Reset()
	viper.Set("rabbitmq_address", "amqp://rabbit.test:5672")
	viper.Set("prometheus_endpoint", "/metrics-test")
	viper.Set("prometheus_listen_address", ":18080")
	viper.Set("database_dsn", "test-dsn")
	viper.Set("redis_address", "cache-host-test")
	viper.Set("redis_database", 3)
	viper.Set("paddle_api_key", "test-paddle-key")
	viper.Set("paddle_price_id_basic", "pri_basic_test")
	viper.Set("paddle_price_id_professional", "pri_pro_test")

	LoadGlobalConfig()

	expect := Config{
		RabbitMQAddress:           "amqp://rabbit.test:5672",
		PrometheusEndpoint:        "/metrics-test",
		PrometheusListenAddress:   ":18080",
		DatabaseDSN:               "test-dsn",
		RedisAddress:              "cache-host-test",
		RedisDatabase:             3,
		PaddleAPIKey:              "test-paddle-key",
		PaddlePriceIDBasic:        "pri_basic_test",
		PaddlePriceIDProfessional: "pri_pro_test",
	}
	if *Get() != expect {
		t.Errorf("wrong config. expect: %+v, got: %+v", expect, *Get())
	}

	// the config is locked after the first load; a later change in viper is not picked up
	viper.Set("database_dsn", "changed-dsn")
	LoadGlobalConfig()
	if Get().DatabaseDSN != "test-dsn" {
		t.Errorf("the loaded config must not change. got: %s", Get().DatabaseDSN)
	}
}
