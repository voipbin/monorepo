package main

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"monorepo/bin-ai-manager/internal/configkeys"
)

// composeEnvironment returns the environment of the ai-manager service of the compose file as a
// map of variable name to value. The file is parsed, so a changed layout can not hide a wrong line.
func composeEnvironment(t *testing.T) map[string]string {
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

	svc, ok := doc.Services["ai-manager"]
	if !ok {
		t.Fatalf("Wrong match. the ai-manager service is missing")
	}

	res := map[string]string{}
	for _, item := range svc.Environment {
		name, value, found := strings.Cut(item, "=")
		if !found {
			t.Fatalf("Wrong match. expect: NAME=value, got: %s", item)
		}
		if _, dup := res[name]; dup {
			t.Errorf("Wrong match. %s is set twice", name)
		}
		res[name] = value
	}

	return res
}

// k8sEnvEntry is one entry of the env list of a container.
type k8sEnvEntry struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom *struct {
		SecretKeyRef *struct {
			Name     string `yaml:"name"`
			Key      string `yaml:"key"`
			Optional *bool  `yaml:"optional"`
		} `yaml:"secretKeyRef"`
	} `yaml:"valueFrom"`
}

func k8sEnvironment(t *testing.T) map[string]k8sEnvEntry {
	t.Helper()

	raw, err := os.ReadFile("../../k8s/deployment.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var doc struct {
		Spec struct {
			Template struct {
				Spec struct {
					Containers []struct {
						Name string        `yaml:"name"`
						Env  []k8sEnvEntry `yaml:"env"`
					} `yaml:"containers"`
				} `yaml:"spec"`
			} `yaml:"template"`
		} `yaml:"spec"`
	}
	if errParse := yaml.Unmarshal(raw, &doc); errParse != nil {
		t.Fatalf("Wrong match. expect: valid yaml, got: %v", errParse)
	}

	for _, c := range doc.Spec.Template.Spec.Containers {
		if c.Name != "ai-manager" {
			continue
		}

		res := map[string]k8sEnvEntry{}
		for _, e := range c.Env {
			if _, dup := res[e.Name]; dup {
				t.Errorf("Wrong match. %s is set twice", e.Name)
			}
			res[e.Name] = e
		}

		return res
	}

	t.Fatalf("Wrong match. the ai-manager container is missing")

	return nil
}

// Every variable of the platform keys is in the compose file with the Komodo variable of the same
// name, so a missing deployment variable stays visible as a placeholder.
func Test_komodoComposeHasThePlatformKeyVariables(t *testing.T) {
	env := composeEnvironment(t)

	names := configkeys.EnvNames()
	if len(names) != 10 {
		t.Fatalf("Wrong match. expect: 10 variables, got: %v", names)
	}
	for _, name := range names {
		want := "[[BIN_MANAGER__" + name + "]]"
		if got, ok := env[name]; !ok || got != want {
			t.Errorf("Wrong match. %s expect: %s, got: %q (present %v)", name, want, got, ok)
		}
	}
}

// The customer owned master keys and the other settings stay as they are.
func Test_komodoComposeKeepsTheOtherLines(t *testing.T) {
	env := composeEnvironment(t)

	expect := map[string]string{
		"MCP_SECRET_ENCRYPTION_KEYS":      "[[BIN_MANAGER__MCP_SECRET_ENCRYPTION_KEYS]]",
		"ENGINE_BASE_KEY_ENCRYPTION_KEYS": "[[BIN_MANAGER__AI_MANAGER_ENGINE_BASE_KEY_ENCRYPTION_KEYS]]",
		"LOCAL_LLM_MODELS":                "[[BIN_MANAGER__LOCAL_LLM_MODELS]]",
	}
	for name, want := range expect {
		if got := env[name]; got != want {
			t.Errorf("Wrong match. %s expect: %s, got: %q", name, want, got)
		}
	}

	// the old names are not passed any more
	for _, name := range []string{"ENGINE_KEY_CHATGPT", "MCP_OAUTH_GITHUB_CLIENT_ID", "MCP_OAUTH_GITHUB_CLIENT_SECRET"} {
		if _, ok := env[name]; ok {
			t.Errorf("Wrong match. %s must not be passed", name)
		}
	}
}

// Kubernetes passes the two LLM keys from the secret of the deployment, always optional, so that a
// missing key does not stop the pod. The MCP OAuth apps are not passed in Kubernetes.
func Test_k8sHasThePlatformKeyVariables(t *testing.T) {
	env := k8sEnvironment(t)

	for _, name := range []string{configkeys.EnvOpenAIAPIKey, configkeys.EnvGoogleAPIKey} {
		e, ok := env[name]
		if !ok {
			t.Errorf("Wrong match. %s is missing", name)
			continue
		}
		if e.Value != "" || e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
			t.Errorf("Wrong match. %s expect: a secretKeyRef, got: %+v", name, e)
			continue
		}
		ref := e.ValueFrom.SecretKeyRef
		if ref.Name != "voipbin" || ref.Key != name || ref.Optional == nil || !*ref.Optional {
			t.Errorf("Wrong match. %s expect: secret voipbin, key %s, optional true, got: %+v", name, name, ref)
		}
	}

	for name := range env {
		if strings.Contains(name, "MCP_OAUTH") || name == "ENGINE_KEY_CHATGPT" {
			t.Errorf("Wrong match. %s must not be passed in Kubernetes", name)
		}
	}
}
