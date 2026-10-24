package config

import (
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fakeCredential is a built value (not a literal) that only checks the field mapping.
var fakeCredential = strings.Repeat("x", 8)

var configEnvKeys = []string{
	"RABBITMQ_ADDRESS",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"DATABASE_DSN",
	"REDIS_ADDRESS",
	"REDIS_PASSWORD",
	"REDIS_DATABASE",
}

// resetGlobals gives the test a clean package state (singleton, once and viper)
// and restores it when the test ends. The environment variables of the service
// are cleared so that the host environment cannot leak into the expectation.
// The tests that use it must not run in parallel.
func resetGlobals(t *testing.T) {
	t.Helper()

	oldCfg := cfg
	oldFormatter := logrus.StandardLogger().Formatter
	oldLevel := logrus.GetLevel()

	cfg = nil
	once = sync.Once{}
	viper.Reset()
	for _, key := range configEnvKeys {
		t.Setenv(key, "")
	}

	t.Cleanup(func() {
		cfg = oldCfg
		once = sync.Once{}
		viper.Reset()
		logrus.SetFormatter(oldFormatter)
		logrus.SetLevel(oldLevel)
	})
}

func Test_Get_panicsBeforeInitialization(t *testing.T) {
	resetGlobals(t)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong match. expect: panic, got: none")
		}
	}()

	Get()
}

func Test_Bootstrap_LoadGlobalConfig(t *testing.T) {
	t.Run("environment values are loaded into every field", func(t *testing.T) {
		resetGlobals(t)

		t.Setenv("RABBITMQ_ADDRESS", "rabbit-address-for-test")
		t.Setenv("PROMETHEUS_ENDPOINT", "/custom-metrics")
		t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
		t.Setenv("DATABASE_DSN", "dsn-for-test")
		t.Setenv("REDIS_ADDRESS", "redis-address-for-test")
		t.Setenv("REDIS_PASSWORD", fakeCredential)
		t.Setenv("REDIS_DATABASE", "7")

		cmd := &cobra.Command{Use: "test"}
		if err := Bootstrap(cmd); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		LoadGlobalConfig()

		expect := Config{
			RabbitMQAddress:         "rabbit-address-for-test",
			PrometheusEndpoint:      "/custom-metrics",
			PrometheusListenAddress: ":9999",
			DatabaseDSN:             "dsn-for-test",
			RedisAddress:            "redis-address-for-test",
			RedisPassword:           fakeCredential,
			RedisDatabase:           7,
		}
		if got := *Get(); got != expect {
			t.Errorf("Wrong config.\nexpect: %+v\ngot: %+v", expect, got)
		}
	})

	t.Run("flag defaults are used when no environment value is set", func(t *testing.T) {
		resetGlobals(t)

		cmd := &cobra.Command{Use: "test"}
		if err := Bootstrap(cmd); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		LoadGlobalConfig()

		expect := Config{
			PrometheusEndpoint:      "/metrics",
			PrometheusListenAddress: ":2112",
		}
		if got := *Get(); got != expect {
			t.Errorf("Wrong config.\nexpect: %+v\ngot: %+v", expect, got)
		}
	})

	t.Run("the first load wins: a later call does not reload", func(t *testing.T) {
		resetGlobals(t)

		t.Setenv("DATABASE_DSN", "first")
		cmd := &cobra.Command{Use: "test"}
		if err := Bootstrap(cmd); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		LoadGlobalConfig()

		t.Setenv("DATABASE_DSN", "second")
		LoadGlobalConfig()

		if got := Get().DatabaseDSN; got != "first" {
			t.Errorf("Wrong match. expect: first, got: %s", got)
		}
	})
}

func Test_Init(t *testing.T) {
	t.Run("a flag set on the command line wins over the environment", func(t *testing.T) {
		resetGlobals(t)

		t.Setenv("DATABASE_DSN", "from-env")
		t.Setenv("RABBITMQ_ADDRESS", "from-env-rabbit-address")
		t.Setenv("REDIS_DATABASE", "5")

		cmd := &cobra.Command{Use: "test"}
		RegisterFlags(cmd)
		if err := cmd.Flags().Set("database_dsn", "from-flag"); err != nil {
			t.Fatalf("Could not set the flag. err: %v", err)
		}

		Init(cmd)

		got := Get()
		if got.DatabaseDSN != "from-flag" {
			t.Errorf("Wrong DatabaseDSN. expect: from-flag, got: %s", got.DatabaseDSN)
		}
		if got.RabbitMQAddress != "from-env-rabbit-address" {
			t.Errorf("Wrong RabbitMQAddress. expect: the environment value, got: %s", got.RabbitMQAddress)
		}
		if got.RedisDatabase != 5 {
			t.Errorf("Wrong RedisDatabase. expect: 5, got: %d", got.RedisDatabase)
		}
	})

	t.Run("registered flag defaults are used when nothing else is set", func(t *testing.T) {
		resetGlobals(t)

		cmd := &cobra.Command{Use: "test"}
		RegisterFlags(cmd)
		Init(cmd)

		expect := Config{
			DatabaseDSN:             defaultDatabaseDSN,
			PrometheusEndpoint:      defaultPrometheusEndpoint,
			PrometheusListenAddress: defaultPrometheusListenAddress,
			RabbitMQAddress:         defaultRabbitMQAddress,
			RedisAddress:            defaultRedisAddress,
			RedisDatabase:           defaultRedisDatabase,
			RedisPassword:           defaultRedisPassword,
		}
		if got := *Get(); got != expect {
			t.Errorf("Wrong config.\nexpect: %+v\ngot: %+v", expect, got)
		}
	})

	t.Run("Init and LoadGlobalConfig share one initialization", func(t *testing.T) {
		resetGlobals(t)

		cmd := &cobra.Command{Use: "test"}
		RegisterFlags(cmd)
		t.Setenv("REDIS_ADDRESS", "first-redis-address")
		Init(cmd)

		t.Setenv("REDIS_ADDRESS", "second-redis-address")
		LoadGlobalConfig()

		if got := Get().RedisAddress; got != "first-redis-address" {
			t.Errorf("Wrong match. expect: the first initialization wins, got: %s", got)
		}
	})
}
