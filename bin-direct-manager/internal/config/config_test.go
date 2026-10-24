package config

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// testCredential is a synthetic value built at run time.
var (
	testCredential = strings.Repeat("c", 12)
	testDSN1       = strings.Repeat("d", 8)
	testDSN2       = strings.Repeat("e", 8)
)

var envKeys = []string{
	"DATABASE_DSN",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"RABBITMQ_ADDRESS",
	"REDIS_ADDRESS",
	"REDIS_DATABASE",
	"REDIS_PASSWORD",
}

// resetConfigState isolates a test from the process wide viper and config
// state. Tests in this package must not run in parallel.
func resetConfigState(t *testing.T) {
	t.Helper()

	prev := globalConfig
	viper.Reset()
	for _, k := range envKeys {
		t.Setenv(k, "")
	}

	t.Cleanup(func() {
		viper.Reset()
		globalConfig = prev
	})
}

// registerFlags registers the flags that InitConfig reads, except for the
// ones listed in skip. It mirrors cmd/direct-manager, where redis_database is
// an int flag.
func registerFlags(cmd *cobra.Command, skip ...string) {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}

	strFlags := map[string]string{
		"database_dsn":              testDSN1,
		"prometheus_endpoint":       "/m",
		"prometheus_listen_address": ":9999",
		"rabbitmq_address":          "mq.example.invalid:5672",
		"redis_address":             "cache.example.invalid:6379",
	}
	for name, val := range strFlags {
		if !skipped[name] {
			cmd.Flags().String(name, val, "")
		}
	}
	if !skipped["redis_password"] {
		cmd.Flags().String("redis_password", testCredential, "")
	}
	if !skipped["redis_database"] {
		cmd.Flags().Int("redis_database", 7, "")
	}
}

func matchDefault(got, want string, suffix bool) bool {
	if suffix {
		return len(got) > len(want) && strings.Contains(got, want)
	}

	return got == want
}

func Test_Bootstrap_flags(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	// the credentials of the two connection defaults are not repeated here, so only the
	// host part is compared for them
	tests := []struct {
		name         string
		expectString string
		suffix       bool
	}{
		{"database_dsn", "127.0.0.1:3306", true},
		{"prometheus_endpoint", "/metrics", false},
		{"prometheus_listen_address", ":2112", false},
		{"rabbitmq_address", "localhost:5672", true},
		{"redis_address", "127.0.0.1:6379", false},
		{"redis_database", "1", false},
		{"redis_password", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("flag %q is not registered", tt.name)
			}
			if !matchDefault(f.DefValue, tt.expectString, tt.suffix) {
				t.Errorf("default of %q = %q, want %q (suffix match: %v)", tt.name, f.DefValue, tt.expectString, tt.suffix)
			}
			if got := viper.GetString(tt.name); !matchDefault(got, tt.expectString, tt.suffix) {
				t.Errorf("viper value of %q = %q, want %q (suffix match: %v)", tt.name, got, tt.expectString, tt.suffix)
			}
		})
	}
}

func Test_Bootstrap_envOverridesDefault(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	t.Setenv("REDIS_ADDRESS", "cache.example.invalid:6380")
	t.Setenv("REDIS_DATABASE", "5")

	if got := viper.GetString("redis_address"); got != "cache.example.invalid:6380" {
		t.Errorf("redis_address = %q, want the value of REDIS_ADDRESS", got)
	}
	if got := viper.GetInt("redis_database"); got != 5 {
		t.Errorf("redis_database = %d, want 5 from REDIS_DATABASE", got)
	}
	if got := viper.GetString("database_dsn"); !strings.Contains(got, "127.0.0.1:3306") {
		t.Errorf("an unset variable must keep the default, got a different value")
	}
}

func Test_InitConfig_fromFlags(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	registerFlags(cmd)

	if err := InitConfig(cmd); err != nil {
		t.Fatalf("InitConfig() error = %v", err)
	}

	expect := Config{
		DatabaseDSN:             testDSN1,
		PrometheusEndpoint:      "/m",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "mq.example.invalid:5672",
		RedisAddress:            "cache.example.invalid:6379",
		RedisDatabase:           7,
	}
	got := *Get()
	if got.RedisPassword != testCredential {
		t.Errorf("Get() secret setting does not match the flag value")
	}
	got.RedisPassword = ""
	if got != expect {
		t.Errorf("Get() = %+v, want %+v", got, expect)
	}
}

func Test_InitConfig_envOverridesFlag(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	registerFlags(cmd)
	t.Setenv("REDIS_DATABASE", "9")
	t.Setenv("DATABASE_DSN", testDSN2)

	if err := InitConfig(cmd); err != nil {
		t.Fatalf("InitConfig() error = %v", err)
	}

	if got := Get().RedisDatabase; got != 9 {
		t.Errorf("RedisDatabase = %d, want 9 from the environment", got)
	}
	if got := Get().DatabaseDSN; got != testDSN2 {
		t.Errorf("DatabaseDSN = %q, want the value of DATABASE_DSN", got)
	}
}

func Test_InitConfig_missingFlag(t *testing.T) {
	flagNames := []string{
		"database_dsn",
		"prometheus_endpoint",
		"prometheus_listen_address",
		"rabbitmq_address",
		"redis_address",
		"redis_database",
		"redis_password",
	}

	for _, name := range flagNames {
		t.Run(name, func(t *testing.T) {
			resetConfigState(t)

			cmd := &cobra.Command{Use: "test"}
			registerFlags(cmd, name)

			err := InitConfig(cmd)
			if err == nil {
				t.Fatalf("InitConfig() error = nil, want an error for the missing flag %q", name)
			}
			if !strings.Contains(err.Error(), name) {
				t.Errorf("InitConfig() error = %q, want it to name %q", err.Error(), name)
			}
		})
	}
}

// Test_LoadGlobalConfig loads the configuration from viper. LoadGlobalConfig
// is guarded by sync.Once, so it applies once per process: this is the only
// test that calls it and it assumes -count=1.
func Test_LoadGlobalConfig(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	t.Setenv("REDIS_PASSWORD", testCredential)
	t.Setenv("REDIS_DATABASE", "4")
	t.Setenv("REDIS_ADDRESS", "cache.example.invalid:6381")

	LoadGlobalConfig()

	got := *Get()
	if got.RedisAddress != "cache.example.invalid:6381" || got.RedisDatabase != 4 || got.RedisPassword != testCredential {
		t.Errorf("Get() = %+v, want the environment values for the redis settings", got)
	}
	if got.PrometheusListenAddress != ":2112" || got.PrometheusEndpoint != "/metrics" {
		t.Errorf("Get() = %+v, want the prometheus defaults", got)
	}

	// a second load does not apply again
	t.Setenv("REDIS_ADDRESS", "cache.example.invalid:6382")
	LoadGlobalConfig()
	if Get().RedisAddress != "cache.example.invalid:6381" {
		t.Errorf("LoadGlobalConfig() applied twice, RedisAddress = %q", Get().RedisAddress)
	}
}
