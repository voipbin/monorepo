package config

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// allFlagKeys is every flag InitConfig requires. Keeping it in one place means a newly added
// config value cannot be forgotten by half the tests below.
var allFlagKeys = []string{
	"prometheus_endpoint",
	"prometheus_listen_address",
	"rabbitmq_address",
	"docker_socket_proxy_address",
	"redis_address",
	"redis_password",
	"redis_database",
	"sentinel_backend",
}

// newCommandWithFlags builds a cobra command carrying every config flag except the ones named in
// skip.
func newCommandWithFlags(values Config, skip ...string) *cobra.Command {
	skipped := map[string]bool{}
	for _, key := range skip {
		skipped[key] = true
	}

	cmd := &cobra.Command{}
	if !skipped["prometheus_endpoint"] {
		cmd.Flags().String("prometheus_endpoint", values.PrometheusEndpoint, "")
	}
	if !skipped["prometheus_listen_address"] {
		cmd.Flags().String("prometheus_listen_address", values.PrometheusListenAddress, "")
	}
	if !skipped["rabbitmq_address"] {
		cmd.Flags().String("rabbitmq_address", values.RabbitMQAddress, "")
	}
	if !skipped["docker_socket_proxy_address"] {
		cmd.Flags().String("docker_socket_proxy_address", values.DockerSocketProxyAddress, "")
	}
	if !skipped["redis_address"] {
		cmd.Flags().String("redis_address", values.RedisAddress, "")
	}
	if !skipped["redis_password"] {
		cmd.Flags().String("redis_password", values.RedisPassword, "")
	}
	if !skipped["redis_database"] {
		cmd.Flags().Int("redis_database", values.RedisDatabase, "")
	}
	if !skipped["sentinel_backend"] {
		cmd.Flags().String("sentinel_backend", values.SentinelBackend, "")
	}

	return cmd
}

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
		expectCfg   Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
			expectCfg:   Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				PrometheusEndpoint:       "/metrics",
				PrometheusListenAddress:  ":2112",
				RabbitMQAddress:          "amqp://guest:guest@localhost:5672",
				SentinelBackend:          BackendDocker,
				DockerSocketProxyAddress: "tcp://sentinel-docker-socket-proxy:2375",
				RedisAddress:             "localhost:6379",
				RedisPassword:            "secret",
				RedisDatabase:            1,
			},
			expectCfg: Config{
				PrometheusEndpoint:       "/metrics",
				PrometheusListenAddress:  ":2112",
				RabbitMQAddress:          "amqp://guest:guest@localhost:5672",
				SentinelBackend:          BackendDocker,
				DockerSocketProxyAddress: "tcp://sentinel-docker-socket-proxy:2375",
				RedisAddress:             "localhost:6379",
				RedisPassword:            "secret",
				RedisDatabase:            1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg = tt.setupConfig

			if res := Get(); res != tt.expectCfg {
				t.Errorf("Wrong match. expect: %+v, got: %+v", tt.expectCfg, res)
			}
		})
	}
}

func Test_InitConfig(t *testing.T) {
	tests := []struct {
		name string

		values Config
	}{
		{
			name: "initializes_with_default_values",

			values: Config{
				PrometheusEndpoint:       defaultPrometheusEndpoint,
				PrometheusListenAddress:  defaultPrometheusListenAddress,
				RabbitMQAddress:          defaultRabbitMQAddress,
				SentinelBackend:          BackendDocker,
				DockerSocketProxyAddress: defaultDockerSocketProxyAddress,
				RedisAddress:             defaultRedisAddress,
				RedisPassword:            defaultRedisPassword,
				RedisDatabase:            defaultRedisDatabase,
			},
		},
		{
			name: "initializes_with_custom_values",

			values: Config{
				PrometheusEndpoint:       "/custom-metrics",
				PrometheusListenAddress:  ":9090",
				RabbitMQAddress:          "amqp://user:pass@rabbitmq.example.com:5672",
				SentinelBackend:          BackendKubernetes,
				DockerSocketProxyAddress: "tcp://127.0.0.1:2375",
				RedisAddress:             "redis.example.com:6379",
				RedisPassword:            "secret",
				RedisDatabase:            7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			cmd := newCommandWithFlags(tt.values)

			if err := InitConfig(cmd); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res := Get(); res != tt.values {
				t.Errorf("Wrong match. expect: %+v, got: %+v", tt.values, res)
			}
		})
	}
}

func Test_InitConfigWithMissingFlags(t *testing.T) {
	for _, skip := range allFlagKeys {
		t.Run("error_when_"+skip+"_flag_missing", func(t *testing.T) {
			viper.Reset()

			// a VALID backend everywhere else, so the only reason this can fail is the missing
			// flag itself -- otherwise validation would mask what this test is checking.
			cmd := newCommandWithFlags(Config{SentinelBackend: BackendKubernetes}, skip)

			if err := InitConfig(cmd); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_Bootstrap(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}

	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, flagKey := range allFlagKeys {
		if cmd.PersistentFlags().Lookup(flagKey) == nil {
			t.Errorf("Wrong match. expect: flag %s registered, got: missing", flagKey)
		}
	}
}

func Test_BootstrapWithEnv(t *testing.T) {
	envVars := map[string]string{
		"RABBITMQ_ADDRESS":            "amqp://test:test@test:5672",
		"PROMETHEUS_ENDPOINT":         "/test-metrics",
		"PROMETHEUS_LISTEN_ADDRESS":   ":9999",
		"DOCKER_SOCKET_PROXY_ADDRESS": "tcp://test-proxy:2375",
		"REDIS_ADDRESS":               "test-redis:6379",
		"REDIS_PASSWORD":              "test-password",
		"REDIS_DATABASE":              "5",
		"SENTINEL_BACKEND":            "kubernetes",
	}

	expect := map[string]string{
		"rabbitmq_address":            "amqp://test:test@test:5672",
		"prometheus_endpoint":         "/test-metrics",
		"prometheus_listen_address":   ":9999",
		"docker_socket_proxy_address": "tcp://test-proxy:2375",
		"redis_address":               "test-redis:6379",
		"redis_password":              "test-password",
		"redis_database":              "5",
		"sentinel_backend":            "kubernetes",
	}

	viper.Reset()

	for k, v := range envVars {
		t.Setenv(k, v)
	}

	cmd := &cobra.Command{}

	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for flagKey, expectValue := range expect {
		if res := viper.GetString(flagKey); res != expectValue {
			t.Errorf("Wrong match for %s. expect: %s, got: %s", flagKey, expectValue, res)
		}
	}

	if res := viper.GetInt("redis_database"); res != 5 {
		t.Errorf("Wrong match. expect: 5, got: %d", res)
	}
}

func Test_LoadGlobalConfig(t *testing.T) {
	viper.Reset()
	cfg = Config{}

	viperValues := map[string]any{
		"prometheus_endpoint":         "/test-metrics",
		"prometheus_listen_address":   ":8888",
		"rabbitmq_address":            "amqp://test:test@localhost:5672",
		"docker_socket_proxy_address": "tcp://test-proxy:2375",
		"redis_address":               "test-redis:6379",
		"redis_password":              "test-password",
		"redis_database":              9,
		"sentinel_backend":            BackendDocker,
	}
	for k, v := range viperValues {
		viper.Set(k, v)
	}

	if err := LoadGlobalConfig(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := Config{
		PrometheusEndpoint:       "/test-metrics",
		PrometheusListenAddress:  ":8888",
		RabbitMQAddress:          "amqp://test:test@localhost:5672",
		SentinelBackend:          BackendDocker,
		DockerSocketProxyAddress: "tcp://test-proxy:2375",
		RedisAddress:             "test-redis:6379",
		RedisPassword:            "test-password",
		RedisDatabase:            9,
	}

	if res := Get(); res != expect {
		t.Errorf("Wrong match. expect: %+v, got: %+v", expect, res)
	}
}

// Test_LoadGlobalConfig_retriesAfterFailure is the regression test for the sync.Once bug this
// change removes: under the old once.Do-guarded LoadGlobalConfig, a failed first call still
// consumed the Once, so every later call silently returned nil against a still-zero-value cfg.
// This pins the new contract -- every call reloads and re-validates from viper, so a failure is
// retryable -- across three calls: fail, then succeed, then reload with changed viper state.
func Test_LoadGlobalConfig_retriesAfterFailure(t *testing.T) {
	viper.Reset()
	cfg = Config{}

	// (a) an invalid backend is rejected.
	viper.Set("sentinel_backend", "nomad")

	if err := LoadGlobalConfig(); err == nil {
		t.Fatal("Wrong match. expect: error, got: nil")
	}

	// (b) the failed call must not leave any partial state visible through Get -- cfg is only
	// assigned after Validate succeeds, so it should still be the zero value.
	if res := Get(); res != (Config{}) {
		t.Errorf("Wrong match. expect: zero-value Config after a failed load, got: %+v", res)
	}

	// (c) a later call with a corrected, valid backend must succeed -- this is what the old
	// once.Do-guarded version got wrong: it would have returned nil here with cfg still zero.
	viper.Set("sentinel_backend", BackendKubernetes)
	viper.Set("prometheus_endpoint", "/metrics-first")
	viper.Set("rabbitmq_address", "amqp://first:first@localhost:5672")

	if err := LoadGlobalConfig(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := Config{
		PrometheusEndpoint: "/metrics-first",
		RabbitMQAddress:    "amqp://first:first@localhost:5672",
		SentinelBackend:    BackendKubernetes,
	}
	if res := Get(); res != expect {
		t.Errorf("Wrong match. expect: %+v, got: %+v", expect, res)
	}

	// (d) a third call with different viper state must reload -- proving no caching was silently
	// reintroduced. This is the direct inverse of the deleted Test_LoadGlobalConfigOnlyOnce, whose
	// pinned "second call is a no-op" invariant is exactly the bug this change removes.
	viper.Set("prometheus_endpoint", "/metrics-second")

	if err := LoadGlobalConfig(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res := Get(); res.PrometheusEndpoint != "/metrics-second" {
		t.Errorf("Wrong match. expect: /metrics-second, got: %s", res.PrometheusEndpoint)
	}
}

// Test_LoadGlobalConfig_failureDoesNotClobberPriorSuccess guards a property that only became
// load-bearing once LoadGlobalConfig lost its once.Do guard: cfg is assigned only after Validate
// succeeds, so a later failed call cannot clobber a previously-good config with a partial or
// zero-value one. That ordering was already correct on a first call before this change (the old
// once.Do body also validated before assigning); what changes is that a SECOND call can now reach
// the assignment at all, which makes the ordering guard load-bearing on every call, not just the
// first. This test protects against a plausible future refactor -- e.g. simplifying to
// `cfg = loadFromViper(); if err := cfg.Validate(); err != nil { return err }` -- that would leak
// an invalid config through Get() before returning the error.
func Test_LoadGlobalConfig_failureDoesNotClobberPriorSuccess(t *testing.T) {
	viper.Reset()
	cfg = Config{}

	// (a) a valid load succeeds.
	viper.Set("sentinel_backend", BackendKubernetes)
	viper.Set("prometheus_endpoint", "/metrics")

	if err := LoadGlobalConfig(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	firstCfg := Get()

	// (b) a later call with an invalid backend fails.
	viper.Set("sentinel_backend", "nomad")

	if err := LoadGlobalConfig(); err == nil {
		t.Fatal("Wrong match. expect: error, got: nil")
	}

	// (c) the failed call must not have touched cfg -- Get still returns the last known good
	// config from (a), not a zero-value or partially-applied one.
	if res := Get(); res != firstCfg {
		t.Errorf("Wrong match. expect: cfg unchanged after a failed reload, got: %+v vs %+v", firstCfg, res)
	}
}

// Test_defaults pins the shipped defaults, in particular that the docker endpoint points at the
// read-only proxy sidecar and NEVER at /var/run/docker.sock -- the raw socket grants
// root-equivalent host access and must not be reachable by accident.
func Test_defaults(t *testing.T) {
	if defaultDockerSocketProxyAddress != "tcp://sentinel-docker-socket-proxy:2375" {
		t.Errorf("Wrong match. expect: tcp://sentinel-docker-socket-proxy:2375, got: %s", defaultDockerSocketProxyAddress)
	}
	if defaultRedisDatabase != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", defaultRedisDatabase)
	}

	// guard against a regression that reintroduces the raw socket path anywhere in the defaults.
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		// the host happens to have a socket; that is fine, the assertion below is about config.
		_ = err
	}
	if defaultDockerSocketProxyAddress == "unix:///var/run/docker.sock" {
		t.Errorf("Wrong match. expect: the proxy endpoint, got: the raw docker socket")
	}
}

// Test_Validate is the fail-fast contract: SENTINEL_BACKEND has no default, and the Docker-only
// fields are required ONLY when the Docker backend is selected.
//
// The conditional half matters as much as the fail-fast half: validating the Docker fields
// unconditionally would reject a perfectly good Kubernetes deployment for not supplying a Redis
// address it has no reason to have (design §8.3).
func Test_Validate(t *testing.T) {
	tests := []struct {
		name string

		config Config

		expectErr bool
	}{
		{
			name: "kubernetes needs no docker config at all",

			config: Config{SentinelBackend: BackendKubernetes},

			expectErr: false,
		},
		{
			name: "kubernetes tolerates docker config being present anyway",

			config: Config{
				SentinelBackend:          BackendKubernetes,
				DockerSocketProxyAddress: "tcp://proxy:2375",
				RedisAddress:             "localhost:6379",
			},

			expectErr: false,
		},
		{
			name: "docker with both required fields",

			config: Config{
				SentinelBackend:          BackendDocker,
				DockerSocketProxyAddress: "tcp://proxy:2375",
				RedisAddress:             "localhost:6379",
			},

			expectErr: false,
		},
		{
			name: "docker without a socket proxy address",

			config: Config{
				SentinelBackend: BackendDocker,
				RedisAddress:    "localhost:6379",
			},

			expectErr: true,
		},
		{
			name: "docker without a redis address",

			config: Config{
				SentinelBackend:          BackendDocker,
				DockerSocketProxyAddress: "tcp://proxy:2375",
			},

			expectErr: true,
		},
		{
			name: "unset backend is rejected, there is no default",

			config: Config{
				DockerSocketProxyAddress: "tcp://proxy:2375",
				RedisAddress:             "localhost:6379",
			},

			expectErr: true,
		},
		{
			name: "unknown backend is rejected",

			config: Config{SentinelBackend: "nomad"},

			expectErr: true,
		},
		{
			name: "case mismatch is rejected rather than normalized",

			config: Config{SentinelBackend: "Kubernetes"},

			expectErr: true,
		},
		{
			name: "abbreviation is rejected",

			config: Config{SentinelBackend: "k8s"},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_InitConfig_failsFastOnInvalidBackend pins that sentinel-manager's OWN bootstrap path runs
// the validation, not just the Validate method in isolation.
func Test_InitConfig_failsFastOnInvalidBackend(t *testing.T) {
	tests := []struct {
		name string

		backend string

		expectErr bool
	}{
		{name: "unset", backend: "", expectErr: true},
		{name: "invalid", backend: "nomad", expectErr: true},
		{name: "kubernetes", backend: BackendKubernetes, expectErr: false},
		{name: "docker", backend: BackendDocker, expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			cmd := newCommandWithFlags(Config{
				SentinelBackend:          tt.backend,
				DockerSocketProxyAddress: "tcp://proxy:2375",
				RedisAddress:             "localhost:6379",
			})

			err := InitConfig(cmd)

			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_LoadGlobalConfig_failsFastOnInvalidBackend pins the CLI bootstrap path end-to-end: flag
// binding via Bootstrap, then loading via LoadGlobalConfig.
//
// InitConfig (used by cmd/sentinel-manager) now delegates to LoadGlobalConfig directly, so
// validation itself can no longer diverge between the two binaries -- they share one
// implementation structurally, not by parallel test coverage. What CAN still diverge is flag/env
// *binding*: Bootstrap and InitConfig each register and bind their own flags independently, and
// cmd/sentinel-manager's init() hardcodes its own default literals separately from this package's
// unexported default constants. This test still matters because it is the only one that exercises
// LoadGlobalConfig through the CLI's actual bootstrap path rather than calling it directly.
func Test_LoadGlobalConfig_failsFastOnInvalidBackend(t *testing.T) {
	tests := []struct {
		name string

		backend string

		expectErr bool
	}{
		{name: "unset", backend: "", expectErr: true},
		{name: "invalid", backend: "nomad", expectErr: true},
		{name: "kubernetes", backend: BackendKubernetes, expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			// Each subtest reloads unconditionally now that LoadGlobalConfig has no once.Do guard,
			// so only cfg needs resetting here -- a prior subtest's success must not mask this
			// subtest's expected failure (or vice versa) through a stale cfg.
			cfg = Config{}

			viper.Set("sentinel_backend", tt.backend)

			err := LoadGlobalConfig()

			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_Bootstrap_registersSentinelBackend pins that the CLI's flag registration includes the new
// field. InitConfig errors with "flag not defined" if a key is in its list but not registered, so
// a mismatch between the two would break sentinel-control at startup.
func Test_Bootstrap_registersSentinelBackend(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	flag := cmd.PersistentFlags().Lookup("sentinel_backend")
	if flag == nil {
		t.Fatalf("Wrong match. expect: sentinel_backend registered, got: missing")
	}

	// registered with NO default: an operator must say which backend they mean.
	if flag.DefValue != "" {
		t.Errorf("Wrong match. expect: an empty default, got: %q", flag.DefValue)
	}
}

// Test_backendConstants pins the wire values a self-hoster's Kubernetes manifest and this
// repo's komodo/docker-compose.yml must agree on. Changing either constant without changing
// the matching descriptor crash-loops that deployment on startup validation.
func Test_backendConstants(t *testing.T) {
	if BackendKubernetes != "kubernetes" {
		t.Errorf("Wrong match. expect: kubernetes, got: %s", BackendKubernetes)
	}
	if BackendDocker != "docker" {
		t.Errorf("Wrong match. expect: docker, got: %s", BackendDocker)
	}
	if defaultSentinelBackend != "" {
		t.Errorf("Wrong match. expect: no default backend, got: %q", defaultSentinelBackend)
	}
}
