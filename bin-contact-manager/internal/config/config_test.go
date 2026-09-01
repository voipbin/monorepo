package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set some test values
	viper.Set("database_dsn", "test_db_dsn")
	viper.Set("prometheus_endpoint", "/test-metrics")
	viper.Set("prometheus_listen_address", ":9999")
	viper.Set("rabbitmq_address", "amqp://test:test@testhost:5672")
	viper.Set("redis_address", "testhost:6379")
	viper.Set("redis_database", "5")
	viper.Set("redis_password", "testpass")

	LoadGlobalConfig()

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	if cfg.DatabaseDSN != "test_db_dsn" {
		t.Errorf("DatabaseDSN = %v, want test_db_dsn", cfg.DatabaseDSN)
	}

	if cfg.PrometheusEndpoint != "/test-metrics" {
		t.Errorf("PrometheusEndpoint = %v, want /test-metrics", cfg.PrometheusEndpoint)
	}

	if cfg.PrometheusListenAddress != ":9999" {
		t.Errorf("PrometheusListenAddress = %v, want :9999", cfg.PrometheusListenAddress)
	}

	if cfg.RabbitMQAddress != "amqp://test:test@testhost:5672" {
		t.Errorf("RabbitMQAddress = %v, want amqp://test:test@testhost:5672", cfg.RabbitMQAddress)
	}

	if cfg.RedisAddress != "testhost:6379" {
		t.Errorf("RedisAddress = %v, want testhost:6379", cfg.RedisAddress)
	}

	if cfg.RedisDatabase != 5 {
		t.Errorf("RedisDatabase = %v, want 5", cfg.RedisDatabase)
	}

	if cfg.RedisPassword != "testpass" {
		t.Errorf("RedisPassword = %v, want testpass", cfg.RedisPassword)
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful bootstrap",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()

			// Create a test cobra command
			cmd := &cobra.Command{
				Use: "test",
			}

			err := Bootstrap(cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify flags were created
			if cmd.PersistentFlags().Lookup("database_dsn") == nil {
				t.Error("database_dsn flag not created")
			}
			if cmd.PersistentFlags().Lookup("prometheus_endpoint") == nil {
				t.Error("prometheus_endpoint flag not created")
			}
			if cmd.PersistentFlags().Lookup("prometheus_listen_address") == nil {
				t.Error("prometheus_listen_address flag not created")
			}
			if cmd.PersistentFlags().Lookup("rabbitmq_address") == nil {
				t.Error("rabbitmq_address flag not created")
			}
			if cmd.PersistentFlags().Lookup("redis_address") == nil {
				t.Error("redis_address flag not created")
			}
			if cmd.PersistentFlags().Lookup("redis_database") == nil {
				t.Error("redis_database flag not created")
			}
			if cmd.PersistentFlags().Lookup("redis_password") == nil {
				t.Error("redis_password flag not created")
			}
		})
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	tests := []struct {
		name string
		setup func()
		check func(*testing.T, *Config)
	}{
		{
			name: "load with default values",
			setup: func() {
				viper.Reset()
				viper.SetDefault("database_dsn", "default_dsn")
				viper.SetDefault("prometheus_endpoint", "/metrics")
				viper.SetDefault("prometheus_listen_address", ":2112")
				viper.SetDefault("rabbitmq_address", "amqp://guest:guest@localhost:5672")
				viper.SetDefault("redis_address", "127.0.0.1:6379")
				viper.SetDefault("redis_database", 1)
				viper.SetDefault("redis_password", "")
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.DatabaseDSN != "default_dsn" {
					t.Errorf("DatabaseDSN = %v, want default_dsn", cfg.DatabaseDSN)
				}
			},
		},
		{
			name: "load with custom values",
			setup: func() {
				viper.Reset()
				viper.Set("database_dsn", "custom_dsn")
				viper.Set("prometheus_endpoint", "/custom")
				viper.Set("prometheus_listen_address", ":8080")
				viper.Set("rabbitmq_address", "amqp://custom")
				viper.Set("redis_address", "custom:6379")
				viper.Set("redis_database", 10)
				viper.Set("redis_password", "custompass")
			},
			check: func(t *testing.T, cfg *Config) {
				if cfg.DatabaseDSN != "custom_dsn" {
					t.Errorf("DatabaseDSN = %v, want custom_dsn", cfg.DatabaseDSN)
				}
				if cfg.PrometheusEndpoint != "/custom" {
					t.Errorf("PrometheusEndpoint = %v, want /custom", cfg.PrometheusEndpoint)
				}
				if cfg.PrometheusListenAddress != ":8080" {
					t.Errorf("PrometheusListenAddress = %v, want :8080", cfg.PrometheusListenAddress)
				}
				if cfg.RabbitMQAddress != "amqp://custom" {
					t.Errorf("RabbitMQAddress = %v, want amqp://custom", cfg.RabbitMQAddress)
				}
				if cfg.RedisAddress != "custom:6379" {
					t.Errorf("RedisAddress = %v, want custom:6379", cfg.RedisAddress)
				}
				if cfg.RedisDatabase != 10 {
					t.Errorf("RedisDatabase = %v, want 10", cfg.RedisDatabase)
				}
				if cfg.RedisPassword != "custompass" {
					t.Errorf("RedisPassword = %v, want custompass", cfg.RedisPassword)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the sync.Once to allow multiple calls to LoadGlobalConfig
			once = sync.Once{}

			tt.setup()
			LoadGlobalConfig()
			cfg := Get()
			tt.check(t, cfg)
		})
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *cobra.Command
		wantErr bool
	}{
		{
			name: "successful init",
			setup: func() *cobra.Command {
				viper.Reset()
				cmd := &cobra.Command{Use: "test"}
				cmd.Flags().String("database_dsn", "test_dsn", "")
				cmd.Flags().String("prometheus_endpoint", "/metrics", "")
				cmd.Flags().String("prometheus_listen_address", ":2112", "")
				cmd.Flags().String("rabbitmq_address", "amqp://test", "")
				cmd.Flags().String("redis_address", "localhost:6379", "")
				cmd.Flags().String("redis_database", "1", "")
				cmd.Flags().String("redis_password", "", "")
				return cmd
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setup()
			err := InitConfig(cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				cfg := Get()
				if cfg == nil {
					t.Fatal("Get() returned nil after InitConfig")
				}
			}
		})
	}
}

func TestInitConfig_BindErrors(t *testing.T) {
	tests := []struct {
		name       string
		missingFlag string
		wantErr    bool
	}{
		{
			name:       "missing database_dsn",
			missingFlag: "database_dsn",
			wantErr:    true,
		},
		{
			name:       "missing prometheus_endpoint",
			missingFlag: "prometheus_endpoint",
			wantErr:    true,
		},
		{
			name:       "missing prometheus_listen_address",
			missingFlag: "prometheus_listen_address",
			wantErr:    true,
		},
		{
			name:       "missing rabbitmq_address",
			missingFlag: "rabbitmq_address",
			wantErr:    true,
		},
		{
			name:       "missing redis_address",
			missingFlag: "redis_address",
			wantErr:    true,
		},
		{
			name:       "missing redis_database",
			missingFlag: "redis_database",
			wantErr:    true,
		},
		{
			name:       "missing redis_password",
			missingFlag: "redis_password",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			cmd := &cobra.Command{Use: "test"}

			// Add all flags except the missing one
			allFlags := []string{
				"database_dsn",
				"prometheus_endpoint",
				"prometheus_listen_address",
				"rabbitmq_address",
				"redis_address",
				"redis_database",
				"redis_password",
			}

			for _, flag := range allFlags {
				if flag != tt.missingFlag {
					cmd.Flags().String(flag, "", "")
				}
			}

			err := InitConfig(cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
