package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"monorepo/bin-tts-manager/internal/configkeys"
)

// envEntry is one item of an environment list of a deployment definition.
type envEntry struct {
	Name      string `yaml:"name"`
	Value     string `yaml:"value"`
	ValueFrom struct {
		SecretKeyRef struct {
			Name     string `yaml:"name"`
			Key      string `yaml:"key"`
			Optional bool   `yaml:"optional"`
		} `yaml:"secretKeyRef"`
	} `yaml:"valueFrom"`
}

// composeEnvironment returns the environment of the service as a name to value map.
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
		t.Fatalf("Wrong match. expect: ok, got: %v", errParse)
	}

	svc, ok := doc.Services["tts-manager"]
	if !ok {
		t.Fatalf("Wrong match. expect: the service message-manager")
	}

	res := map[string]string{}
	for _, item := range svc.Environment {
		name, value, _ := strings.Cut(item, "=")
		res[name] = value
	}

	return res
}

// k8sEnvironment returns the environment entries of the container of the deployment.
func k8sEnvironment(t *testing.T) map[string]envEntry {
	t.Helper()

	raw, err := os.ReadFile("../../k8s/deployment.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(raw))
	for {
		var doc struct {
			Kind string `yaml:"kind"`
			Spec struct {
				Template struct {
					Spec struct {
						Containers []struct {
							Env []envEntry `yaml:"env"`
						} `yaml:"containers"`
					} `yaml:"spec"`
				} `yaml:"template"`
			} `yaml:"spec"`
		}
		if errDecode := dec.Decode(&doc); errDecode != nil {
			if errors.Is(errDecode, io.EOF) {
				break
			}
			t.Fatalf("Wrong match. expect: ok, got: %v", errDecode)
		}
		if doc.Kind != "Deployment" || len(doc.Spec.Template.Spec.Containers) == 0 {
			continue
		}

		res := map[string]envEntry{}
		for _, e := range doc.Spec.Template.Spec.Containers[0].Env {
			res[e.Name] = e
		}

		return res
	}

	t.Fatalf("Wrong match. expect: a Deployment")
	return nil
}

// The provider keys come from the environment: each variable is in the compose definition with the
// deployment variable of the same name, and in k8s as an optional key of the secret.
func Test_deploymentPassesTheProviderKeys(t *testing.T) {
	names := []string{configkeys.EnvAWSAccessKey, configkeys.EnvAWSSecretKey, configkeys.EnvElevenlabsAPIKey}

	compose := composeEnvironment(t)
	k8s := k8sEnvironment(t)
	for _, name := range names {
		if got, ok := compose[name]; !ok || got != "[[BIN_MANAGER__"+name+"]]" {
			t.Errorf("Wrong match. compose %s expect: [[BIN_MANAGER__%s]], got: %q %v", name, name, got, ok)
		}

		e, ok := k8s[name]
		if !ok {
			t.Errorf("Wrong match. k8s %s is missing", name)
			continue
		}
		ref := e.ValueFrom.SecretKeyRef
		if ref.Name != "voipbin" || ref.Key != name || !ref.Optional || e.Value != "" {
			t.Errorf("Wrong match. k8s %s expect: secret voipbin key %s optional, got: %+v", name, name, e)
		}
	}
}
