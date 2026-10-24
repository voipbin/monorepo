package config

import (
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetConfigState isolates a test from the package level singletons and the global
// viper and logrus state that Bootstrap modifies.
func resetConfigState(t *testing.T) {
	t.Helper()

	savedConfig := globalConfig
	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()

	viper.Reset()
	globalConfig = Config{}
	once = sync.Once{}

	t.Cleanup(func() {
		viper.Reset()
		globalConfig = savedConfig
		once = sync.Once{}
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
}

func Test_Bootstrap_defaults(t *testing.T) {
	resetConfigState(t)

	for _, key := range []string{"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS", "RABBITMQ_ADDRESS", "REDIS_ADDRESS", "REDIS_PASSWORD", "REDIS_DATABASE"} {
		t.Setenv(key, "")
	}

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := Config{
		DatabaseDSN:             "testid:testpassword@tcp(127.0.0.1:3306)/test",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		RabbitMQAddress:         "amqp://guest:***@localhost:5672",
		RedisAddress:            "127.0.0.1:6379",
		RedisDatabase:           1,
		RedisPassword:           "",
	}
	if *Get() != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}

	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("Wrong match. expect: %v, got: %v", logrus.DebugLevel, logrus.GetLevel())
	}
}

func Test_Bootstrap_env_override(t *testing.T) {
	resetConfigState(t)

	t.Setenv("DATABASE_DSN", "dbuser@tcp(db.invalid:3306)/synthetic")
	t.Setenv("PROMETHEUS_ENDPOINT", "/m")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://mq.invalid:5672")
	t.Setenv("REDIS_ADDRESS", "cache.invalid:6380")
	t.Setenv("REDIS_PASSWORD", "synthetic-secret")
	t.Setenv("REDIS_DATABASE", "7")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := Config{
		DatabaseDSN:             "dbuser@tcp(db.invalid:3306)/synthetic",
		PrometheusEndpoint:      "/m",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "amqp://mq.invalid:5672",
		RedisAddress:            "cache.invalid:6380",
		RedisDatabase:           7,
		RedisPassword:           "synthetic-secret",
	}
	if *Get() != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}
}

func Test_LoadGlobalConfig(t *testing.T) {
	resetConfigState(t)

	viper.Set("database_dsn", "first-dsn")
	viper.Set("redis_database", 3)

	LoadGlobalConfig()
	if Get().DatabaseDSN != "first-dsn" || Get().RedisDatabase != 3 {
		t.Errorf("Wrong match. got: %+v", *Get())
	}

	// the singleton is loaded only once, a later change in viper is not picked up
	viper.Set("database_dsn", "second-dsn")
	LoadGlobalConfig()
	if Get().DatabaseDSN != "first-dsn" {
		t.Errorf("Wrong match. expect: first-dsn, got: %s", Get().DatabaseDSN)
	}
}
