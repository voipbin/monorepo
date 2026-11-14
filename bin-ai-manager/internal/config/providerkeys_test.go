package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The platform LLM keys are read from the environment by internal/aiconfig when a call needs them,
// so Config has no field for them.
func Test_Config_hasNoProviderKeyFields(t *testing.T) {
	typ := reflect.TypeOf(Config{})

	for _, name := range []string{"EngineKeyChatGPT", "GoogleAPIKey"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("Wrong match. expect: no field %s, got: a field", name)
		}
	}
	if _, ok := reflect.TypeOf(&Config{}).MethodByName("EnvValue"); ok {
		t.Errorf("Wrong match. expect: no EnvValue method, got: a method")
	}
}

// The MCP OAuth app credentials are read from their environment variables by internal/aiconfig, so
// Config has no field and no flag for them, and the load does not touch the variables.
func Test_Config_hasNoMcpOAuthFields(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for i := 0; i < typ.NumField(); i++ {
		if strings.HasPrefix(typ.Field(i).Name, "McpOAuth") {
			t.Errorf("Wrong match. expect: no McpOAuth field, got: %s", typ.Field(i).Name)
		}
	}

	t.Setenv("GITHUB_MCP_OAUTH_CLIENT_ID", "synthetic-id")
	t.Setenv("GITHUB_MCP_OAUTH_CLIENT_SECRET", "synthetic-secret")
	cmd, _ := loadForTest(t)
	for _, name := range []string{"github_mcp_oauth_client_id", "github_mcp_oauth_client_secret"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. expect: no flag %s, got: a flag", name)
		}
	}
}

func loadForTest(t *testing.T) (*cobra.Command, Config) {
	t.Helper()

	viper.Reset()
	ResetForTest()
	t.Cleanup(func() {
		viper.Reset()
		ResetForTest()
	})

	cmd := &cobra.Command{}
	if err := Bootstrap(cmd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	LoadGlobalConfig()

	return cmd, *Get()
}

// The variables of the platform keys are not part of the loaded configuration: the load succeeds,
// the names are not flags, and the master key and every other field are the same as without them.
// The old name of the OpenAI key is not read by anything either.
func Test_LoadGlobalConfig_platformKeyVariablesAreNotConfig(t *testing.T) {
	t.Setenv(EnvConfigEncryptionKeys, "1:prefixed-value")
	t.Setenv("DATABASE_DSN", "user:pass@tcp(127.0.0.1:3306)/db")

	_, without := loadForTest(t)

	t.Setenv("OPENAI_API_KEY", "synthetic-env-openai")
	t.Setenv("ENGINE_KEY_CHATGPT", "synthetic-old-env-openai")
	t.Setenv("GOOGLE_API_KEY", "synthetic-env-gemini")

	cmd, with := loadForTest(t)

	if with.ConfigEncryptionKeys != "1:prefixed-value" || with.DatabaseDSN != "user:pass@tcp(127.0.0.1:3306)/db" {
		t.Errorf("Wrong match. expect: the master key and the DSN as set, got: %q %q", with.ConfigEncryptionKeys, with.DatabaseDSN)
	}
	if !reflect.DeepEqual(without, with) {
		t.Errorf("Wrong match. expect: the variables change no field, got: %+v and %+v", without, with)
	}
	for _, name := range []string{"engine_key_chatgpt", "openai_api_key", "google_api_key"} {
		if cmd.PersistentFlags().Lookup(name) != nil {
			t.Errorf("Wrong match. expect: no flag %s, got: a flag", name)
		}
	}
}
