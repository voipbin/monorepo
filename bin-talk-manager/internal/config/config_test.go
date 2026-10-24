package config

import (
	"reflect"
	"sync"
	"testing"

	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var allEnvKeys = []string{
	"RABBITMQ_ADDRESS",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"DATABASE_DSN",
	"REDIS_ADDRESS",
	"REDIS_PASSWORD",
	"REDIS_DATABASE",
}

// resetState clears the package and viper globals and restores them (and the logrus
// settings that Bootstrap changes) when the test ends.
func resetState(t *testing.T) {
	t.Helper()

	oldFormatter := logrus.StandardLogger().Formatter
	oldLevel := logrus.GetLevel()
	oldOut := logrus.StandardLogger().Out

	for _, k := range allEnvKeys {
		t.Setenv(k, "")
	}

	viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Cleanup(func() {
		viper.Reset()
		once = sync.Once{}
		globalConfig = Config{}
		logrus.SetFormatter(oldFormatter)
		logrus.SetLevel(oldLevel)
		logrus.SetOutput(oldOut)
	})
}

func Test_bindConfig_defaults(t *testing.T) {
	resetState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bindConfig() error = %v", err)
	}

	LoadGlobalConfig()

	want := Config{
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
	}
	if got := *Get(); !reflect.DeepEqual(got, want) {
		t.Errorf("config = %+v, want %+v", got, want)
	}

	for _, name := range []string{
		"rabbitmq_address", "prometheus_endpoint", "prometheus_listen_address",
		"database_dsn", "redis_address", "redis_password", "redis_database",
	} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("flag %q is not registered", name)
		}
	}
}

func Test_LoadGlobalConfig_env(t *testing.T) {
	resetState(t)

	// every field gets its own distinct value, so a field that reads another field's viper key is detected
	t.Setenv("RABBITMQ_ADDRESS", "amqp://mq.example.invalid:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/custom-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("DATABASE_DSN", "tcp(db.example.invalid:3306)/test")
	t.Setenv("REDIS_ADDRESS", "redis.example.invalid:6379")
	t.Setenv("REDIS_PASSWORD", "placeholder")
	t.Setenv("REDIS_DATABASE", "3")

	if err := bindConfig(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("bindConfig() error = %v", err)
	}

	LoadGlobalConfig()

	want := Config{
		RabbitMQAddress:         "amqp://mq.example.invalid:5672",
		PrometheusEndpoint:      "/custom-metrics",
		PrometheusListenAddress: ":9999",
		DatabaseDSN:             "tcp(db.example.invalid:3306)/test",
		RedisAddress:            "redis.example.invalid:6379",
		RedisPassword:           "placeholder",
		RedisDatabase:           3,
	}
	if got := *Get(); !reflect.DeepEqual(got, want) {
		t.Errorf("config = %+v, want %+v", got, want)
	}
}

func Test_LoadGlobalConfig_once(t *testing.T) {
	resetState(t)

	t.Setenv("REDIS_ADDRESS", "first.example.invalid:6379")
	if err := bindConfig(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("bindConfig() error = %v", err)
	}

	LoadGlobalConfig()

	t.Setenv("REDIS_ADDRESS", "second.example.invalid:6379")
	LoadGlobalConfig()

	if got := Get().RedisAddress; got != "first.example.invalid:6379" {
		t.Errorf("RedisAddress = %q, want the value of the first load", got)
	}
}

func Test_LoadGlobalConfig_flag_over_env(t *testing.T) {
	resetState(t)

	t.Setenv("DATABASE_DSN", "from-env")

	cmd := &cobra.Command{Use: "test"}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bindConfig() error = %v", err)
	}
	if err := cmd.PersistentFlags().Set("database_dsn", "from-flag"); err != nil {
		t.Fatalf("could not set the flag: %v", err)
	}

	LoadGlobalConfig()

	if got := Get().DatabaseDSN; got != "from-flag" {
		t.Errorf("DatabaseDSN = %q, want the explicit flag value", got)
	}
}

func Test_Bootstrap(t *testing.T) {
	resetState(t)

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if got := logrus.GetLevel(); got != logrus.DebugLevel {
		t.Errorf("log level = %v, want debug", got)
	}
	if _, ok := logrus.StandardLogger().Formatter.(*joonix.Formatter); !ok {
		t.Errorf("formatter = %T, want the joonix formatter", logrus.StandardLogger().Formatter)
	}
	if cmd.PersistentFlags().Lookup("database_dsn") == nil {
		t.Errorf("Bootstrap() did not register the flags")
	}
}
