package config

import (
	"os"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configEnvNames = []string{
	"RABBITMQ_ADDRESS",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"DATABASE_DSN",
	"REDIS_ADDRESS",
	"REDIS_PASSWORD",
	"REDIS_DATABASE",
	"SCHEDULE_TICK_INTERVAL_SEC",
	"SCHEDULE_DISPATCH_CONCURRENCY",
	"SCHEDULE_EXECUTION_RETENTION_DAYS",
	"SCHEDULE_BACKUP_DIR",
	"SCHEDULE_BACKUP_RETENTION_COUNT",
}

// resetConfigState clears every config env name, the viper global, the config
// singleton and the logrus global, and restores them when the test ends.
func resetConfigState(t *testing.T) {
	t.Helper()

	for _, name := range configEnvNames {
		// t.Setenv registers the restore of the original value
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("could not unset the env. name: %s, err: %v", name, err)
		}
	}

	oldConfig := globalConfig
	oldFormatter := logrus.StandardLogger().Formatter
	oldLevel := logrus.GetLevel()
	viper.Reset()
	globalConfig = Config{}
	once = sync.Once{}

	t.Cleanup(func() {
		viper.Reset()
		globalConfig = oldConfig
		once = sync.Once{}
		logrus.SetFormatter(oldFormatter)
		logrus.SetLevel(oldLevel)
	})
}

func Test_LoadGlobalConfig_defaults(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	expect := Config{
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		RedisDatabase:           1,

		ScheduleTickIntervalSec:        10,
		ScheduleDispatchConcurrency:    10,
		ScheduleExecutionRetentionDays: 90,
		ScheduleBackupRetentionCount:   7,
	}

	if *Get() != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}
}

func Test_LoadGlobalConfig_env(t *testing.T) {
	resetConfigState(t)

	// every field gets a distinct value so that swapped keys are detected
	redisSecret := "fake-value-1"
	t.Setenv("RABBITMQ_ADDRESS", "amqp.example.invalid:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/m2")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("DATABASE_DSN", "dsn-value")
	t.Setenv("REDIS_ADDRESS", "redis.example.invalid:6379")
	t.Setenv("REDIS_PASSWORD", redisSecret)
	t.Setenv("REDIS_DATABASE", "5")
	t.Setenv("SCHEDULE_TICK_INTERVAL_SEC", "3")
	t.Setenv("SCHEDULE_DISPATCH_CONCURRENCY", "4")
	t.Setenv("SCHEDULE_EXECUTION_RETENTION_DAYS", "30")
	t.Setenv("SCHEDULE_BACKUP_DIR", "/backup/dir")
	t.Setenv("SCHEDULE_BACKUP_RETENTION_COUNT", "2")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	expect := Config{
		RabbitMQAddress:         "amqp.example.invalid:5672",
		PrometheusEndpoint:      "/m2",
		PrometheusListenAddress: ":9999",
		DatabaseDSN:             "dsn-value",
		RedisAddress:            "redis.example.invalid:6379",
		RedisPassword:           redisSecret,
		RedisDatabase:           5,

		ScheduleTickIntervalSec:        3,
		ScheduleDispatchConcurrency:    4,
		ScheduleExecutionRetentionDays: 30,
		ScheduleBackupDir:              "/backup/dir",
		ScheduleBackupRetentionCount:   2,
	}

	if *Get() != expect {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *Get())
	}
}

func Test_LoadGlobalConfig_empty_env_is_unset(t *testing.T) {
	resetConfigState(t)

	t.Setenv("SCHEDULE_TICK_INTERVAL_SEC", "")
	t.Setenv("PROMETHEUS_ENDPOINT", "")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if Get().ScheduleTickIntervalSec != 10 {
		t.Errorf("Wrong match. expect: 10, got: %d", Get().ScheduleTickIntervalSec)
	}
	if Get().PrometheusEndpoint != "/metrics" {
		t.Errorf("Wrong match. expect: /metrics, got: %s", Get().PrometheusEndpoint)
	}
}

func Test_LoadGlobalConfig_flag(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if err := cmd.PersistentFlags().Set("schedule_dispatch_concurrency", "6"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := cmd.PersistentFlags().Set("redis_address", "flag-redis:6379"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	if Get().ScheduleDispatchConcurrency != 6 {
		t.Errorf("Wrong match. expect: 6, got: %d", Get().ScheduleDispatchConcurrency)
	}
	if Get().RedisAddress != "flag-redis:6379" {
		t.Errorf("Wrong match. expect: flag-redis:6379, got: %s", Get().RedisAddress)
	}
}

func Test_LoadGlobalConfig_locked_after_first_load(t *testing.T) {
	resetConfigState(t)

	t.Setenv("SCHEDULE_TICK_INTERVAL_SEC", "3")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	t.Setenv("SCHEDULE_TICK_INTERVAL_SEC", "99")
	LoadGlobalConfig()

	if Get().ScheduleTickIntervalSec != 3 {
		t.Errorf("Wrong match. expect: 3, got: %d", Get().ScheduleTickIntervalSec)
	}
}

func Test_Bootstrap_sets_logrus(t *testing.T) {
	resetConfigState(t)

	logrus.SetLevel(logrus.ErrorLevel)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("Wrong match. expect: %v, got: %v", logrus.DebugLevel, logrus.GetLevel())
	}
}
