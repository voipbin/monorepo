package config

import (
	"reflect"
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

	t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.example.invalid:5672")
	t.Setenv("PROMETHEUS_ENDPOINT", "/metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":2112")
	t.Setenv("DATABASE_DSN", "tester@tcp(db.example.invalid:3306)/test")
	t.Setenv("REDIS_ADDRESS", "redis.example.invalid:6379")
	t.Setenv("REDIS_DATABASE", "3")
	t.Setenv("REDIS_PASSWORD", redisValue)
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

// Test_Config_removedFields checks that the provider secrets are not fields of the Config any
// more: they are read from the environment by transcribeconfig.
func Test_Config_removedFields(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for _, name := range []string{"AWSAccessKey", "AWSSecretKey"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("Wrong match. Config must not have the field %s", name)
		}
	}
	if _, ok := reflect.TypeOf(&Config{}).MethodByName("EnvValue"); ok {
		t.Errorf("Wrong match. Config must not have the method EnvValue")
	}
}

// Test_LoadGlobalConfig_oldVariablesIgnored checks that a deployment that still exports the old
// variables starts normally: the load succeeds and no other field is affected.
func Test_LoadGlobalConfig_oldVariablesIgnored(t *testing.T) {
	load := func(t *testing.T) Config {
		t.Helper()
		resetLoadState(t)
		t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.example.invalid:5672")
		t.Setenv("REDIS_DATABASE", "3")
		t.Setenv("STREAMING_LISTEN_PORT", "9090")

		return *loadForTest(t)
	}

	base := load(t)

	t.Setenv("AWS_ACCESS_KEY", "aws-access-fake")
	t.Setenv("AWS_SECRET_KEY", "aws-secret-fake")
	t.Setenv("AWS_ACCESS_KEY_ID", "aws-access-id-fake")
	got := load(t)

	if got != base {
		t.Errorf("Wrong match. the old variables changed the config.\nexpect: %+v\ngot: %+v", base, got)
	}
	if got.RabbitMQAddress != "amqp://rabbit.example.invalid:5672" || got.RedisDatabase != 3 || got.StreamingListenPort != 9090 {
		t.Errorf("Wrong match. got: %+v", got)
	}
}

func loadForTest(t *testing.T) *Config {
	t.Helper()

	if err := Bootstrap(&cobra.Command{}); err != nil {
		t.Fatalf("Bootstrap() returned an error. err: %v", err)
	}
	LoadGlobalConfig()

	return Get()
}
