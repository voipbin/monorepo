package config

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}

func TestBootstrap(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful_bootstrap",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper for each test
			viper.Reset()

			cmd := &cobra.Command{}
			err := Bootstrap(cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBindConfig(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful_bind",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			cmd := &cobra.Command{}
			err := bindConfig(cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("bindConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitLog(t *testing.T) {
	// This just ensures initLog doesn't panic
	initLog()
}
