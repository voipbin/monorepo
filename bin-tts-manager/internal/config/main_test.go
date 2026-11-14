package config

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		setupConfig Config
	}{
		{
			name: "returns_default_config",

			setupConfig: Config{},
		},
		{
			name: "returns_configured_values",

			setupConfig: Config{
				RabbitMQAddress:         "amqp://guest:guest@localhost:5672",
				PrometheusEndpoint:      "/metrics",
				PrometheusListenAddress: ":2112",
				GCPTTSEndpoint:          "texttospeech.googleapis.com:443",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalConfig = tt.setupConfig

			res := Get()

			if res.RabbitMQAddress != tt.setupConfig.RabbitMQAddress {
				t.Errorf("Wrong RabbitMQAddress. expect: %s, got: %s", tt.setupConfig.RabbitMQAddress, res.RabbitMQAddress)
			}
			if res.PrometheusEndpoint != tt.setupConfig.PrometheusEndpoint {
				t.Errorf("Wrong PrometheusEndpoint. expect: %s, got: %s", tt.setupConfig.PrometheusEndpoint, res.PrometheusEndpoint)
			}
			if res.PrometheusListenAddress != tt.setupConfig.PrometheusListenAddress {
				t.Errorf("Wrong PrometheusListenAddress. expect: %s, got: %s", tt.setupConfig.PrometheusListenAddress, res.PrometheusListenAddress)
			}
			if res.GCPTTSEndpoint != tt.setupConfig.GCPTTSEndpoint {
				t.Errorf("Wrong GCPTTSEndpoint. expect: %s, got: %s", tt.setupConfig.GCPTTSEndpoint, res.GCPTTSEndpoint)
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name string

		expectErr bool
	}{
		{
			name: "initializes_with_default_values",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{}

			err := Bootstrap(rootCmd)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify flags were registered
			if rootCmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
				t.Errorf("Expected rabbitmq_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
				t.Errorf("Expected prometheus_endpoint flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
				t.Errorf("Expected prometheus_listen_address flag to be registered")
			}
			if rootCmd.PersistentFlags().Lookup("gcp_tts_endpoint") == nil {
				t.Errorf("Expected gcp_tts_endpoint flag to be registered")
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "loads_global_config_only_once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset once for test isolation
			once = sync.Once{}

			// First call should work without panic
			LoadGlobalConfig()

			// Second call should be a no-op due to sync.Once
			LoadGlobalConfig()

			// Verify Get returns a valid pointer
			cfg := Get()
			if cfg == nil {
				t.Errorf("Expected non-nil config from Get()")
			}
		})
	}
}

// The provider keys are read from the environment only: Config has no field for them, and no flag
// of the old names exists.
func Test_removedProviderFields(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for _, name := range []string{"AWSAccessKey", "AWSSecretKey", "ElevenlabsAPIKey"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("Wrong match. Config must not have the field %s", name)
		}
	}
	if _, ok := reflect.TypeOf(&Config{}).MethodByName("EnvValue"); ok {
		t.Errorf("Wrong match. Config must not have the method EnvValue")
	}

	cmd := &cobra.Command{}
	viper.Reset()
	ResetForTest()
	t.Cleanup(func() {
		viper.Reset()
		ResetForTest()
	})
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for _, name := range []string{"aws_access_key", "aws_secret_key", "elevenlabs_api_key"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. the flag %q must not exist", name)
		}
	}
}

// A deployment that still exports the old variables starts normally: the load ignores them and
// the other fields are not affected.
func Test_oldProviderVariablesIgnored(t *testing.T) {
	viper.Reset()
	ResetForTest()
	t.Cleanup(func() {
		viper.Reset()
		ResetForTest()
	})

	t.Setenv("AWS_ACCESS_KEY", "synthetic-old-access")
	t.Setenv("AWS_SECRET_KEY", "synthetic-old-secret")
	t.Setenv("ELEVENLABS_API_KEY", "synthetic-old-eleven")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://synthetic-host:5672")
	t.Setenv("GCP_TTS_ENDPOINT", "synthetic-gcp:443")

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	got := Get()
	if got.RabbitMQAddress != "amqp://synthetic-host:5672" || got.GCPTTSEndpoint != "synthetic-gcp:443" {
		t.Errorf("Wrong match. the other fields are affected: %+v", got)
	}

	// no loaded field holds an old value
	v := reflect.ValueOf(*got)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() == reflect.String && strings.Contains(v.Field(i).String(), "synthetic-old-") {
			t.Errorf("Wrong match. the field %s holds an old variable value", v.Type().Field(i).Name)
		}
	}
}
