package config

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resetConfigState restores the package singletons and the global viper so
// each test starts from a clean state.
func resetConfigState(t *testing.T) {
	t.Helper()

	viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Cleanup(func() {
		viper.Reset()
		once = sync.Once{}
		globalConfig = Config{}
	})
}

func Test_Bootstrap_registersFlags(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tests := []struct {
		name        string
		flag        string
		expectValue string
	}{
		{"prometheus endpoint", "prometheus_endpoint", "/metrics"},
		{"prometheus listen address", "prometheus_listen_address", ":2112"},
		{"redis database", "redis_database", "1"},
		{"health check interval", "health_check_interval", "30s"},
		{"redis password", "redis_password", ""},
		{"sip gateway fqdn", "external_sip_gateway_fqdn_for_pstn", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(tt.flag)
			if f == nil {
				t.Fatalf("Wrong match. expect: flag %s registered, got: nil", tt.flag)
			}

			if f.DefValue != tt.expectValue {
				t.Errorf("Wrong match.\nexpect: %q\ngot: %q", tt.expectValue, f.DefValue)
			}
		})
	}

	for _, name := range []string{"rabbitmq_address", "database_dsn", "redis_address"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect: flag %s registered, got: nil", name)
		}
	}
}

func Test_LoadGlobalConfig_fromEnv(t *testing.T) {
	resetConfigState(t)

	redisPassword := "synthetic-redis-secret"

	t.Setenv("DATABASE_DSN", "tcp(db.example.invalid:3306)/testdb")
	t.Setenv("PROMETHEUS_ENDPOINT", "/custom-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://mq.example.invalid:5672")
	t.Setenv("REDIS_ADDRESS", "redis.example.invalid:6379")
	t.Setenv("REDIS_DATABASE", "7")
	t.Setenv("REDIS_PASSWORD", redisPassword)
	t.Setenv("HEALTH_CHECK_INTERVAL", "45s")
	t.Setenv("EXTERNAL_SIP_GATEWAY_FQDN_FOR_PSTN", "gw.example.invalid:5060")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	LoadGlobalConfig()
	got := Get()

	expect := Config{
		DatabaseDSN:             "tcp(db.example.invalid:3306)/testdb",
		PrometheusEndpoint:      "/custom-metrics",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "amqp://mq.example.invalid:5672",
		RedisAddress:            "redis.example.invalid:6379",
		RedisDatabase:           7,
		RedisPassword:           redisPassword,
		HealthCheckInterval:     45 * time.Second,
		SipGatewayFQDNForPSTN:   "gw.example.invalid:5060",
	}
	if !reflect.DeepEqual(*got, expect) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", expect, *got)
	}
}

func Test_LoadGlobalConfig_defaults(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	LoadGlobalConfig()
	got := Get()

	if got.PrometheusEndpoint != "/metrics" {
		t.Errorf("Wrong match. expect: /metrics, got: %s", got.PrometheusEndpoint)
	}
	if got.PrometheusListenAddress != ":2112" {
		t.Errorf("Wrong match. expect: :2112, got: %s", got.PrometheusListenAddress)
	}
	if got.RedisDatabase != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", got.RedisDatabase)
	}
	if got.HealthCheckInterval != 30*time.Second {
		t.Errorf("Wrong match. expect: 30s, got: %s", got.HealthCheckInterval)
	}
}

func Test_LoadGlobalConfig_onlyOnce(t *testing.T) {
	resetConfigState(t)

	t.Setenv("PROMETHEUS_ENDPOINT", "/first")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	LoadGlobalConfig()

	t.Setenv("PROMETHEUS_ENDPOINT", "/second")
	LoadGlobalConfig()

	if Get().PrometheusEndpoint != "/first" {
		t.Errorf("Wrong match. expect: /first, got: %s", Get().PrometheusEndpoint)
	}
}
