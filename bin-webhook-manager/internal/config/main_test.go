package config

import (
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetState clears the process-wide state the config package keeps (viper, the
// loaded singleton, the once guard and the logrus settings) and restores it
// when the test ends. Tests using it must not run in parallel.
func resetState(t *testing.T) {
	t.Helper()

	formatter := logrus.StandardLogger().Formatter
	level := logrus.GetLevel()

	viper.Reset()
	globalConfig = Config{}
	once = sync.Once{}

	t.Cleanup(func() {
		viper.Reset()
		globalConfig = Config{}
		once = sync.Once{}
		logrus.SetFormatter(formatter)
		logrus.SetLevel(level)
	})
}

func newTestCommand() *cobra.Command {
	return &cobra.Command{Use: "test"}
}

func Test_LoadGlobalConfig_env(t *testing.T) {
	resetState(t)

	t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.test:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/metrics-test")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":2112")
	t.Setenv("DATABASE_DSN", "dsn-placeholder")
	t.Setenv("REDIS_ADDRESS", "redis.test:6379")
	t.Setenv("REDIS_PASSWORD", "placeholder")
	t.Setenv("REDIS_DATABASE", "3")

	if err := bindConfig(newTestCommand()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	expect := Config{
		RabbitMQAddress:         "amqp://rabbit.test:5672",
		PrometheusEndpoint:      "/metrics-test",
		PrometheusListenAddress: ":2112",
		DatabaseDSN:             "dsn-placeholder",
		RedisAddress:            "redis.test:6379",
		RedisPassword:           "placeholder",
		RedisDatabase:           3,
	}

	if got := *Get(); got != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, got)
	}
}

func Test_LoadGlobalConfig_flag_overrides_env(t *testing.T) {
	resetState(t)

	t.Setenv("REDIS_ADDRESS", "redis.env:6379")
	t.Setenv("REDIS_DATABASE", "1")

	cmd := newTestCommand()
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := cmd.PersistentFlags().Parse([]string{"--redis_address=redis.flag:6379", "--redis_database=7"}); err != nil {
		t.Fatalf("Could not parse the flags. err: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().RedisAddress; got != "redis.flag:6379" {
		t.Errorf("Wrong match. expect: redis.flag:6379, got: %s", got)
	}
	if got := Get().RedisDatabase; got != 7 {
		t.Errorf("Wrong match. expect: 7, got: %d", got)
	}
}

func Test_LoadGlobalConfig_defaults_when_unset(t *testing.T) {
	resetState(t)

	for _, key := range []string{
		"RABBITMQ_ADDRESS", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"DATABASE_DSN", "REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE",
	} {
		t.Setenv(key, "")
	}

	if err := bindConfig(newTestCommand()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if got := *Get(); got != (Config{}) {
		t.Errorf("Wrong match. expect: zero config, got: %+v", got)
	}
}

func Test_LoadGlobalConfig_loads_once(t *testing.T) {
	resetState(t)

	t.Setenv("REDIS_ADDRESS", "redis.first:6379")
	if err := bindConfig(newTestCommand()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	t.Setenv("REDIS_ADDRESS", "redis.second:6379")
	LoadGlobalConfig()

	if got := Get().RedisAddress; got != "redis.first:6379" {
		t.Errorf("Wrong match. expect the first value to stay locked: redis.first:6379, got: %s", got)
	}
}

func Test_Bootstrap(t *testing.T) {
	resetState(t)

	cmd := newTestCommand()
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, name := range []string{
		"rabbitmq_address", "prometheus_endpoint", "prometheus_listen_address",
		"database_dsn", "redis_address", "redis_password", "redis_database",
	} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect the flag to be registered: %s", name)
		}
	}

	if got := logrus.GetLevel(); got != logrus.DebugLevel {
		t.Errorf("Wrong match. expect: %v, got: %v", logrus.DebugLevel, got)
	}
}
