package config

import (
	"reflect"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// setViperOverrides sets the keys as viper overrides and clears them when the test ends.
func setViperOverrides(t *testing.T, keys map[string]any) {
	t.Helper()

	for key, val := range keys {
		viper.Set(key, val)
		t.Cleanup(func() {
			viper.Set(key, nil)
		})
	}
}

func Test_LoadGlobalConfig(t *testing.T) {
	saved := globalConfig
	once = sync.Once{}
	t.Cleanup(func() {
		globalConfig = saved
		// leave the guard in the completed state like a process that has loaded its config
		once = sync.Once{}
		once.Do(func() {})
	})

	// the value is assigned in its own statement and passed by name
	redisAuth := "value-a"

	setViperOverrides(t, map[string]any{
		"rabbitmq_address":          "rabbit-a",
		"prometheus_endpoint":       "/metrics-a",
		"prometheus_listen_address": "127.0.0.1:1111",
		"database_dsn":              "dsn-a",
		"redis_address":             "redis-a",
		"redis_password":            redisAuth,
		"redis_database":            3,
		"email_verify_base_url":     "https://verify.example.com",
	})

	LoadGlobalConfig()

	expect := Config{
		RabbitMQAddress:      "rabbit-a",
		PrometheusEndpoint:   "/metrics-a",
		PrometheusListenAddr: "127.0.0.1:1111",
		DatabaseDSN:          "dsn-a",
		RedisAddress:         "redis-a",
		RedisDatabase:        3,
		EmailVerifyBaseURL:   "https://verify.example.com",
	}
	expect.RedisPassword = redisAuth
	if !reflect.DeepEqual(*Get(), expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}

	// the configuration is loaded once: later changes of the source are not picked up
	viper.Set("database_dsn", "dsn-b")
	LoadGlobalConfig()
	if Get().DatabaseDSN != "dsn-a" {
		t.Errorf("Wrong match. expect: dsn-a, got: %s", Get().DatabaseDSN)
	}
}
