package aiconfig

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	testGeminiURL = "https://generativelanguage.googleapis.com/v1beta/openai/"
	testOpenAIURL = "https://api.openai.com/v1"
)

// clearKeyEnv makes the platform key variables empty, so a variable of the machine does not leak
// into a test.
func clearKeyEnv(t *testing.T) {
	t.Helper()

	t.Setenv(configkeys.EnvOpenAIAPIKey, "")
	t.Setenv(configkeys.EnvGoogleAPIKey, "")
	for _, vendor := range configkeys.MCPOAuthVendors() {
		idName, secretName, _ := configkeys.MCPOAuthEnvNames(vendor)
		t.Setenv(idName, "")
		t.Setenv(secretName, "")
	}
}

func Test_Supplier(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool

		expect string
	}{
		{"not set", "", false, ""},
		{"empty", "", true, ""},
		{"spaces only", "   ", true, ""},
		{"a value", "synthetic-key", true, "synthetic-key"},
		{"a value with spaces around", "  synthetic-key \n", true, "synthetic-key"},
		{"placeholder", "[[BIN_MANAGER__OPENAI_API_KEY]]", true, ""},
		{"placeholder with spaces before", "  [[BIN_MANAGER__OPENAI_API_KEY]]", true, ""},
		{"brackets in the middle are a value", "abc[[def", true, "abc[[def"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearKeyEnv(t)
			if tt.set {
				t.Setenv(configkeys.EnvOpenAIAPIKey, tt.value)
			}

			got, err := Supplier(configkeys.EnvOpenAIAPIKey)(context.Background())
			if err != nil || got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q %v", tt.expect, got, err)
			}
		})
	}
}

// The variable is read at the time of use: nothing is cached.
func Test_Supplier_readsAtTheTimeOfUse(t *testing.T) {
	clearKeyEnv(t)
	f := Supplier(configkeys.EnvGoogleAPIKey)

	if got, err := f(context.Background()); err != nil || got != "" {
		t.Errorf("Wrong match. expect: empty, got: %q %v", got, err)
	}
	t.Setenv(configkeys.EnvGoogleAPIKey, "synthetic-one")
	if got, err := f(context.Background()); err != nil || got != "synthetic-one" {
		t.Errorf("Wrong match. expect: synthetic-one, got: %q %v", got, err)
	}
	t.Setenv(configkeys.EnvGoogleAPIKey, "synthetic-two")
	if got, err := f(context.Background()); err != nil || got != "synthetic-two" {
		t.Errorf("Wrong match. expect: synthetic-two, got: %q %v", got, err)
	}
}

// A canceled context does not matter: an environment read does not block.
func Test_Supplier_contextEnded(t *testing.T) {
	clearKeyEnv(t)
	t.Setenv(configkeys.EnvOpenAIAPIKey, "synthetic-key")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if got, err := Supplier(configkeys.EnvOpenAIAPIKey)(ctx); err != nil || got != "synthetic-key" {
		t.Errorf("Wrong match. expect: synthetic-key, got: %q %v", got, err)
	}
}

// The key of a provider follows the base URL: the Gemini key for the Gemini endpoint, the OpenAI
// key for every other URL. Both variables are set, so a wrong pick shows.
func Test_KeyForBaseURL_picksTheKeyOfTheProvider(t *testing.T) {
	clearKeyEnv(t)
	t.Setenv(configkeys.EnvOpenAIAPIKey, "openai-value")
	t.Setenv(configkeys.EnvGoogleAPIKey, "gemini-value")

	tests := []struct {
		name    string
		baseURL string

		expect string
	}{
		{"empty is the default endpoint", "", "openai-value"},
		{"openai", testOpenAIURL, "openai-value"},
		{"gemini", testGeminiURL, "gemini-value"},
		{"another compatible provider", "https://api.x.ai/v1", "openai-value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := KeyForBaseURL(tt.baseURL)(context.Background())
			if err != nil || got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q %v", tt.expect, got, err)
			}
		})
	}
}

func Test_GeminiKey(t *testing.T) {
	clearKeyEnv(t)
	t.Setenv(configkeys.EnvOpenAIAPIKey, "openai-value")

	if got, err := GeminiKey()(context.Background()); err != nil || got != "" {
		t.Errorf("Wrong match. expect: empty (the OpenAI variable is not the Gemini key), got: %q %v", got, err)
	}

	t.Setenv(configkeys.EnvGoogleAPIKey, "gemini-value")
	if got, err := GeminiKey()(context.Background()); err != nil || got != "gemini-value" {
		t.Errorf("Wrong match. expect: gemini-value, got: %q %v", got, err)
	}
}

func Test_Configured(t *testing.T) {
	clearKeyEnv(t)
	f := Supplier(configkeys.EnvOpenAIAPIKey)

	if ok, err := Configured(context.Background(), f); err != nil || ok {
		t.Errorf("Wrong match. expect: not configured, got: %v %v", ok, err)
	}

	t.Setenv(configkeys.EnvOpenAIAPIKey, "[[BIN_MANAGER__OPENAI_API_KEY]]")
	if ok, err := Configured(context.Background(), f); err != nil || ok {
		t.Errorf("Wrong match. expect: a placeholder is not configured, got: %v %v", ok, err)
	}

	t.Setenv(configkeys.EnvOpenAIAPIKey, "synthetic-key")
	if ok, err := Configured(context.Background(), f); err != nil || !ok {
		t.Errorf("Wrong match. expect: configured, got: %v %v", ok, err)
	}

	boom := errors.New("boom")
	if ok, err := Configured(context.Background(), func(context.Context) (string, error) { return "", boom }); !errors.Is(err, boom) || ok {
		t.Errorf("Wrong match. expect: the error, got: %v %v", ok, err)
	}
}

// LogStates: a variable that is not set or is a placeholder is a warning, a set one is info, and
// no line holds a value.
func Test_LogStates(t *testing.T) {
	const secret = "synthetic-secret-text"

	clearKeyEnv(t)
	t.Setenv(configkeys.EnvOpenAIAPIKey, secret)
	t.Setenv(configkeys.EnvGoogleAPIKey, "[[BIN_MANAGER__GOOGLE_API_KEY]]")
	t.Setenv("GITHUB_MCP_OAUTH_CLIENT_ID", "synthetic-client-id-text")

	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))

	LogStates()

	levels := map[string]logrus.Level{}
	msgs := map[string]string{}
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, secret) || strings.Contains(e.Message, "synthetic-client-id-text") || strings.Contains(strings.Join(fieldValues(e.Data), " "), secret) {
			t.Errorf("Wrong match. a value was logged: %s %v", e.Message, e.Data)
		}
		name, _ := e.Data["env"].(string)
		if name == "" {
			t.Errorf("Wrong match. expect: the variable name in the field env, got: %v %s", e.Data, e.Message)
			continue
		}
		levels[name] = e.Level
		msgs[name] = e.Message
	}

	if len(levels) != 10 {
		t.Errorf("Wrong match. expect: a line for each of the 10 variables, got: %v", levels)
	}
	if levels[configkeys.EnvOpenAIAPIKey] != logrus.InfoLevel {
		t.Errorf("Wrong match. expect: info for a set variable, got: %v", levels[configkeys.EnvOpenAIAPIKey])
	}
	if levels[configkeys.EnvGoogleAPIKey] != logrus.WarnLevel || !strings.Contains(msgs[configkeys.EnvGoogleAPIKey], "unresolved placeholder") || !strings.Contains(msgs[configkeys.EnvGoogleAPIKey], configkeys.EnvGoogleAPIKey) {
		t.Errorf("Wrong match. expect: a warning with the name and unresolved placeholder, got: %v %q", levels[configkeys.EnvGoogleAPIKey], msgs[configkeys.EnvGoogleAPIKey])
	}
	if levels["LINEAR_MCP_OAUTH_CLIENT_SECRET"] != logrus.WarnLevel || !strings.Contains(msgs["LINEAR_MCP_OAUTH_CLIENT_SECRET"], "not set") {
		t.Errorf("Wrong match. expect: a warning for a variable that is not set, got: %v %q", levels["LINEAR_MCP_OAUTH_CLIENT_SECRET"], msgs["LINEAR_MCP_OAUTH_CLIENT_SECRET"])
	}
}

func fieldValues(data logrus.Fields) []string {
	out := []string{}
	for _, v := range data {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}

	return out
}
