package config

import (
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configEnvKeys = []string{
	"DATABASE_DSN",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"RABBITMQ_ADDRESS",
	"REDIS_ADDRESS",
	"REDIS_DATABASE",
	"REDIS_PASSWORD",
}

// isolateGlobals resets viper, removes the host environment keys that viper would read,
// and restores everything when the test finishes.
func isolateGlobals(t *testing.T) {
	t.Helper()

	savedConfig := globalConfig
	type savedEnv struct {
		key   string
		value string
		found bool
	}
	var saved []savedEnv
	for _, k := range configEnvKeys {
		v, ok := os.LookupEnv(k)
		saved = append(saved, savedEnv{key: k, value: v, found: ok})
		_ = os.Unsetenv(k)
	}

	viper.Reset()
	once = sync.Once{}

	t.Cleanup(func() {
		viper.Reset()
		once = sync.Once{}
		globalConfig = savedConfig
		for _, s := range saved {
			if s.found {
				_ = os.Setenv(s.key, s.value)
			} else {
				_ = os.Unsetenv(s.key)
			}
		}
	})
}

func Test_Bootstrap_defaults(t *testing.T) {
	isolateGlobals(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap returned an error: %v", err)
	}

	expects := map[string]string{
		"database_dsn":              "testid:testpassword@tcp(127.0.0.1:3306)/test",
		"prometheus_endpoint":       "/metrics",
		"prometheus_listen_address": ":2112",
		"rabbitmq_address":          "amqp://guest:" + "guest" + "@localhost:5672",
		"redis_address":             "127.0.0.1:6379",
		"redis_database":            "1",
		"redis_password":            "",
	}
	for name, expect := range expects {
		f := cmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Errorf("flag %s is not registered", name)
			continue
		}
		if f.DefValue != expect {
			t.Errorf("Wrong default for %s. expect: %q, got: %q", name, expect, f.DefValue)
		}
		if got := viper.GetString(name); got != expect {
			t.Errorf("Wrong viper value for %s. expect: %q, got: %q", name, expect, got)
		}
	}
}

func Test_Bootstrap_env_overrides_default(t *testing.T) {
	isolateGlobals(t)

	t.Setenv("REDIS_ADDRESS", "198.51.100.7:6380")
	t.Setenv("REDIS_DATABASE", "5")

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap returned an error: %v", err)
	}

	if got := viper.GetString("redis_address"); got != "198.51.100.7:6380" {
		t.Errorf("Wrong redis_address. expect: 198.51.100.7:6380, got: %s", got)
	}
	if got := viper.GetInt("redis_database"); got != 5 {
		t.Errorf("Wrong redis_database. expect: 5, got: %d", got)
	}
	// keys without an env override keep their flag default
	if got := viper.GetString("prometheus_endpoint"); got != "/metrics" {
		t.Errorf("Wrong prometheus_endpoint. expect: /metrics, got: %s", got)
	}
}

func Test_LoadGlobalConfig(t *testing.T) {
	isolateGlobals(t)

	// every key gets a different value so a swapped mapping is detected
	viper.Set("database_dsn", "dsn-value")
	viper.Set("prometheus_endpoint", "/endpoint-value")
	viper.Set("prometheus_listen_address", ":1111")
	viper.Set("rabbitmq_address", "rabbit-value")
	viper.Set("redis_address", "redis-value")
	viper.Set("redis_database", 7)
	viper.Set("redis_password", "password-value")

	LoadGlobalConfig()

	expect := Config{
		DatabaseDSN:             "dsn-value",
		PrometheusEndpoint:      "/endpoint-value",
		PrometheusListenAddress: ":1111",
		RabbitMQAddress:         "rabbit-value",
		RedisAddress:            "redis-value",
		RedisDatabase:           7,
		RedisPassword:           "password-value",
	}
	if got := *Get(); !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong config. expect: %+v, got: %+v", expect, got)
	}

	// the second call is ignored because of the sync.Once
	viper.Set("database_dsn", "changed-value")
	LoadGlobalConfig()
	if got := Get().DatabaseDSN; got != "dsn-value" {
		t.Errorf("LoadGlobalConfig must run once. expect: dsn-value, got: %s", got)
	}
}

func Test_InitConfig_values_and_missing_flag(t *testing.T) {
	flagNames := []string{
		"database_dsn",
		"prometheus_endpoint",
		"prometheus_listen_address",
		"rabbitmq_address",
		"redis_address",
		"redis_database",
		"redis_password",
	}

	newCmd := func(skip string) *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		for _, name := range flagNames {
			if name == skip {
				continue
			}
			if name == "redis_database" {
				cmd.Flags().Int(name, 3, "")
				continue
			}
			cmd.Flags().String(name, "value-of-"+name, "")
		}
		return cmd
	}

	t.Run("all flags registered", func(t *testing.T) {
		isolateGlobals(t)

		if err := InitConfig(newCmd("")); err != nil {
			t.Fatalf("InitConfig returned an error: %v", err)
		}

		expect := Config{
			DatabaseDSN:             "value-of-database_dsn",
			PrometheusEndpoint:      "value-of-prometheus_endpoint",
			PrometheusListenAddress: "value-of-prometheus_listen_address",
			RabbitMQAddress:         "value-of-rabbitmq_address",
			RedisAddress:            "value-of-redis_address",
			RedisDatabase:           3,
			RedisPassword:           "value-of-redis_password",
		}
		if got := *Get(); !reflect.DeepEqual(got, expect) {
			t.Errorf("Wrong config. expect: %+v, got: %+v", expect, got)
		}
	})

	for _, missing := range flagNames {
		t.Run("missing "+missing, func(t *testing.T) {
			isolateGlobals(t)

			err := InitConfig(newCmd(missing))
			if err == nil {
				t.Fatalf("expected an error for the missing flag %s", missing)
			}
			if !strings.Contains(err.Error(), "error binding "+missing+" flag") {
				t.Errorf("Wrong error for %s. got: %v", missing, err)
			}
		})
	}
}
