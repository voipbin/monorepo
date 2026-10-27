package config

import (
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// Test_LoadGlobalConfig_coverage90 verifies that the values of viper are copied into the global
// config once, and that a later change of viper does not reach the locked config.
func Test_LoadGlobalConfig_coverage90(t *testing.T) {
	prevConfig := globalConfig
	viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}
	t.Cleanup(func() {
		viper.Reset()
		once = sync.Once{}
		globalConfig = prevConfig
	})

	redisPassword := "synthetic-redis-credential"
	viper.Set("rabbitmq_address", "synthetic-rabbitmq-address")
	viper.Set("prometheus_endpoint", "/synthetic-metrics")
	viper.Set("prometheus_listen_address", "127.0.0.1:0")
	viper.Set("database_dsn", "synthetic-database-dsn")
	viper.Set("redis_address", "synthetic-redis-address")
	viper.Set("redis_password", redisPassword)
	viper.Set("redis_database", 3)

	LoadGlobalConfig()

	expect := Config{
		RabbitMQAddress:         "synthetic-rabbitmq-address",
		PrometheusEndpoint:      "/synthetic-metrics",
		PrometheusListenAddress: "127.0.0.1:0",
		DatabaseDSN:             "synthetic-database-dsn",
		RedisAddress:            "synthetic-redis-address",
		RedisPassword:           redisPassword,
		RedisDatabase:           3,
	}
	if got := *Get(); got != expect {
		t.Errorf("Wrong config.\nexpect: %+v\ngot: %+v", expect, got)
	}

	// the config is locked by the first load.
	viper.Set("redis_address", "changed-redis-address")
	LoadGlobalConfig()
	if got := Get().RedisAddress; got != "synthetic-redis-address" {
		t.Errorf("The loaded config must stay locked. got: %s", got)
	}
}
