package config

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configEnvKeys = []string{
	"PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS", "RABBITMQ_ADDRESS", "GCP_PROJECT_ID", "GCP_REGION",
	"GOOGLE_EMBEDDING_MODEL", "RAG_TOP_K", "GCP_BUCKET_NAME_MEDIA", "POSTGRESQL_DSN",
}

// resetConfigState isolates a test from the caller's environment and from the package globals
// (viper, the once guard and globalConfig), and restores everything afterwards.
func resetConfigState(t *testing.T) {
	t.Helper()

	saved := map[string]*string{}
	for _, k := range configEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			v := v
			saved[k] = &v
		} else {
			saved[k] = nil
		}
		_ = os.Unsetenv(k)
	}
	savedConfig := globalConfig

	viper.Reset()
	once = sync.Once{}
	globalConfig = Config{}

	t.Cleanup(func() {
		for k, v := range saved {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
		viper.Reset()
		once = sync.Once{}
		globalConfig = savedConfig
	})
}

func Test_Config_Validate(t *testing.T) {
	valid := Config{RabbitMQAddress: "amqp://rabbit.invalid", GoogleCloudProject: "proj", GoogleCloudRegion: "region", PostgreSQLDSN: "dsn"}

	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{name: "all required values set", mutate: func(c *Config) {}},
		{name: "missing rabbitmq address", mutate: func(c *Config) { c.RabbitMQAddress = "" }, wantErr: "RABBITMQ_ADDRESS is required"},
		{name: "missing project", mutate: func(c *Config) { c.GoogleCloudProject = "" }, wantErr: "GCP_PROJECT_ID is required"},
		{name: "missing region", mutate: func(c *Config) { c.GoogleCloudRegion = "" }, wantErr: "GCP_REGION is required"},
		{name: "missing dsn", mutate: func(c *Config) { c.PostgreSQLDSN = "" }, wantErr: "POSTGRESQL_DSN is required"},
		{name: "optional values may be empty", mutate: func(c *Config) { c.GCPBucketNameMedia = ""; c.GoogleEmbeddingModel = ""; c.RAGTopK = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid
			tt.mutate(&c)

			err := c.Validate()
			if tt.wantErr == "" && err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func Test_Get_returns_the_global_config(t *testing.T) {
	resetConfigState(t)

	globalConfig = Config{PostgreSQLDSN: "marker"}
	if got := Get(); got != &globalConfig || got.PostgreSQLDSN != "marker" {
		t.Errorf("Wrong match. expect the global config pointer, got: %+v", got)
	}
}

func Test_Bootstrap_and_LoadGlobalConfig_defaults(t *testing.T) {
	resetConfigState(t)

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}
	LoadGlobalConfig()

	// The default rabbitmq address embeds a credential-shaped literal; check its shape instead of repeating it.
	rabbit := Get().RabbitMQAddress
	if !strings.HasPrefix(rabbit, "amqp://") || !strings.HasSuffix(rabbit, "@localhost:5672") {
		t.Errorf("Wrong match. unexpected default rabbitmq address shape: %d chars", len(rabbit))
	}

	want := Config{
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":2112",
		RabbitMQAddress:         rabbit,
		GoogleEmbeddingModel:    "text-embedding-004",
		RAGTopK:                 5,
	}
	if *Get() != want {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", want, *Get())
	}

	for _, name := range []string{"prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address", "gcp_project_id", "gcp_region", "google_embedding_model", "rag_top_k", "gcp_bucket_name_media", "postgresql_dsn"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect flag %s to be registered", name)
		}
	}
}

func Test_Bootstrap_environment_overrides_defaults(t *testing.T) {
	resetConfigState(t)

	t.Setenv("PROMETHEUS_ENDPOINT", "/m")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.invalid")
	t.Setenv("GCP_PROJECT_ID", "proj")
	t.Setenv("GCP_REGION", "region")
	t.Setenv("GOOGLE_EMBEDDING_MODEL", "model-x")
	t.Setenv("RAG_TOP_K", "12")
	t.Setenv("GCP_BUCKET_NAME_MEDIA", "bucket")
	t.Setenv("POSTGRESQL_DSN", "dsn")

	if err := Bootstrap(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}
	LoadGlobalConfig()

	want := Config{
		PrometheusEndpoint:      "/m",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "amqp://rabbit.invalid",
		GoogleCloudProject:      "proj",
		GoogleCloudRegion:       "region",
		GoogleEmbeddingModel:    "model-x",
		RAGTopK:                 12,
		GCPBucketNameMedia:      "bucket",
		PostgreSQLDSN:           "dsn",
	}
	if *Get() != want {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", want, *Get())
	}
	if err := Get().Validate(); err != nil {
		t.Errorf("Wrong match. a fully configured service must validate, got: %v", err)
	}
}

func Test_LoadGlobalConfig_runs_only_once(t *testing.T) {
	resetConfigState(t)

	t.Setenv("GCP_PROJECT_ID", "first")
	if err := Bootstrap(&cobra.Command{Use: "test"}); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}
	LoadGlobalConfig()

	t.Setenv("GCP_PROJECT_ID", "second")
	LoadGlobalConfig()

	if got := Get().GoogleCloudProject; got != "first" {
		t.Errorf("Wrong match. the second load must be ignored, expect: first, got: %s", got)
	}
}

// newInitCommand returns a command whose flags are visible through cmd.Flags(), which is what
// InitConfig reads.
func newInitCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("could not bootstrap: %v", err)
	}
	cmd.Flags().AddFlagSet(cmd.PersistentFlags())
	return cmd
}

func Test_InitConfig_maps_every_setting_from_the_environment(t *testing.T) {
	resetConfigState(t)

	t.Setenv("PROMETHEUS_ENDPOINT", "/m")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", ":9999")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://rabbit.invalid")
	t.Setenv("GCP_PROJECT_ID", "proj")
	t.Setenv("GCP_REGION", "region")
	t.Setenv("GOOGLE_EMBEDDING_MODEL", "model-x")
	t.Setenv("RAG_TOP_K", "8")
	t.Setenv("GCP_BUCKET_NAME_MEDIA", "bucket")
	t.Setenv("POSTGRESQL_DSN", "dsn-from-env")

	if err := InitConfig(newInitCommand(t)); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}

	want := Config{
		PrometheusEndpoint:      "/m",
		PrometheusListenAddress: ":9999",
		RabbitMQAddress:         "amqp://rabbit.invalid",
		GoogleCloudProject:      "proj",
		GoogleCloudRegion:       "region",
		GoogleEmbeddingModel:    "model-x",
		RAGTopK:                 8,
		GCPBucketNameMedia:      "bucket",
		PostgreSQLDSN:           "dsn-from-env",
	}
	if *Get() != want {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", want, *Get())
	}
}

func Test_InitConfig_uses_flag_defaults_without_environment(t *testing.T) {
	resetConfigState(t)

	if err := InitConfig(newInitCommand(t)); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}

	got := Get()
	if got.RAGTopK != 5 || got.GoogleEmbeddingModel != "text-embedding-004" || got.PrometheusEndpoint != "/metrics" || got.PrometheusListenAddress != ":2112" {
		t.Errorf("Wrong match. unexpected defaults: %+v", *got)
	}
	if got.PostgreSQLDSN != "" || got.GoogleCloudProject != "" || got.GCPBucketNameMedia != "" {
		t.Errorf("Wrong match. unset settings must stay empty, got: %+v", *got)
	}
}

func Test_InitConfig_flag_value_is_used_when_no_environment_is_set(t *testing.T) {
	resetConfigState(t)

	cmd := newInitCommand(t)
	if err := cmd.Flags().Set("gcp_region", "from-flag"); err != nil {
		t.Fatalf("could not set flag: %v", err)
	}
	if err := InitConfig(cmd); err != nil {
		t.Fatalf("Wrong match. expect ok, got: %v", err)
	}

	if got := Get().GoogleCloudRegion; got != "from-flag" {
		t.Errorf("Wrong match. expect: from-flag, got: %s", got)
	}
}

func Test_InitConfig_missing_flag_is_reported_by_name(t *testing.T) {
	flags := []string{"prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address", "gcp_project_id", "gcp_region", "google_embedding_model", "rag_top_k", "gcp_bucket_name_media", "postgresql_dsn"}

	for _, missing := range flags {
		t.Run(missing, func(t *testing.T) {
			resetConfigState(t)

			// register every flag except the missing one
			cmd := &cobra.Command{Use: "test"}
			for _, name := range flags {
				if name == missing {
					continue
				}
				if name == "rag_top_k" {
					cmd.Flags().Int(name, 5, "")
					continue
				}
				cmd.Flags().String(name, "", "")
			}

			err := InitConfig(cmd)
			if err == nil || !strings.Contains(err.Error(), "error binding "+missing+" flag") {
				t.Errorf("Wrong match. expect an error naming %s, got: %v", missing, err)
			}
		})
	}
}
