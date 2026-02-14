package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DatabaseDSN:             "user:pass@tcp(localhost:3306)/db",
		PrometheusEndpoint:      "/metrics",
		PrometheusListenAddress: ":8080",
		RabbitMQAddress:         "amqp://localhost:5672",
		SSLPrivkeyBase64:        "dGVzdA==",
		SSLCertBase64:           "dGVzdA==",
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"DatabaseDSN", cfg.DatabaseDSN, "user:pass@tcp(localhost:3306)/db"},
		{"PrometheusEndpoint", cfg.PrometheusEndpoint, "/metrics"},
		{"PrometheusListenAddress", cfg.PrometheusListenAddress, ":8080"},
		{"RabbitMQAddress", cfg.RabbitMQAddress, "amqp://localhost:5672"},
		{"SSLPrivkeyBase64", cfg.SSLPrivkeyBase64, "dGVzdA=="},
		{"SSLCertBase64", cfg.SSLCertBase64, "dGVzdA=="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Config.%s = %v, expected %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *cobra.Command
		wantErr   bool
	}{
		{
			name: "successful bootstrap",
			setupFunc: func() *cobra.Command {
				cmd := &cobra.Command{}
				return cmd
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for clean test
			viper.Reset()

			cmd := tt.setupFunc()
			err := Bootstrap(cmd)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify flags were created
				if cmd.PersistentFlags().Lookup("database_dsn") == nil {
					t.Error("database_dsn flag not created")
				}
				if cmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
					t.Error("rabbitmq_address flag not created")
				}
				if cmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
					t.Error("prometheus_endpoint flag not created")
				}
				if cmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
					t.Error("prometheus_listen_address flag not created")
				}
				if cmd.PersistentFlags().Lookup("ssl_privkey_base64") == nil {
					t.Error("ssl_privkey_base64 flag not created")
				}
				if cmd.PersistentFlags().Lookup("ssl_cert_base64") == nil {
					t.Error("ssl_cert_base64 flag not created")
				}
			}
		})
	}
}

func TestBindConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *cobra.Command
		wantErr   bool
	}{
		{
			name: "successful bind",
			setupFunc: func() *cobra.Command {
				cmd := &cobra.Command{}
				return cmd
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			cmd := tt.setupFunc()

			// Call bindConfig through Bootstrap
			err := Bootstrap(cmd)

			if (err != nil) != tt.wantErr {
				t.Errorf("bindConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	// Test InitConfig first to ensure it initializes the config
	// Due to sync.Once, this must run before LoadGlobalConfig
	t.Run("init with cobra command and all flags", func(t *testing.T) {
		viper.Reset()

		cmd := &cobra.Command{}
		cmd.Flags().String("database_dsn", "init:init@tcp(localhost:3306)/initdb", "")
		cmd.Flags().String("prometheus_endpoint", "/init-metrics", "")
		cmd.Flags().String("prometheus_listen_address", ":8888", "")
		cmd.Flags().String("rabbitmq_address", "amqp://init:init@localhost:5672", "")
		cmd.Flags().String("ssl_privkey_base64", "init_privkey", "")
		cmd.Flags().String("ssl_cert_base64", "init_cert", "")

		// Should not panic
		InitConfig(cmd)

		// After InitConfig, Get() should work
		c := Get()
		if c == nil {
			t.Fatal("Get() returned nil after InitConfig")
		}

		// Verify all fields were set
		if c.DatabaseDSN == "" {
			t.Error("DatabaseDSN was not set")
		}
		if c.PrometheusEndpoint == "" {
			t.Error("PrometheusEndpoint was not set")
		}
		if c.PrometheusListenAddress == "" {
			t.Error("PrometheusListenAddress was not set")
		}
		if c.RabbitMQAddress == "" {
			t.Error("RabbitMQAddress was not set")
		}
	})
}

func TestLoadGlobalConfig(t *testing.T) {
	// This test verifies LoadGlobalConfig can be called
	// Note: Since InitConfig was called first, sync.Once will prevent re-initialization
	// We just verify it doesn't panic
	t.Run("load global config does not panic", func(t *testing.T) {
		viper.Reset()
		viper.Set("database_dsn", "test:test@tcp(localhost:3306)/testdb")
		viper.Set("prometheus_endpoint", "/test-metrics")
		viper.Set("prometheus_listen_address", ":9999")
		viper.Set("rabbitmq_address", "amqp://test:test@localhost:5672")
		viper.Set("ssl_privkey_base64", "test_privkey")
		viper.Set("ssl_cert_base64", "test_cert")

		// Should not panic (but won't reinitialize due to sync.Once)
		LoadGlobalConfig()

		// After LoadGlobalConfig, Get() should still work
		c := Get()
		if c == nil {
			t.Error("Get() returned nil after LoadGlobalConfig")
		}
	})
}

func TestGet(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func()
		wantPanic bool
	}{
		{
			name: "get initialized config",
			setupFunc: func() {
				viper.Reset()
				cfg = &Config{
					DatabaseDSN: "test_dsn",
				}
			},
			wantPanic: false,
		},
		{
			name: "get uninitialized config panics",
			setupFunc: func() {
				cfg = nil
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("Get() panic = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			c := Get()
			if !tt.wantPanic && c == nil {
				t.Error("Get() returned nil config")
			}
		})
	}
}
