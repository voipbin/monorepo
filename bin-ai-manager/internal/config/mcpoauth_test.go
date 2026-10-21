package config

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Test_McpOAuthAsana_envToConfigField pins the path from the Asana env names
// to the Config fields. A wrong viper key in LoadGlobalConfig or a missing
// field would leave the credentials silently empty. (The env name binding
// itself is not pinned: viper.AutomaticEnv resolves the same names.)
func Test_McpOAuthAsana_envToConfigField(t *testing.T) {
	viper.Reset()
	prevConfig := globalConfig
	t.Cleanup(func() {
		viper.Reset()
		globalConfig = prevConfig
		once = sync.Once{}
	})
	cmd := &cobra.Command{}
	if err := bindConfig(cmd); err != nil {
		t.Fatalf("bind failed: %v", err)
	}

	t.Setenv("MCP_OAUTH_ASANA_CLIENT_ID", "asana-id-x")
	t.Setenv("MCP_OAUTH_ASANA_CLIENT_SECRET", "asana-secret-x")

	once = sync.Once{}
	LoadGlobalConfig()
	cfg := Get()

	if cfg.McpOAuthAsanaClientID != "asana-id-x" {
		t.Errorf("McpOAuthAsanaClientID: got %q", cfg.McpOAuthAsanaClientID)
	}
	if cfg.McpOAuthAsanaClientSecret != "asana-secret-x" {
		t.Errorf("McpOAuthAsanaClientSecret: got %q", cfg.McpOAuthAsanaClientSecret)
	}
}
