package main

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// composeEnvironment returns the environment of each service of the compose file as a map of
// variable name to value. The file is parsed, so a changed layout can not hide a wrong line.
func composeEnvironment(t *testing.T) map[string]map[string]string {
	t.Helper()

	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Environment []string `yaml:"environment"`
		} `yaml:"services"`
	}
	if errParse := yaml.Unmarshal(raw, &doc); errParse != nil {
		t.Fatalf("Wrong match. expect: valid yaml, got: %v", errParse)
	}

	res := map[string]map[string]string{}
	for name, svc := range doc.Services {
		env := map[string]string{}
		for _, item := range svc.Environment {
			key, value, found := strings.Cut(item, "=")
			if !found {
				t.Fatalf("Wrong match. %s expect: NAME=value, got: %s", name, item)
			}
			if _, dup := env[key]; dup {
				t.Errorf("Wrong match. %s sets %s twice", name, key)
			}
			env[key] = value
		}
		res[name] = env
	}

	return res
}

// The runner containers read the vendor keys and the local endpoints from their environment; the
// Go containers read none of them and send no key. The service account path stays with the
// runners and the list of the local models stays with the Go services.
func Test_komodoComposeGivesPlatformKeyVariablesToTheRunnersOnly(t *testing.T) {
	envs := composeEnvironment(t)

	keyVariables := []string{
		"OPENAI_API_KEY", "XAI_API_KEY", "GOOGLE_API_KEY", "CARTESIA_API_KEY",
		"ELEVENLABS_API_KEY", "DEEPGRAM_API_KEY", "OPENROUTER_API_KEY", "LOCAL_LLM_ENDPOINTS",
	}

	for _, name := range []string{"pipecat-script-runner-1", "pipecat-script-runner-2"} {
		env, ok := envs[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		for _, v := range keyVariables {
			want := "[[BIN_MANAGER__" + v + "]]"
			if got := env[v]; got != want {
				t.Errorf("Wrong match. %s %s expect: %s, got: %s", name, v, want, got)
			}
		}
		if _, ok := env["GOOGLE_APPLICATION_CREDENTIALS"]; !ok {
			t.Errorf("Wrong match. %s expect: GOOGLE_APPLICATION_CREDENTIALS", name)
		}
	}

	for _, name := range []string{"pipecat-manager-1", "pipecat-manager-2"} {
		env, ok := envs[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		for _, v := range keyVariables {
			if _, found := env[v]; found {
				t.Errorf("Wrong match. %s must not get %s", name, v)
			}
		}
		if _, ok := env["LOCAL_LLM_MODELS"]; !ok {
			t.Errorf("Wrong match. %s expect: LOCAL_LLM_MODELS", name)
		}
	}
}
