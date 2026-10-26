package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetLoadState gives the test a fresh once and config, and restores a fresh once and the
// previous config afterwards, so the singleton does not leak across tests.
func resetLoadState(t *testing.T) {
	t.Helper()

	saved := globalConfig
	once = sync.Once{}
	globalConfig = Config{}
	viper.Reset()

	t.Cleanup(func() {
		once = sync.Once{}
		globalConfig = saved
		viper.Reset()
	})
}

func Test_LoadGlobalConfig_values(t *testing.T) {
	resetLoadState(t)

	// credential-like values are derived from the test name instead of being written as literals
	redisValue := "synthetic-redis-" + t.Name()
	awsValue := "synthetic-aws-" + t.Name()

	t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.example.invalid:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":2112")
	t.Setenv("DATABASE_DSN", "tester@tcp(db.example.invalid:3306)/test")
	t.Setenv("REDIS_ADDRESS", "redis.example.invalid:6379")
	t.Setenv("REDIS_DATABASE", "3")
	t.Setenv("REDIS_PASSWORD", redisValue)
	t.Setenv("AWS_ACCESS_KEY", "aws-access-fake")
	t.Setenv("AWS_SECRET_KEY", awsValue)
	t.Setenv("POD_IP", "192.0.2.10")
	t.Setenv("STREAMING_LISTEN_PORT", "9090")

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap() returned an error. err: %v", err)
	}
	LoadGlobalConfig()

	got := Get()
	want := Config{
		RabbitMQAddress:         "amqp://rabbit.example.invalid:5672",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		DatabaseDSN:             "tester@tcp(db.example.invalid:3306)/test",
		RedisAddress:            "redis.example.invalid:6379",
		RedisDatabase:           3,
		AWSAccessKey:            "aws-access-fake",
		AWSSecretKey:            awsValue,
		PodIP:                   "192.0.2.10",
		StreamingListenPort:     9090,
	}
	// the password is compared on its own, then blanked so the rest compares as one struct
	if got.RedisPassword != redisValue {
		t.Errorf("LoadGlobalConfig() redis credential mismatch")
	}
	rest := *got
	rest.RedisPassword = ""
	if rest != want {
		t.Errorf("LoadGlobalConfig() mismatch.\nexpect: %+v\ngot: %+v", want, rest)
	}
}

func Test_LoadGlobalConfig_defaults(t *testing.T) {
	resetLoadState(t)

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap() returned an error. err: %v", err)
	}
	LoadGlobalConfig()

	if got := Get().StreamingListenPort; got != 8080 {
		t.Errorf("StreamingListenPort default mismatch. expect: 8080, got: %d", got)
	}
	if got := Get().RedisDatabase; got != 0 {
		t.Errorf("RedisDatabase default mismatch. expect: 0, got: %d", got)
	}
}

// Test_LoadGlobalConfig_once checks the values are locked after the first load.
func Test_LoadGlobalConfig_once(t *testing.T) {
	resetLoadState(t)

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap() returned an error. err: %v", err)
	}

	t.Setenv("REDIS_ADDRESS", "first.example.invalid:6379")
	LoadGlobalConfig()

	t.Setenv("REDIS_ADDRESS", "second.example.invalid:6379")
	LoadGlobalConfig()

	if got := Get().RedisAddress; got != "first.example.invalid:6379" {
		t.Errorf("config was reloaded. got: %s", got)
	}
}
