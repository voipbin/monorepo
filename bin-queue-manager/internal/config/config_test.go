package config

import (
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetConfigState isolates a test from the process-global state that
// Bootstrap and LoadGlobalConfig mutate (viper, the logrus formatter and
// level, the config singleton and its sync.Once) and restores it afterwards.
func resetConfigState(t *testing.T) {
	t.Helper()

	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()

	reset := func() {
		viper.Reset()
		globalConfig = Config{}
		once = sync.Once{}
	}

	reset()
	t.Cleanup(func() {
		reset()
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
	})

	// an empty value is treated as unset by viper's automatic env lookup, which
	// keeps the host environment from leaking into the expectations.
	for _, key := range []string{
		"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"RABBITMQ_ADDRESS", "REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE",
	} {
		t.Setenv(key, "")
	}
}

func Test_Bootstrap_defaults(t *testing.T) {
	resetConfigState(t)

	if err := Bootstrap(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	cfg := Get()
	if cfg.PrometheusEndpoint != "/metrics" {
		t.Errorf("Wrong match. expect: /metrics, got: %s", cfg.PrometheusEndpoint)
	}
	if cfg.PrometheusListenAddress != ":2112" {
		t.Errorf("Wrong match. expect: :2112, got: %s", cfg.PrometheusListenAddress)
	}
	if cfg.RedisAddress != "127.0.0.1:6379" {
		t.Errorf("Wrong match. expect: 127.0.0.1:6379, got: %s", cfg.RedisAddress)
	}
	if cfg.RedisDatabase != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", cfg.RedisDatabase)
	}
	if cfg.RedisPassword != "" {
		t.Errorf("Wrong match. expect: empty password by default, got a non-empty value")
	}
	if cfg.DatabaseDSN == "" || cfg.RabbitMQAddress == "" {
		t.Errorf("Wrong match. the database and rabbitmq defaults must be non-empty")
	}

	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("Wrong match. expect: debug log level, got: %v", logrus.GetLevel())
	}
}

func Test_Bootstrap_environmentOverridesDefaults(t *testing.T) {
	resetConfigState(t)

	t.Setenv("DATABASE_DSN", "dsn-from-env")
	t.Setenv("PROMETHEUS_ENDPOINT", "/env-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("RABBITMQ_ADDRESS", "rabbit-from-env")
	t.Setenv("REDIS_ADDRESS", "redis-from-env:6380")
	t.Setenv("REDIS_PASSWORD", "redis-secret-from-env")
	t.Setenv("REDIS_DATABASE", "7")

	if err := Bootstrap(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := Config{
		DatabaseDSN:             "dsn-from-env",
		PrometheusEndpoint:      "/env-metrics",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "rabbit-from-env",
		RedisAddress:            "redis-from-env:6380",
		RedisDatabase:           7,
		RedisPassword:           "redis-secret-from-env",
	}
	if *Get() != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}
}

func Test_Bootstrap_flagOverridesDefault(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the flags are registered on the persistent flag set of the command.
	for _, name := range []string{
		"database_dsn", "prometheus_endpoint", "prometheus_listen_address",
		"rabbitmq_address", "redis_address", "redis_password", "redis_database",
	} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect: flag %s to be registered", name)
		}
	}

	if err := cmd.PersistentFlags().Set("redis_database", "5"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got := viper.GetInt("redis_database"); got != 5 {
		t.Errorf("Wrong match. expect: the flag value 5 to be visible through viper, got: %d", got)
	}
}

func Test_LoadGlobalConfig(t *testing.T) {
	resetConfigState(t)

	if err := Bootstrap(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// drop what Bootstrap loaded so only LoadGlobalConfig can fill the singleton.
	globalConfig = Config{}
	once = sync.Once{}

	t.Setenv("DATABASE_DSN", "dsn-first")
	t.Setenv("PROMETHEUS_ENDPOINT", "/first-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":8081")
	t.Setenv("RABBITMQ_ADDRESS", "rabbit-first")
	t.Setenv("REDIS_ADDRESS", "redis-first:6379")
	t.Setenv("REDIS_PASSWORD", "redis-secret-first")
	t.Setenv("REDIS_DATABASE", "3")

	LoadGlobalConfig()

	expect := Config{
		DatabaseDSN:             "dsn-first",
		PrometheusEndpoint:      "/first-metrics",
		PrometheusListenAddress: ":8081",
		RabbitMQAddress:         "rabbit-first",
		RedisAddress:            "redis-first:6379",
		RedisDatabase:           3,
		RedisPassword:           "redis-secret-first",
	}
	if *Get() != expect {
		t.Fatalf("Wrong match. the singleton must be loaded from viper.\nexpect: %+v\ngot: %+v", expect, *Get())
	}

	// the singleton is loaded exactly once: later environment changes are not picked up.
	t.Setenv("REDIS_ADDRESS", "redis-second:6379")
	t.Setenv("REDIS_DATABASE", "4")
	LoadGlobalConfig()

	if *Get() != expect {
		t.Errorf("Wrong match. a second LoadGlobalConfig must not reload.\nexpect: %+v\ngot: %+v", expect, *Get())
	}
}
