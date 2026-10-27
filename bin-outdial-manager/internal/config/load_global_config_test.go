package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The test changes process wide state, so it must not run in parallel with other tests.
func Test_LoadGlobalConfig(t *testing.T) {
	// the once, the global config and the viper state are process wide
	origConfig := globalConfig
	viper.Reset()
	once = sync.Once{}
	t.Cleanup(func() {
		globalConfig = origConfig
		once = sync.Once{}
		viper.Reset()
	})

	if errBind := bindConfig(&cobra.Command{}); errBind != nil {
		t.Fatalf("could not bind the config: %v", errBind)
	}

	t.Setenv("DATABASE_DSN", "test-dsn")
	t.Setenv("PROMETHEUS_ENDPOINT", "/test-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":12345")
	t.Setenv("RABBITMQ_ADDRESS", "test-rabbitmq-address")
	t.Setenv("REDIS_ADDRESS", "test-redis-address")
	t.Setenv("REDIS_DATABASE", "7")
	t.Setenv("REDIS_PASSWORD", "test-redis-pw")

	LoadGlobalConfig()

	expect := Config{
		DatabaseDSN:             "test-dsn",
		PrometheusEndpoint:      "/test-metrics",
		PrometheusListenAddress: ":12345",
		RabbitMQAddress:         "test-rabbitmq-address",
		RedisAddress:            "test-redis-address",
		RedisDatabase:           7,
		RedisPassword:           "test-redis-pw",
	}
	if *Get() != expect {
		t.Errorf("Wrong config. expect: %+v, got: %+v", expect, *Get())
	}

	// the config is loaded once: later changes of the environment do not reach it
	t.Setenv("DATABASE_DSN", "changed-dsn")
	LoadGlobalConfig()
	if Get().DatabaseDSN != "test-dsn" {
		t.Errorf("Wrong DatabaseDSN after the second load. expect: test-dsn, got: %s", Get().DatabaseDSN)
	}
}
