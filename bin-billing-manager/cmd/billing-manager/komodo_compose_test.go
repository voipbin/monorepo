package main

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"monorepo/bin-billing-manager/internal/configkeys"
)

// composeEnvironment returns the environment of the billing service of the compose file as a map
// of variable name to value. The file is parsed, so a changed layout can not hide a wrong line.
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

	svc, ok := doc.Services["billing-manager"]
	if !ok {
		t.Fatalf("Wrong match. the billing-manager service is missing")
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
	if len(doc.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("Wrong match. expect: one container, got: %d", len(doc.Spec.Template.Spec.Containers))
	}

	res := map[string]k8sEnvEntry{}
	for _, e := range doc.Spec.Template.Spec.Containers[0].Env {
		res[e.Name] = e
	}

	return res
}

// The Paddle credentials are passed under the name that the service reads, and the deployment
// variable has the same name.
func Test_komodoComposePassesThePlatformSecrets(t *testing.T) {
	env := composeEnvironment(t)

	for _, name := range []string{configkeys.EnvPaddleAPIKey, configkeys.EnvPaddleWebhookSecret} {
		if got, want := env[name], "[[BIN_MANAGER__"+name+"]]"; got != want {
			t.Errorf("Wrong match. %s expect: %q, got: %q", name, want, got)
		}
	}
}

// The Kubernetes definition passes the Paddle credentials as optional
// secret references: a cluster whose secret has no such key still starts the pod (the key is then
// not configured).
func Test_k8sDeploymentPassesTheSecretsAsOptionalReferences(t *testing.T) {
	env := k8sEnvironment(t)

	for _, name := range []string{configkeys.EnvPaddleAPIKey, configkeys.EnvPaddleWebhookSecret} {
		e, ok := env[name]
		if !ok {
			t.Errorf("Wrong match. %s is missing", name)
			continue
		}
		if e.Value != "" || e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
			t.Errorf("Wrong match. %s expect: a secret reference, got: %+v", name, e)
			continue
		}

		ref := e.ValueFrom.SecretKeyRef
		if ref.Name != "voipbin" || ref.Key != name || ref.Optional == nil || !*ref.Optional {
			t.Errorf("Wrong match. %s expect: voipbin/%s optional, got: %+v", name, name, ref)
		}
	}
}
